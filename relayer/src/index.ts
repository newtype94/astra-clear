#!/usr/bin/env node

import { Relayer } from './relayer';
import { loadConfig, validateConfig } from './config';
import { createLogger } from './utils/logger';

/**
 * Main entry point for Astra Clear Relayer
 */
async function main() {
  let logger;

  try {
    // Load configuration
    const config = loadConfig();

    // Create logger
    logger = createLogger(config.logging.level, config.logging.format);

    logger.info('Astra Clear Relayer starting', {
      version: '1.0.0',
      besuRpc: config.besu.rpcUrl,
      cosmosRpc: config.cosmos.rpcEndpoint,
    });

    // Validate configuration
    validateConfig(config);
    logger.info('Configuration validated successfully');

    // Create relayer instance
    const relayer = new Relayer(config, logger);

    // Handle graceful shutdown
    let isShuttingDown = false;

    const shutdown = async (signal: string) => {
      if (isShuttingDown) {
        logger.warn('Shutdown already in progress');
        return;
      }

      isShuttingDown = true;
      logger.info(`Received ${signal}, shutting down gracefully`);

      try {
        await relayer.stop();
        logger.info('Relayer shutdown complete');
        process.exit(0);
      } catch (error) {
        logger.error('Error during shutdown', {
          error: error instanceof Error ? error.message : String(error),
        });
        process.exit(1);
      }
    };

    // Register signal handlers
    process.on('SIGINT', () => shutdown('SIGINT'));
    process.on('SIGTERM', () => shutdown('SIGTERM'));

    // Handle uncaught errors
    process.on('uncaughtException', (error) => {
      logger.error('Uncaught exception', {
        error: error.message,
        stack: error.stack,
      });
      shutdown('uncaughtException');
    });

    process.on('unhandledRejection', (reason, promise) => {
      logger.error('Unhandled rejection', {
        reason: String(reason),
        promise: String(promise),
      });
    });

    // Start relayer
    await relayer.start();

    // Log status periodically
    setInterval(() => {
      const status = relayer.getStatus();
      logger.info('Relayer status', status);
    }, 60000); // Log status every 60 seconds

    logger.info('Relayer is running. Press Ctrl+C to stop.');
  } catch (error) {
    if (logger) {
      logger.error('Fatal error', {
        error: error instanceof Error ? error.message : String(error),
        stack: error instanceof Error ? error.stack : undefined,
      });
    } else {
      console.error('Fatal error:', error);
    }
    process.exit(1);
  }
}

// Run main function
main();
