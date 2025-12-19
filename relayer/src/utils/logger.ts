import winston from 'winston';

/**
 * Create a configured logger instance
 */
export function createLogger(
  level: string = 'info',
  format: 'json' | 'simple' = 'json'
): winston.Logger {
  const logFormat =
    format === 'json'
      ? winston.format.combine(
          winston.format.timestamp(),
          winston.format.errors({ stack: true }),
          winston.format.json()
        )
      : winston.format.combine(
          winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
          winston.format.errors({ stack: true }),
          winston.format.printf(
            ({ timestamp, level, message, ...meta }) =>
              `${timestamp} [${level.toUpperCase()}]: ${message} ${
                Object.keys(meta).length ? JSON.stringify(meta) : ''
              }`
          )
        );

  return winston.createLogger({
    level,
    format: logFormat,
    transports: [
      new winston.transports.Console({
        format:
          format === 'simple'
            ? winston.format.combine(winston.format.colorize(), logFormat)
            : logFormat,
      }),
      new winston.transports.File({
        filename: 'logs/error.log',
        level: 'error',
      }),
      new winston.transports.File({
        filename: 'logs/combined.log',
      }),
    ],
  });
}
