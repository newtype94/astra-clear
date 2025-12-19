import { ethers, Contract, EventLog } from 'ethers';
import { TransferEvent } from '../types';
import { Logger } from 'winston';

/**
 * BesuMonitor monitors Besu blockchain for TransferInitiated events
 * Requirement 6.1: Besu 이벤트 모니터링 구현
 */
export class BesuMonitor {
  private provider: ethers.WebSocketProvider | ethers.JsonRpcProvider;
  private gateway: Contract;
  private logger: Logger;
  private lastProcessedBlock: number;

  // Gateway contract ABI (minimal, only the events we need)
  private static readonly GATEWAY_ABI = [
    'event TransferInitiated(address indexed sender, string recipient, uint256 amount, string destChain, uint256 nonce)',
  ];

  constructor(
    rpcUrl: string,
    wsUrl: string | null,
    gatewayAddress: string,
    startBlock: number,
    logger: Logger
  ) {
    // Use WebSocket if available, otherwise fall back to HTTP polling
    if (wsUrl) {
      this.provider = new ethers.WebSocketProvider(wsUrl);
    } else {
      this.provider = new ethers.JsonRpcProvider(rpcUrl);
    }

    this.gateway = new Contract(
      gatewayAddress,
      BesuMonitor.GATEWAY_ABI,
      this.provider
    );

    this.logger = logger;
    this.lastProcessedBlock = startBlock;
  }

  /**
   * Start monitoring for TransferInitiated events
   * Uses WebSocket for real-time monitoring or polling for HTTP
   */
  async start(
    callback: (event: TransferEvent) => Promise<void>
  ): Promise<void> {
    this.logger.info('Starting Besu event monitor', {
      startBlock: this.lastProcessedBlock,
    });

    if (this.provider instanceof ethers.WebSocketProvider) {
      // Real-time WebSocket monitoring
      await this.startWebSocketMonitoring(callback);
    } else {
      // HTTP polling fallback
      await this.startPollingMonitoring(callback);
    }
  }

  /**
   * Real-time monitoring using WebSocket
   */
  private async startWebSocketMonitoring(
    callback: (event: TransferEvent) => Promise<void>
  ): Promise<void> {
    this.logger.info('Using WebSocket for real-time event monitoring');

    // Listen for new events
    this.gateway.on(
      'TransferInitiated',
      async (
        sender: string,
        recipient: string,
        amount: bigint,
        destChain: string,
        nonce: bigint,
        event: EventLog
      ) => {
        try {
          const transferEvent = await this.parseTransferEvent(event);
          await callback(transferEvent);
          this.lastProcessedBlock = event.blockNumber;
        } catch (error) {
          this.logger.error('Error processing WebSocket event', {
            error: error instanceof Error ? error.message : String(error),
            txHash: event.transactionHash,
          });
        }
      }
    );

    // Handle connection errors
    this.provider.on('error', (error) => {
      this.logger.error('WebSocket provider error', {
        error: error.message,
      });
    });
  }

  /**
   * Polling-based monitoring using HTTP RPC
   */
  private async startPollingMonitoring(
    callback: (event: TransferEvent) => Promise<void>,
    pollInterval: number = 5000
  ): Promise<void> {
    this.logger.info('Using HTTP polling for event monitoring', {
      pollInterval,
    });

    const poll = async () => {
      try {
        const currentBlock = await this.provider.getBlockNumber();

        if (currentBlock > this.lastProcessedBlock) {
          const events = await this.gateway.queryFilter(
            'TransferInitiated',
            this.lastProcessedBlock + 1,
            currentBlock
          );

          for (const event of events) {
            if (event instanceof EventLog) {
              const transferEvent = await this.parseTransferEvent(event);
              await callback(transferEvent);
            }
          }

          this.lastProcessedBlock = currentBlock;
          this.logger.debug('Processed blocks', {
            fromBlock: this.lastProcessedBlock + 1,
            toBlock: currentBlock,
            eventsFound: events.length,
          });
        }
      } catch (error) {
        this.logger.error('Error during polling', {
          error: error instanceof Error ? error.message : String(error),
        });
      }

      // Schedule next poll
      setTimeout(poll, pollInterval);
    };

    // Start polling
    await poll();
  }

  /**
   * Parse raw event log into TransferEvent
   */
  private async parseTransferEvent(event: EventLog): Promise<TransferEvent> {
    const block = await this.provider.getBlock(event.blockNumber);
    const parsed = this.gateway.interface.parseLog({
      topics: [...event.topics],
      data: event.data,
    });

    if (!parsed) {
      throw new Error(`Failed to parse event: ${event.transactionHash}`);
    }

    const sourceChain = await this.getSourceChainId();

    return {
      txHash: event.transactionHash,
      blockNumber: event.blockNumber,
      sender: parsed.args.sender,
      recipient: parsed.args.recipient,
      amount: parsed.args.amount,
      sourceChain,
      destChain: parsed.args.destChain,
      nonce: parsed.args.nonce,
      timestamp: block?.timestamp || Math.floor(Date.now() / 1000),
    };
  }

  /**
   * Get the source chain identifier
   */
  private async getSourceChainId(): Promise<string> {
    const network = await this.provider.getNetwork();
    return `besu-${network.chainId}`;
  }

  /**
   * Stop monitoring
   */
  async stop(): Promise<void> {
    this.logger.info('Stopping Besu event monitor');

    // Remove all listeners
    await this.gateway.removeAllListeners();

    // Close provider if it's a WebSocket
    if (this.provider instanceof ethers.WebSocketProvider) {
      await this.provider.destroy();
    }
  }

  /**
   * Get the last processed block number
   */
  getLastProcessedBlock(): number {
    return this.lastProcessedBlock;
  }
}
