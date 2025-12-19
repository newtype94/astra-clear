import dotenv from 'dotenv';

// Load environment variables
dotenv.config();

/**
 * Relayer configuration
 * Loads configuration from environment variables
 */
export interface RelayerConfig {
  // Besu configuration
  besu: {
    rpcUrl: string;
    wsUrl: string | null;
    gatewayAddress: string;
    executorAddress: string;
    privateKey: string;
    startBlock: number;
    pollInterval: number;
  };

  // Cosmos configuration
  cosmos: {
    rpcEndpoint: string;
    mnemonic: string;
    gasPrice: string;
    startHeight: number;
    pollInterval: number;
  };

  // Logging configuration
  logging: {
    level: string;
    format: 'json' | 'simple';
  };

  // Retry configuration
  retry: {
    maxAttempts: number;
    backoffMs: number;
    maxBackoffMs: number;
  };
}

/**
 * Load configuration from environment variables
 */
export function loadConfig(): RelayerConfig {
  // Validate required environment variables
  const required = [
    'BESU_RPC_URL',
    'BESU_GATEWAY_ADDRESS',
    'BESU_EXECUTOR_ADDRESS',
    'BESU_PRIVATE_KEY',
    'COSMOS_RPC_ENDPOINT',
    'COSMOS_MNEMONIC',
  ];

  for (const key of required) {
    if (!process.env[key]) {
      throw new Error(`Missing required environment variable: ${key}`);
    }
  }

  const config: RelayerConfig = {
    besu: {
      rpcUrl: process.env.BESU_RPC_URL!,
      wsUrl: process.env.BESU_WS_URL || null,
      gatewayAddress: process.env.BESU_GATEWAY_ADDRESS!,
      executorAddress: process.env.BESU_EXECUTOR_ADDRESS!,
      privateKey: process.env.BESU_PRIVATE_KEY!,
      startBlock: parseInt(process.env.BESU_START_BLOCK || '0'),
      pollInterval: parseInt(process.env.BESU_POLL_INTERVAL || '5000'),
    },

    cosmos: {
      rpcEndpoint: process.env.COSMOS_RPC_ENDPOINT!,
      mnemonic: process.env.COSMOS_MNEMONIC!,
      gasPrice: process.env.COSMOS_GAS_PRICE || '0.025uatom',
      startHeight: parseInt(process.env.COSMOS_START_HEIGHT || '0'),
      pollInterval: parseInt(process.env.COSMOS_POLL_INTERVAL || '3000'),
    },

    logging: {
      level: process.env.LOG_LEVEL || 'info',
      format: (process.env.LOG_FORMAT as 'json' | 'simple') || 'json',
    },

    retry: {
      maxAttempts: parseInt(process.env.RETRY_MAX_ATTEMPTS || '3'),
      backoffMs: parseInt(process.env.RETRY_BACKOFF_MS || '1000'),
      maxBackoffMs: parseInt(process.env.RETRY_MAX_BACKOFF_MS || '30000'),
    },
  };

  return config;
}

/**
 * Validate configuration
 */
export function validateConfig(config: RelayerConfig): void {
  // Validate Besu configuration
  if (!config.besu.rpcUrl.startsWith('http')) {
    throw new Error('Invalid BESU_RPC_URL: must start with http or https');
  }

  if (config.besu.wsUrl && !config.besu.wsUrl.startsWith('ws')) {
    throw new Error('Invalid BESU_WS_URL: must start with ws or wss');
  }

  if (!config.besu.gatewayAddress.match(/^0x[a-fA-F0-9]{40}$/)) {
    throw new Error('Invalid BESU_GATEWAY_ADDRESS: must be valid Ethereum address');
  }

  if (!config.besu.executorAddress.match(/^0x[a-fA-F0-9]{40}$/)) {
    throw new Error('Invalid BESU_EXECUTOR_ADDRESS: must be valid Ethereum address');
  }

  if (!config.besu.privateKey.match(/^(0x)?[a-fA-F0-9]{64}$/)) {
    throw new Error('Invalid BESU_PRIVATE_KEY: must be 32-byte hex string');
  }

  // Validate Cosmos configuration
  if (!config.cosmos.rpcEndpoint.startsWith('http')) {
    throw new Error('Invalid COSMOS_RPC_ENDPOINT: must start with http or https');
  }

  const mnemonicWords = config.cosmos.mnemonic.split(' ');
  if (mnemonicWords.length < 12) {
    throw new Error('Invalid COSMOS_MNEMONIC: must be at least 12 words');
  }

  // Validate retry configuration
  if (config.retry.maxAttempts < 1) {
    throw new Error('RETRY_MAX_ATTEMPTS must be at least 1');
  }

  if (config.retry.backoffMs < 100) {
    throw new Error('RETRY_BACKOFF_MS must be at least 100');
  }

  if (config.retry.maxBackoffMs < config.retry.backoffMs) {
    throw new Error('RETRY_MAX_BACKOFF_MS must be greater than RETRY_BACKOFF_MS');
  }
}
