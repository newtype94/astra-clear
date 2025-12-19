import { Logger } from 'winston';
import { BesuMonitor } from './besu/monitor';
import { BesuExecutor } from './besu/executor';
import { CosmosSubmitter } from './cosmos/submitter';
import { CosmosMonitor } from './cosmos/monitor';
import { RelayerConfig } from './config';
import { retry, retryBlockchain, CircuitBreaker } from './utils/retry';
import { TransferEvent, MintCommand } from './types';

/**
 * Main Relayer class that orchestrates cross-chain communication
 * between Besu and Cosmos Hub
 */
export class Relayer {
  private besuMonitor: BesuMonitor;
  private besuExecutor: BesuExecutor;
  private cosmosSubmitter: CosmosSubmitter;
  private cosmosMonitor: CosmosMonitor;
  private logger: Logger;
  private config: RelayerConfig;

  // Circuit breakers for fault tolerance
  private cosmosCircuitBreaker: CircuitBreaker;
  private besuCircuitBreaker: CircuitBreaker;

  // Tracking processed events to avoid duplicates
  private processedTransfers: Set<string> = new Set();
  private processedCommands: Set<string> = new Set();

  constructor(config: RelayerConfig, logger: Logger) {
    this.config = config;
    this.logger = logger;

    // Initialize Besu components
    this.besuMonitor = new BesuMonitor(
      config.besu.rpcUrl,
      config.besu.wsUrl,
      config.besu.gatewayAddress,
      config.besu.startBlock,
      logger
    );

    this.besuExecutor = new BesuExecutor(
      config.besu.rpcUrl,
      config.besu.executorAddress,
      config.besu.privateKey,
      logger
    );

    // Initialize Cosmos components
    this.cosmosSubmitter = new CosmosSubmitter(
      config.cosmos.rpcEndpoint,
      config.cosmos.mnemonic,
      config.cosmos.gasPrice,
      logger
    );

    this.cosmosMonitor = new CosmosMonitor(
      config.cosmos.rpcEndpoint,
      config.cosmos.startHeight,
      config.cosmos.pollInterval,
      logger
    );

    // Initialize circuit breakers
    this.cosmosCircuitBreaker = new CircuitBreaker(5, 60000, logger);
    this.besuCircuitBreaker = new CircuitBreaker(5, 60000, logger);
  }

  /**
   * Start the relayer
   */
  async start(): Promise<void> {
    this.logger.info('Starting Astra Clear Relayer');

    try {
      // Connect to Cosmos Hub
      await retry(
        () => this.cosmosSubmitter.connect(),
        this.config.retry,
        this.logger
      );

      await retry(
        () => this.cosmosMonitor.connect(),
        this.config.retry,
        this.logger
      );

      this.logger.info('Connected to Cosmos Hub', {
        validatorAddress: this.cosmosSubmitter.getValidatorAddress(),
      });

      // Start monitoring Besu for TransferInitiated events
      this.logger.info('Starting Besu monitor');
      await this.besuMonitor.start(this.handleBesuTransfer.bind(this));

      // Start monitoring Cosmos for MintCommand events
      this.logger.info('Starting Cosmos monitor');
      await this.cosmosMonitor.start(this.handleCosmosCommand.bind(this));

      this.logger.info('Relayer started successfully');
    } catch (error) {
      this.logger.error('Failed to start relayer', {
        error: error instanceof Error ? error.message : String(error),
      });
      throw error;
    }
  }

  /**
   * Handle TransferInitiated event from Besu
   * Flow: Besu Transfer -> Cosmos Vote (Requirement 6.1 -> 6.2)
   */
  private async handleBesuTransfer(event: TransferEvent): Promise<void> {
    // Check if already processed
    if (this.processedTransfers.has(event.txHash)) {
      this.logger.debug('Transfer already processed, skipping', {
        txHash: event.txHash,
      });
      return;
    }

    this.logger.info('Processing Besu transfer event', {
      txHash: event.txHash,
      sender: event.sender,
      recipient: event.recipient,
      amount: event.amount.toString(),
      destChain: event.destChain,
    });

    try {
      // Submit vote to Cosmos Hub with circuit breaker and retry
      await this.cosmosCircuitBreaker.execute(async () => {
        await retryBlockchain(
          () => this.cosmosSubmitter.submitVote(event),
          this.config.retry,
          this.logger
        );
      });

      // Mark as processed
      this.processedTransfers.add(event.txHash);

      this.logger.info('Successfully submitted vote for transfer', {
        txHash: event.txHash,
      });
    } catch (error) {
      this.logger.error('Failed to process Besu transfer', {
        txHash: event.txHash,
        error: error instanceof Error ? error.message : String(error),
      });

      // Don't throw - continue processing other events
      // The event will be retried on next relayer restart if not processed
    }
  }

  /**
   * Handle MintCommand event from Cosmos
   * Flow: Cosmos MintCommand -> Besu Execution (Requirement 6.3 -> 6.4)
   */
  private async handleCosmosCommand(command: MintCommand): Promise<void> {
    // Check if already processed
    if (this.processedCommands.has(command.commandId)) {
      this.logger.debug('Command already processed, skipping', {
        commandId: command.commandId,
      });
      return;
    }

    this.logger.info('Processing Cosmos mint command', {
      commandId: command.commandId,
      targetChain: command.targetChain,
      recipient: command.recipient,
      amount: command.amount.toString(),
      signatureCount: command.signatures.length,
    });

    try {
      // Execute mint command on Besu with circuit breaker and retry
      await this.besuCircuitBreaker.execute(async () => {
        await retryBlockchain(
          () => this.besuExecutor.executeMintCommand(command),
          this.config.retry,
          this.logger
        );
      });

      // Mark as processed
      this.processedCommands.add(command.commandId);

      this.logger.info('Successfully executed mint command', {
        commandId: command.commandId,
      });
    } catch (error) {
      this.logger.error('Failed to process Cosmos command', {
        commandId: command.commandId,
        error: error instanceof Error ? error.message : String(error),
      });

      // Don't throw - continue processing other events
    }
  }

  /**
   * Stop the relayer
   */
  async stop(): Promise<void> {
    this.logger.info('Stopping Astra Clear Relayer');

    try {
      // Stop monitors
      this.besuMonitor.stop();
      this.cosmosMonitor.stop();

      // Disconnect clients
      await this.cosmosSubmitter.disconnect();
      this.cosmosMonitor.disconnect();

      this.logger.info('Relayer stopped successfully');
    } catch (error) {
      this.logger.error('Error stopping relayer', {
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }

  /**
   * Get relayer status
   */
  getStatus(): RelayerStatus {
    return {
      besu: {
        lastProcessedBlock: this.besuMonitor.getLastProcessedBlock(),
        executorAddress: this.besuExecutor.getExecutorAddress(),
        walletAddress: this.besuExecutor.getWalletAddress(),
      },
      cosmos: {
        lastProcessedHeight: this.cosmosMonitor.getLastProcessedHeight(),
        validatorAddress: this.cosmosSubmitter.getValidatorAddress(),
        connected: this.cosmosSubmitter.isConnected(),
      },
      circuitBreakers: {
        cosmos: this.cosmosCircuitBreaker.getState(),
        besu: this.besuCircuitBreaker.getState(),
      },
      processed: {
        transferCount: this.processedTransfers.size,
        commandCount: this.processedCommands.size,
      },
    };
  }
}

/**
 * Relayer status information
 */
export interface RelayerStatus {
  besu: {
    lastProcessedBlock: number;
    executorAddress: string;
    walletAddress: string;
  };
  cosmos: {
    lastProcessedHeight: number;
    validatorAddress: string;
    connected: boolean;
  };
  circuitBreakers: {
    cosmos: 'closed' | 'open' | 'half-open';
    besu: 'closed' | 'open' | 'half-open';
  };
  processed: {
    transferCount: number;
    commandCount: number;
  };
}
