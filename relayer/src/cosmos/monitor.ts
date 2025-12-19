import { StargateClient } from '@cosmjs/stargate';
import { MintCommand, ECDSASignature } from '../types';
import { Logger } from 'winston';

/**
 * CosmosMonitor monitors Cosmos Hub for MintCommand events
 * Requirement 6.3: Cosmos 이벤트 모니터링 구현
 */
export class CosmosMonitor {
  private client: StargateClient | null = null;
  private logger: Logger;
  private rpcEndpoint: string;
  private lastProcessedHeight: number;
  private pollInterval: number;
  private isRunning: boolean = false;

  constructor(
    rpcEndpoint: string,
    startHeight: number,
    pollInterval: number,
    logger: Logger
  ) {
    this.rpcEndpoint = rpcEndpoint;
    this.lastProcessedHeight = startHeight;
    this.pollInterval = pollInterval;
    this.logger = logger;
  }

  /**
   * Connect to Cosmos Hub
   */
  async connect(): Promise<void> {
    this.logger.info('Connecting to Cosmos Hub for monitoring', {
      rpcEndpoint: this.rpcEndpoint,
    });

    this.client = await StargateClient.connect(this.rpcEndpoint);

    const height = await this.client.getHeight();
    this.logger.info('Connected to Cosmos Hub', {
      currentHeight: height,
      startHeight: this.lastProcessedHeight,
    });
  }

  /**
   * Start monitoring for MintCommand events
   */
  async start(
    callback: (command: MintCommand) => Promise<void>
  ): Promise<void> {
    if (!this.client) {
      throw new Error('Cosmos client not connected. Call connect() first.');
    }

    this.logger.info('Starting Cosmos event monitor', {
      startHeight: this.lastProcessedHeight,
      pollInterval: this.pollInterval,
    });

    this.isRunning = true;
    await this.pollForEvents(callback);
  }

  /**
   * Poll for new blocks and events
   */
  private async pollForEvents(
    callback: (command: MintCommand) => Promise<void>
  ): Promise<void> {
    while (this.isRunning) {
      try {
        if (!this.client) {
          throw new Error('Client disconnected');
        }

        const currentHeight = await this.client.getHeight();

        if (currentHeight > this.lastProcessedHeight) {
          // Process blocks from lastProcessedHeight + 1 to currentHeight
          for (
            let height = this.lastProcessedHeight + 1;
            height <= currentHeight;
            height++
          ) {
            await this.processBlock(height, callback);
          }

          this.lastProcessedHeight = currentHeight;
          this.logger.debug('Processed blocks', {
            fromHeight: this.lastProcessedHeight + 1,
            toHeight: currentHeight,
          });
        }
      } catch (error) {
        this.logger.error('Error polling for events', {
          error: error instanceof Error ? error.message : String(error),
        });
      }

      // Wait for next poll interval
      await this.sleep(this.pollInterval);
    }
  }

  /**
   * Process a single block for MintCommand events
   */
  private async processBlock(
    height: number,
    callback: (command: MintCommand) => Promise<void>
  ): Promise<void> {
    if (!this.client) {
      throw new Error('Client not connected');
    }

    try {
      // Get block data
      const block = await this.client.getBlock(height);

      // Search for transactions with MintCommand events
      for (const tx of block.txs) {
        const txHash = this.bytesToHex(tx);

        // Get transaction result to access events
        const txResult = await this.client.getTx(txHash);
        if (!txResult) {
          continue;
        }

        // Look for MintCommandCreated events
        for (const event of txResult.events) {
          if (event.type === 'mint_command_created') {
            const mintCommand = this.parseMintCommandEvent(event, height);
            if (mintCommand) {
              await callback(mintCommand);
            }
          }
        }
      }
    } catch (error) {
      this.logger.error('Error processing block', {
        height,
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }

  /**
   * Parse MintCommandCreated event into MintCommand object
   */
  private parseMintCommandEvent(
    event: any,
    blockHeight: number
  ): MintCommand | null {
    try {
      // Extract attributes from event
      const attributes: { [key: string]: string } = {};
      for (const attr of event.attributes) {
        const key = this.decodeAttribute(attr.key);
        const value = this.decodeAttribute(attr.value);
        attributes[key] = value;
      }

      // Parse signatures
      const signatures: ECDSASignature[] = [];
      const signatureData = JSON.parse(attributes['signatures'] || '[]');
      for (const sig of signatureData) {
        signatures.push({
          v: parseInt(sig.v),
          r: sig.r,
          s: sig.s,
          signer: sig.signer,
        });
      }

      const mintCommand: MintCommand = {
        commandId: attributes['command_id'],
        targetChain: attributes['target_chain'],
        recipient: attributes['recipient'],
        amount: BigInt(attributes['amount']),
        signatures,
        createdAt: parseInt(attributes['created_at'] || '0'),
        status: 'pending',
      };

      this.logger.info('Parsed MintCommand event', {
        commandId: mintCommand.commandId,
        targetChain: mintCommand.targetChain,
        recipient: mintCommand.recipient,
        amount: mintCommand.amount.toString(),
        signatureCount: signatures.length,
      });

      return mintCommand;
    } catch (error) {
      this.logger.error('Failed to parse MintCommand event', {
        error: error instanceof Error ? error.message : String(error),
        event,
      });
      return null;
    }
  }

  /**
   * Query pending mint commands from oracle module
   */
  async queryPendingCommands(): Promise<MintCommand[]> {
    if (!this.client) {
      throw new Error('Client not connected');
    }

    try {
      // This would need to be implemented based on your oracle module's query interface
      // For now, returning empty array as placeholder
      this.logger.warn('queryPendingCommands not fully implemented');
      return [];
    } catch (error) {
      this.logger.error('Failed to query pending commands', {
        error: error instanceof Error ? error.message : String(error),
      });
      return [];
    }
  }

  /**
   * Stop monitoring
   */
  stop(): void {
    this.logger.info('Stopping Cosmos event monitor');
    this.isRunning = false;
  }

  /**
   * Disconnect from Cosmos Hub
   */
  disconnect(): void {
    if (this.client) {
      this.client.disconnect();
      this.client = null;
      this.logger.info('Disconnected from Cosmos Hub');
    }
  }

  /**
   * Get last processed height
   */
  getLastProcessedHeight(): number {
    return this.lastProcessedHeight;
  }

  /**
   * Helper: Convert bytes to hex string
   */
  private bytesToHex(bytes: Uint8Array): string {
    return Array.from(bytes)
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('');
  }

  /**
   * Helper: Decode base64 attribute (Cosmos events are base64 encoded)
   */
  private decodeAttribute(attr: string): string {
    try {
      return Buffer.from(attr, 'base64').toString('utf-8');
    } catch {
      return attr;
    }
  }

  /**
   * Helper: Sleep for specified milliseconds
   */
  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
