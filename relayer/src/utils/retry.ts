import { Logger } from 'winston';

/**
 * Retry configuration
 */
export interface RetryConfig {
  maxAttempts: number;
  backoffMs: number;
  maxBackoffMs: number;
  shouldRetry?: (error: Error) => boolean;
}

/**
 * Default retry configuration
 */
const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxAttempts: 3,
  backoffMs: 1000,
  maxBackoffMs: 30000,
};

/**
 * Retry a function with exponential backoff
 * Requirement 6.5: Relayer 오류 처리 및 재시도 로직
 */
export async function retry<T>(
  fn: () => Promise<T>,
  config: Partial<RetryConfig> = {},
  logger?: Logger
): Promise<T> {
  const finalConfig = { ...DEFAULT_RETRY_CONFIG, ...config };
  let lastError: Error | null = null;
  let attempt = 0;

  while (attempt < finalConfig.maxAttempts) {
    attempt++;

    try {
      return await fn();
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error));

      // Check if we should retry this error
      if (finalConfig.shouldRetry && !finalConfig.shouldRetry(lastError)) {
        logger?.error('Non-retryable error encountered', {
          error: lastError.message,
          attempt,
        });
        throw lastError;
      }

      // Don't sleep on last attempt
      if (attempt >= finalConfig.maxAttempts) {
        break;
      }

      // Calculate backoff with exponential increase
      const backoff = Math.min(
        finalConfig.backoffMs * Math.pow(2, attempt - 1),
        finalConfig.maxBackoffMs
      );

      logger?.warn('Operation failed, retrying', {
        error: lastError.message,
        attempt,
        maxAttempts: finalConfig.maxAttempts,
        backoffMs: backoff,
      });

      // Wait before retrying
      await sleep(backoff);
    }
  }

  // All retries exhausted
  logger?.error('All retry attempts exhausted', {
    error: lastError?.message,
    attempts: attempt,
  });

  throw lastError || new Error('Retry failed with unknown error');
}

/**
 * Retry wrapper specifically for network operations
 * Automatically retries on network errors
 */
export async function retryNetwork<T>(
  fn: () => Promise<T>,
  config: Partial<RetryConfig> = {},
  logger?: Logger
): Promise<T> {
  return retry(
    fn,
    {
      ...config,
      shouldRetry: (error) => {
        // Retry on network errors
        const networkErrors = [
          'ECONNREFUSED',
          'ECONNRESET',
          'ETIMEDOUT',
          'ENOTFOUND',
          'ENETUNREACH',
          'EHOSTUNREACH',
        ];

        const errorMessage = error.message.toUpperCase();
        const isNetworkError = networkErrors.some((code) =>
          errorMessage.includes(code)
        );

        // Also retry on timeout and rate limit errors
        const isTimeout = errorMessage.includes('TIMEOUT');
        const isRateLimit = errorMessage.includes('RATE LIMIT') ||
                           errorMessage.includes('TOO MANY REQUESTS');

        return isNetworkError || isTimeout || isRateLimit;
      },
    },
    logger
  );
}

/**
 * Retry wrapper for blockchain operations
 * Handles common blockchain errors
 */
export async function retryBlockchain<T>(
  fn: () => Promise<T>,
  config: Partial<RetryConfig> = {},
  logger?: Logger
): Promise<T> {
  return retry(
    fn,
    {
      ...config,
      shouldRetry: (error) => {
        const errorMessage = error.message.toLowerCase();

        // Don't retry on transaction errors that are permanent
        const permanentErrors = [
          'insufficient funds',
          'nonce too low',
          'invalid signature',
          'transaction reverted',
          'execution reverted',
        ];

        const isPermanent = permanentErrors.some((msg) =>
          errorMessage.includes(msg)
        );

        if (isPermanent) {
          return false;
        }

        // Retry on temporary errors
        const temporaryErrors = [
          'timeout',
          'connection',
          'network',
          'nonce too high',
          'replacement transaction underpriced',
          'known transaction',
        ];

        return temporaryErrors.some((msg) => errorMessage.includes(msg));
      },
    },
    logger
  );
}

/**
 * Circuit breaker for preventing cascading failures
 */
export class CircuitBreaker {
  private failureCount: number = 0;
  private lastFailureTime: number = 0;
  private state: 'closed' | 'open' | 'half-open' = 'closed';
  private logger?: Logger;

  constructor(
    private threshold: number = 5,
    private timeout: number = 60000,
    logger?: Logger
  ) {
    this.logger = logger;
  }

  /**
   * Execute function with circuit breaker protection
   */
  async execute<T>(fn: () => Promise<T>): Promise<T> {
    // Check circuit state
    if (this.state === 'open') {
      const timeSinceLastFailure = Date.now() - this.lastFailureTime;

      if (timeSinceLastFailure > this.timeout) {
        this.state = 'half-open';
        this.logger?.info('Circuit breaker entering half-open state');
      } else {
        throw new Error(
          `Circuit breaker is open. Retry after ${
            this.timeout - timeSinceLastFailure
          }ms`
        );
      }
    }

    try {
      const result = await fn();

      // Success - reset if in half-open state
      if (this.state === 'half-open') {
        this.reset();
        this.logger?.info('Circuit breaker closed after successful execution');
      }

      return result;
    } catch (error) {
      this.recordFailure();
      throw error;
    }
  }

  /**
   * Record a failure
   */
  private recordFailure(): void {
    this.failureCount++;
    this.lastFailureTime = Date.now();

    if (this.failureCount >= this.threshold) {
      this.state = 'open';
      this.logger?.warn('Circuit breaker opened', {
        failureCount: this.failureCount,
        threshold: this.threshold,
      });
    }
  }

  /**
   * Reset circuit breaker
   */
  reset(): void {
    this.failureCount = 0;
    this.state = 'closed';
    this.lastFailureTime = 0;
  }

  /**
   * Get current state
   */
  getState(): 'closed' | 'open' | 'half-open' {
    return this.state;
  }

  /**
   * Get failure count
   */
  getFailureCount(): number {
    return this.failureCount;
  }
}

/**
 * Sleep for specified milliseconds
 */
function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
