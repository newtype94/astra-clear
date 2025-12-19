import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { SigningStargateClient, GasPrice, StdFee } from '@cosmjs/stargate';
import { TransferEvent, ECDSASignature } from '../types';
import { Logger } from 'winston';

/**
 * CosmosSubmitter handles submitting MsgVote transactions to Cosmos Hub
 * Requirement 6.2: Cosmos Hub 메시지 전송 구현
 */
export class CosmosSubmitter {
  private client: SigningStargateClient | null = null;
  private wallet: DirectSecp256k1HdWallet | null = null;
  private validatorAddress: string = '';
  private logger: Logger;
  private rpcEndpoint: string;
  private mnemonic: string;
  private gasPrice: GasPrice;

  // Custom message type for oracle module
  private static readonly MSG_VOTE_TYPE = '/interbank.netting.oracle.MsgVote';

  constructor(
    rpcEndpoint: string,
    mnemonic: string,
    gasPrice: string,
    logger: Logger
  ) {
    this.rpcEndpoint = rpcEndpoint;
    this.mnemonic = mnemonic;
    this.gasPrice = GasPrice.fromString(gasPrice);
    this.logger = logger;
  }

  /**
   * Initialize the Cosmos client and wallet
   */
  async connect(): Promise<void> {
    this.logger.info('Connecting to Cosmos Hub', {
      rpcEndpoint: this.rpcEndpoint,
    });

    // Create wallet from mnemonic
    this.wallet = await DirectSecp256k1HdWallet.fromMnemonic(this.mnemonic, {
      prefix: 'cosmos',
    });

    // Get validator address
    const [firstAccount] = await this.wallet.getAccounts();
    this.validatorAddress = firstAccount.address;

    // Create signing client
    this.client = await SigningStargateClient.connectWithSigner(
      this.rpcEndpoint,
      this.wallet,
      {
        gasPrice: this.gasPrice,
      }
    );

    this.logger.info('Connected to Cosmos Hub', {
      validatorAddress: this.validatorAddress,
    });
  }

  /**
   * Submit a vote for a Besu transfer event
   * This implements the validator voting mechanism (Requirement 3.1)
   */
  async submitVote(event: TransferEvent): Promise<string> {
    if (!this.client || !this.validatorAddress) {
      throw new Error('Cosmos client not initialized. Call connect() first.');
    }

    this.logger.info('Submitting vote for transfer', {
      txHash: event.txHash,
      sender: event.sender,
      recipient: event.recipient,
      amount: event.amount.toString(),
    });

    // Create MsgVote message
    const msgVote = {
      typeUrl: CosmosSubmitter.MSG_VOTE_TYPE,
      value: {
        validator: this.validatorAddress,
        txHash: event.txHash,
        sender: event.sender,
        recipient: event.recipient,
        amount: event.amount.toString(),
        sourceChain: event.sourceChain,
        destChain: event.destChain,
        nonce: event.nonce.toString(),
        blockNumber: event.blockNumber.toString(),
        timestamp: event.timestamp,
      },
    };

    try {
      // Estimate gas
      const gasEstimate = await this.estimateGas(msgVote);

      // Calculate fee with 20% buffer
      const fee = this.calculateFee(gasEstimate);

      // Broadcast transaction
      const result = await this.client.signAndBroadcast(
        this.validatorAddress,
        [msgVote],
        fee,
        `Vote for transfer ${event.txHash}`
      );

      if (result.code !== 0) {
        throw new Error(
          `Transaction failed: ${result.rawLog || 'Unknown error'}`
        );
      }

      this.logger.info('Vote submitted successfully', {
        txHash: event.txHash,
        cosmosTxHash: result.transactionHash,
        gasUsed: result.gasUsed,
        gasWanted: result.gasWanted,
      });

      return result.transactionHash;
    } catch (error) {
      this.logger.error('Failed to submit vote', {
        txHash: event.txHash,
        error: error instanceof Error ? error.message : String(error),
      });
      throw error;
    }
  }

  /**
   * Query the vote status for a transfer
   */
  async queryVoteStatus(txHash: string): Promise<VoteStatus | null> {
    if (!this.client) {
      throw new Error('Cosmos client not initialized. Call connect() first.');
    }

    try {
      // Query the oracle module for vote status
      const queryData = {
        vote_status: {
          tx_hash: txHash,
        },
      };

      const response = await this.client.queryContractSmart(
        'oracle', // Module query endpoint
        queryData
      );

      return {
        txHash: response.tx_hash,
        voteCount: parseInt(response.vote_count),
        threshold: parseInt(response.threshold),
        confirmed: response.confirmed,
        createdAt: parseInt(response.created_at),
      };
    } catch (error) {
      this.logger.error('Failed to query vote status', {
        txHash,
        error: error instanceof Error ? error.message : String(error),
      });
      return null;
    }
  }

  /**
   * Check if a transfer has reached consensus (2/3+ votes)
   */
  async hasReachedConsensus(txHash: string): Promise<boolean> {
    const status = await this.queryVoteStatus(txHash);
    if (!status) {
      return false;
    }
    return status.confirmed;
  }

  /**
   * Estimate gas for a message
   */
  private async estimateGas(msg: any): Promise<number> {
    if (!this.client || !this.validatorAddress) {
      throw new Error('Cosmos client not initialized');
    }

    try {
      // Simulate transaction to estimate gas
      const gasEstimate = await this.client.simulate(
        this.validatorAddress,
        [msg],
        'Gas estimation'
      );

      // Add 20% buffer for safety
      return Math.ceil(gasEstimate * 1.2);
    } catch (error) {
      this.logger.warn('Gas estimation failed, using default', {
        error: error instanceof Error ? error.message : String(error),
      });
      // Return default gas amount if estimation fails
      return 200000;
    }
  }

  /**
   * Calculate fee based on gas estimate
   */
  private calculateFee(gasEstimate: number): StdFee {
    const amount = Math.ceil(gasEstimate * parseFloat(this.gasPrice.amount));
    return {
      amount: [
        {
          denom: this.gasPrice.denom,
          amount: amount.toString(),
        },
      ],
      gas: gasEstimate.toString(),
    };
  }

  /**
   * Get current validator address
   */
  getValidatorAddress(): string {
    return this.validatorAddress;
  }

  /**
   * Disconnect from Cosmos Hub
   */
  async disconnect(): Promise<void> {
    if (this.client) {
      this.client.disconnect();
      this.client = null;
      this.wallet = null;
      this.validatorAddress = '';
      this.logger.info('Disconnected from Cosmos Hub');
    }
  }

  /**
   * Check if client is connected
   */
  isConnected(): boolean {
    return this.client !== null && this.validatorAddress !== '';
  }
}

/**
 * Vote status response from oracle module
 */
export interface VoteStatus {
  txHash: string;
  voteCount: number;
  threshold: number;
  confirmed: boolean;
  createdAt: number;
}
