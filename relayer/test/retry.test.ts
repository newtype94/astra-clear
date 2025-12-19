import { describe, it, expect, vi, beforeEach } from 'vitest';
import { retry, retryNetwork, retryBlockchain, CircuitBreaker } from '../src/utils/retry';
import { createLogger } from '../src/utils/logger';

describe('retry', () => {
  const logger = createLogger('error', 'simple');

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('retry function', () => {
    it('should succeed on first attempt', async () => {
      const fn = vi.fn().mockResolvedValue('success');
      const result = await retry(fn, {}, logger);
      expect(result).toBe('success');
      expect(fn).toHaveBeenCalledTimes(1);
    });

    it('should retry on failure and eventually succeed', async () => {
      const fn = vi
        .fn()
        .mockRejectedValueOnce(new Error('fail 1'))
        .mockRejectedValueOnce(new Error('fail 2'))
        .mockResolvedValue('success');

      const result = await retry(fn, { maxAttempts: 3, backoffMs: 10 }, logger);
      expect(result).toBe('success');
      expect(fn).toHaveBeenCalledTimes(3);
    });

    it('should throw after max attempts', async () => {
      const fn = vi.fn().mockRejectedValue(new Error('persistent failure'));

      await expect(
        retry(fn, { maxAttempts: 3, backoffMs: 10 }, logger)
      ).rejects.toThrow('persistent failure');

      expect(fn).toHaveBeenCalledTimes(3);
    });

    it('should respect shouldRetry predicate', async () => {
      const fn = vi.fn().mockRejectedValue(new Error('non-retryable'));

      await expect(
        retry(
          fn,
          {
            maxAttempts: 3,
            shouldRetry: () => false,
          },
          logger
        )
      ).rejects.toThrow('non-retryable');

      expect(fn).toHaveBeenCalledTimes(1);
    });

    it('should use exponential backoff', async () => {
      const fn = vi
        .fn()
        .mockRejectedValueOnce(new Error('fail 1'))
        .mockRejectedValueOnce(new Error('fail 2'))
        .mockResolvedValue('success');

      const startTime = Date.now();
      await retry(fn, { maxAttempts: 3, backoffMs: 100, maxBackoffMs: 1000 }, logger);
      const duration = Date.now() - startTime;

      // First retry: 100ms, second retry: 200ms
      // Total should be at least 300ms
      expect(duration).toBeGreaterThanOrEqual(300);
    });
  });

  describe('retryNetwork', () => {
    it('should retry on network errors', async () => {
      const networkError = new Error('ECONNREFUSED');
      const fn = vi
        .fn()
        .mockRejectedValueOnce(networkError)
        .mockResolvedValue('success');

      const result = await retryNetwork(fn, { maxAttempts: 3, backoffMs: 10 }, logger);
      expect(result).toBe('success');
      expect(fn).toHaveBeenCalledTimes(2);
    });

    it('should retry on timeout errors', async () => {
      const timeoutError = new Error('Request timeout');
      const fn = vi
        .fn()
        .mockRejectedValueOnce(timeoutError)
        .mockResolvedValue('success');

      const result = await retryNetwork(fn, { maxAttempts: 3, backoffMs: 10 }, logger);
      expect(result).toBe('success');
    });

    it('should retry on rate limit errors', async () => {
      const rateLimitError = new Error('Too many requests');
      const fn = vi
        .fn()
        .mockRejectedValueOnce(rateLimitError)
        .mockResolvedValue('success');

      const result = await retryNetwork(fn, { maxAttempts: 3, backoffMs: 10 }, logger);
      expect(result).toBe('success');
    });
  });

  describe('retryBlockchain', () => {
    it('should not retry on permanent errors', async () => {
      const permanentError = new Error('insufficient funds');
      const fn = vi.fn().mockRejectedValue(permanentError);

      await expect(
        retryBlockchain(fn, { maxAttempts: 3, backoffMs: 10 }, logger)
      ).rejects.toThrow('insufficient funds');

      expect(fn).toHaveBeenCalledTimes(1);
    });

    it('should retry on temporary errors', async () => {
      const temporaryError = new Error('nonce too high');
      const fn = vi
        .fn()
        .mockRejectedValueOnce(temporaryError)
        .mockResolvedValue('success');

      const result = await retryBlockchain(fn, { maxAttempts: 3, backoffMs: 10 }, logger);
      expect(result).toBe('success');
      expect(fn).toHaveBeenCalledTimes(2);
    });

    it('should not retry on transaction reverted', async () => {
      const revertError = new Error('transaction reverted');
      const fn = vi.fn().mockRejectedValue(revertError);

      await expect(
        retryBlockchain(fn, { maxAttempts: 3, backoffMs: 10 }, logger)
      ).rejects.toThrow('transaction reverted');

      expect(fn).toHaveBeenCalledTimes(1);
    });
  });

  describe('CircuitBreaker', () => {
    let circuitBreaker: CircuitBreaker;

    beforeEach(() => {
      circuitBreaker = new CircuitBreaker(3, 1000, logger);
    });

    it('should execute successfully in closed state', async () => {
      const fn = vi.fn().mockResolvedValue('success');
      const result = await circuitBreaker.execute(fn);
      expect(result).toBe('success');
      expect(circuitBreaker.getState()).toBe('closed');
    });

    it('should open after threshold failures', async () => {
      const fn = vi.fn().mockRejectedValue(new Error('failure'));

      // Execute 3 times to reach threshold
      for (let i = 0; i < 3; i++) {
        await expect(circuitBreaker.execute(fn)).rejects.toThrow('failure');
      }

      expect(circuitBreaker.getState()).toBe('open');
      expect(circuitBreaker.getFailureCount()).toBe(3);
    });

    it('should reject immediately when open', async () => {
      const fn = vi.fn().mockRejectedValue(new Error('failure'));

      // Open the circuit
      for (let i = 0; i < 3; i++) {
        await expect(circuitBreaker.execute(fn)).rejects.toThrow();
      }

      expect(circuitBreaker.getState()).toBe('open');

      // Next call should fail immediately
      await expect(circuitBreaker.execute(fn)).rejects.toThrow(
        'Circuit breaker is open'
      );
    });

    it('should transition to half-open after timeout', async () => {
      const fn = vi.fn().mockRejectedValue(new Error('failure'));

      // Open the circuit
      for (let i = 0; i < 3; i++) {
        await expect(circuitBreaker.execute(fn)).rejects.toThrow();
      }

      expect(circuitBreaker.getState()).toBe('open');

      // Wait for timeout
      await new Promise((resolve) => setTimeout(resolve, 1100));

      // Next execution should move to half-open
      fn.mockResolvedValue('success');
      const result = await circuitBreaker.execute(fn);

      expect(result).toBe('success');
      expect(circuitBreaker.getState()).toBe('closed');
    });

    it('should reset on successful execution in half-open state', async () => {
      const fn = vi.fn().mockRejectedValue(new Error('failure'));

      // Open the circuit
      for (let i = 0; i < 3; i++) {
        await expect(circuitBreaker.execute(fn)).rejects.toThrow();
      }

      // Wait for timeout
      await new Promise((resolve) => setTimeout(resolve, 1100));

      // Succeed in half-open state
      fn.mockResolvedValue('success');
      await circuitBreaker.execute(fn);

      expect(circuitBreaker.getState()).toBe('closed');
      expect(circuitBreaker.getFailureCount()).toBe(0);
    });
  });
});
