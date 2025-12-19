import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { loadConfig, validateConfig, RelayerConfig } from '../src/config';

describe('config', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    // Reset process.env before each test
    process.env = { ...originalEnv };
  });

  afterEach(() => {
    // Restore original env
    process.env = originalEnv;
  });

  describe('loadConfig', () => {
    it('should throw if required env vars are missing', () => {
      process.env = {};
      expect(() => loadConfig()).toThrow('Missing required environment variable');
    });

    it('should load config from environment variables', () => {
      process.env.BESU_RPC_URL = 'http://localhost:8545';
      process.env.BESU_GATEWAY_ADDRESS = '0x1234567890123456789012345678901234567890';
      process.env.BESU_EXECUTOR_ADDRESS = '0x0987654321098765432109876543210987654321';
      process.env.BESU_PRIVATE_KEY = 'abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789';
      process.env.COSMOS_RPC_ENDPOINT = 'http://localhost:26657';
      process.env.COSMOS_MNEMONIC = 'word1 word2 word3 word4 word5 word6 word7 word8 word9 word10 word11 word12';

      const config = loadConfig();

      expect(config.besu.rpcUrl).toBe('http://localhost:8545');
      expect(config.besu.gatewayAddress).toBe('0x1234567890123456789012345678901234567890');
      expect(config.cosmos.rpcEndpoint).toBe('http://localhost:26657');
    });

    it('should use default values for optional env vars', () => {
      process.env.BESU_RPC_URL = 'http://localhost:8545';
      process.env.BESU_GATEWAY_ADDRESS = '0x1234567890123456789012345678901234567890';
      process.env.BESU_EXECUTOR_ADDRESS = '0x0987654321098765432109876543210987654321';
      process.env.BESU_PRIVATE_KEY = 'abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789';
      process.env.COSMOS_RPC_ENDPOINT = 'http://localhost:26657';
      process.env.COSMOS_MNEMONIC = 'word1 word2 word3 word4 word5 word6 word7 word8 word9 word10 word11 word12';

      const config = loadConfig();

      expect(config.besu.startBlock).toBe(0);
      expect(config.besu.pollInterval).toBe(5000);
      expect(config.cosmos.startHeight).toBe(0);
      expect(config.cosmos.pollInterval).toBe(3000);
      expect(config.logging.level).toBe('info');
      expect(config.retry.maxAttempts).toBe(3);
    });
  });

  describe('validateConfig', () => {
    let validConfig: RelayerConfig;

    beforeEach(() => {
      validConfig = {
        besu: {
          rpcUrl: 'http://localhost:8545',
          wsUrl: null,
          gatewayAddress: '0x1234567890123456789012345678901234567890',
          executorAddress: '0x0987654321098765432109876543210987654321',
          privateKey: 'abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789',
          startBlock: 0,
          pollInterval: 5000,
        },
        cosmos: {
          rpcEndpoint: 'http://localhost:26657',
          mnemonic: 'word1 word2 word3 word4 word5 word6 word7 word8 word9 word10 word11 word12',
          gasPrice: '0.025uatom',
          startHeight: 0,
          pollInterval: 3000,
        },
        logging: {
          level: 'info',
          format: 'json',
        },
        retry: {
          maxAttempts: 3,
          backoffMs: 1000,
          maxBackoffMs: 30000,
        },
      };
    });

    it('should pass for valid config', () => {
      expect(() => validateConfig(validConfig)).not.toThrow();
    });

    it('should throw for invalid Besu RPC URL', () => {
      validConfig.besu.rpcUrl = 'invalid-url';
      expect(() => validateConfig(validConfig)).toThrow(
        'Invalid BESU_RPC_URL: must start with http or https'
      );
    });

    it('should throw for invalid Besu WS URL', () => {
      validConfig.besu.wsUrl = 'http://invalid';
      expect(() => validateConfig(validConfig)).toThrow(
        'Invalid BESU_WS_URL: must start with ws or wss'
      );
    });

    it('should throw for invalid gateway address', () => {
      validConfig.besu.gatewayAddress = 'invalid-address';
      expect(() => validateConfig(validConfig)).toThrow(
        'Invalid BESU_GATEWAY_ADDRESS: must be valid Ethereum address'
      );
    });

    it('should throw for invalid executor address', () => {
      validConfig.besu.executorAddress = '0xinvalid';
      expect(() => validateConfig(validConfig)).toThrow(
        'Invalid BESU_EXECUTOR_ADDRESS: must be valid Ethereum address'
      );
    });

    it('should throw for invalid private key', () => {
      validConfig.besu.privateKey = 'short';
      expect(() => validateConfig(validConfig)).toThrow(
        'Invalid BESU_PRIVATE_KEY: must be 32-byte hex string'
      );
    });

    it('should throw for invalid Cosmos RPC URL', () => {
      validConfig.cosmos.rpcEndpoint = 'invalid';
      expect(() => validateConfig(validConfig)).toThrow(
        'Invalid COSMOS_RPC_ENDPOINT: must start with http or https'
      );
    });

    it('should throw for invalid mnemonic', () => {
      validConfig.cosmos.mnemonic = 'word1 word2 word3';
      expect(() => validateConfig(validConfig)).toThrow(
        'Invalid COSMOS_MNEMONIC: must be at least 12 words'
      );
    });

    it('should throw for invalid retry max attempts', () => {
      validConfig.retry.maxAttempts = 0;
      expect(() => validateConfig(validConfig)).toThrow(
        'RETRY_MAX_ATTEMPTS must be at least 1'
      );
    });

    it('should throw for invalid retry backoff', () => {
      validConfig.retry.backoffMs = 50;
      expect(() => validateConfig(validConfig)).toThrow(
        'RETRY_BACKOFF_MS must be at least 100'
      );
    });

    it('should throw for invalid retry max backoff', () => {
      validConfig.retry.maxBackoffMs = 500;
      expect(() => validateConfig(validConfig)).toThrow(
        'RETRY_MAX_BACKOFF_MS must be greater than RETRY_BACKOFF_MS'
      );
    });
  });
});
