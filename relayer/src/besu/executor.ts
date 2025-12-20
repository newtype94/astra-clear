import { ethers, Contract, Wallet } from 'ethers';
import { MintCommand, ECDSASignature } from '../types';
import { Logger } from 'winston';

/**
 * BesuExecutor handles executing MintCommands on Besu blockchain
 * Requirement 6.4: Besu 명령 실행 구현
 */
export class BesuExecutor {
  private provider: ethers.JsonRpcProvider;
  private wallet: Wallet;
  private executor: Contract;
  private logger: Logger;

  // Executor contract ABI (matches Executor.sol contract)
  private static readonly EXECUTOR_ABI = [
    'function executeMint(bytes32 commandId, address recipient, uint256 amount, bytes[] calldata signatures) external',
    'function isCommandProcessed(bytes32 commandId) external view returns (bool)',
    'function getMessageHash(bytes32 commandId, address recipient, uint256 amount) external view returns (bytes32)',
    'function threshold() external view returns (uint256)',
    'function getValidatorCount() external view returns (uint256)',
    'event MintExecuted(bytes32 indexed commandId, address indexed recipient, uint256 amount, uint256 timestamp)',
    'event MintRejected(bytes32 indexed commandId, string reason)',
  ];

  constructor(
    rpcUrl: string,
    executorAddress: string,
    privateKey: string,
    logger: Logger
  ) {
    this.provider = new ethers.JsonRpcProvider(rpcUrl);
    this.wallet = new Wallet(privateKey, this.provider);
    this.executor = new Contract(
      executorAddress,
      BesuExecutor.EXECUTOR_ABI,
      this.wallet
    );
    this.logger = logger;
  }

  /**
   * Execute a mint command on Besu
   * This implements the cross-chain mint execution after consensus (Requirement 2.4)
   */
  async executeMintCommand(command: MintCommand): Promise<string> {
    this.logger.info('Executing mint command on Besu', {
      commandId: command.commandId,
      recipient: command.recipient,
      amount: command.amount.toString(),
      targetChain: command.targetChain,
    });

    try {
      // Check if command already executed
      const isExecuted = await this.isCommandExecuted(command.commandId);
      if (isExecuted) {
        this.logger.warn('Command already executed', {
          commandId: command.commandId,
        });
        return ''; // Return empty string to indicate already executed
      }

      // Validate signatures before sending transaction
      if (command.signatures.length === 0) {
        throw new Error('No signatures provided for mint command');
      }

      // Convert commandId to bytes32
      const commandIdBytes = ethers.id(command.commandId);

      // Convert signatures to bytes array format expected by contract
      const signaturesBytes = this.encodeSignatures(command.signatures);

      // Estimate gas
      const gasEstimate = await this.executor.executeMint.estimateGas(
        commandIdBytes,
        command.recipient,
        command.amount,
        signaturesBytes
      );

      // Add 20% buffer to gas estimate
      const gasLimit = Math.ceil(Number(gasEstimate) * 1.2);

      // Get current gas price
      const feeData = await this.provider.getFeeData();
      const gasPrice = feeData.gasPrice || ethers.parseUnits('20', 'gwei');

      // Execute the mint command
      const tx = await this.executor.executeMint(
        commandIdBytes,
        command.recipient,
        command.amount,
        signaturesBytes,
        {
          gasLimit,
          gasPrice,
        }
      );

      this.logger.info('Mint command transaction sent', {
        commandId: command.commandId,
        txHash: tx.hash,
        gasLimit,
        gasPrice: gasPrice.toString(),
      });

      // Wait for transaction confirmation
      const receipt = await tx.wait();

      if (receipt.status === 0) {
        throw new Error('Transaction reverted');
      }

      this.logger.info('Mint command executed successfully', {
        commandId: command.commandId,
        txHash: receipt.hash,
        blockNumber: receipt.blockNumber,
        gasUsed: receipt.gasUsed.toString(),
      });

      return receipt.hash;
    } catch (error) {
      this.logger.error('Failed to execute mint command', {
        commandId: command.commandId,
        error: error instanceof Error ? error.message : String(error),
      });
      throw error;
    }
  }

  /**
   * Check if a command has already been executed
   */
  async isCommandExecuted(commandId: string): Promise<boolean> {
    try {
      const commandIdBytes = ethers.id(commandId);
      const isExecuted = await this.executor.isCommandProcessed(commandIdBytes);
      return isExecuted;
    } catch (error) {
      this.logger.error('Failed to check command execution status', {
        commandId,
        error: error instanceof Error ? error.message : String(error),
      });
      return false;
    }
  }

  /**
   * Encode ECDSA signatures into bytes array format for Solidity
   * Each signature is encoded as: [v (1 byte)][r (32 bytes)][s (32 bytes)]
   */
  private encodeSignatures(signatures: ECDSASignature[]): string[] {
    return signatures.map((sig) => {
      // Ensure r and s are 32 bytes (64 hex chars without 0x)
      const r = sig.r.replace('0x', '').padStart(64, '0');
      const s = sig.s.replace('0x', '').padStart(64, '0');

      // v is recovery id (27 or 28 for Ethereum, or 0/1)
      // Normalize to 27/28 if it's 0/1
      let v = sig.v;
      if (v < 27) {
        v += 27;
      }

      // Concatenate: v (2 hex chars) + r (64 hex chars) + s (64 hex chars)
      const vHex = v.toString(16).padStart(2, '0');
      const signature = `0x${r}${s}${vHex}`;

      return signature;
    });
  }

  /**
   * Verify signatures against a message hash
   * This is useful for pre-validation before sending to contract
   */
  async verifySignatures(
    messageHash: string,
    signatures: ECDSASignature[],
    expectedSigners: string[]
  ): Promise<boolean> {
    try {
      const recoveredSigners: string[] = [];

      for (const sig of signatures) {
        // Reconstruct the signature
        const r = sig.r;
        const s = sig.s;
        let v = sig.v;
        if (v < 27) {
          v += 27;
        }

        // Recover signer address
        const signature = ethers.Signature.from({
          r,
          s,
          v,
        });

        const recovered = ethers.recoverAddress(messageHash, signature);
        recoveredSigners.push(recovered.toLowerCase());
      }

      // Check if all recovered signers are in expected signers list
      for (const signer of recoveredSigners) {
        if (
          !expectedSigners.some((expected) => expected.toLowerCase() === signer)
        ) {
          this.logger.warn('Unexpected signer found', {
            signer,
            expectedSigners,
          });
          return false;
        }
      }

      // Check if we have enough signatures (2/3+ threshold)
      const threshold = Math.ceil((expectedSigners.length * 2) / 3);
      if (recoveredSigners.length < threshold) {
        this.logger.warn('Insufficient signatures', {
          received: recoveredSigners.length,
          required: threshold,
        });
        return false;
      }

      return true;
    } catch (error) {
      this.logger.error('Signature verification failed', {
        error: error instanceof Error ? error.message : String(error),
      });
      return false;
    }
  }

  /**
   * Get executor contract address
   */
  getExecutorAddress(): string {
    return this.executor.target as string;
  }

  /**
   * Get wallet address
   */
  getWalletAddress(): string {
    return this.wallet.address;
  }

  /**
   * Get current block number
   */
  async getCurrentBlock(): Promise<number> {
    return await this.provider.getBlockNumber();
  }
}
