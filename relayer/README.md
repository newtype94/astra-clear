# Astra Clear Relayer

Off-chain relayer service for the Astra Clear interbank netting system. This relayer bridges events between Hyperledger Besu and Cosmos Hub blockchains.

## Overview

The relayer performs two main functions:

1. **Besu → Cosmos**: Monitors `TransferInitiated` events on Besu and submits validator votes to Cosmos Hub oracle module
2. **Cosmos → Besu**: Monitors `MintCommand` events on Cosmos Hub and executes mint operations on Besu

## Architecture

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Besu      │         │   Relayer    │         │ Cosmos Hub  │
│  Gateway    │────────▶│              │────────▶│   Oracle    │
│  Contract   │ Events  │  - Monitor   │  Votes  │   Module    │
└─────────────┘         │  - Validate  │         └─────────────┘
                        │  - Submit    │
┌─────────────┐         │              │         ┌─────────────┐
│   Besu      │◀────────│              │◀────────│ Cosmos Hub  │
│  Executor   │ Execute │              │ Events  │   Oracle    │
│  Contract   │         └──────────────┘         │   Module    │
└─────────────┘                                  └─────────────┘
```

## Components

### Besu Components

- **BesuMonitor** ([src/besu/monitor.ts](src/besu/monitor.ts)): Monitors Besu blockchain for `TransferInitiated` events
  - Supports WebSocket (real-time) and HTTP polling (fallback)
  - Handles connection errors and automatic reconnection

- **BesuExecutor** ([src/besu/executor.ts](src/besu/executor.ts)): Executes mint commands on Besu
  - Verifies ECDSA signatures before execution
  - Handles gas estimation and transaction management

### Cosmos Components

- **CosmosSubmitter** ([src/cosmos/submitter.ts](src/cosmos/submitter.ts)): Submits validator votes to Cosmos Hub
  - Creates and signs `MsgVote` transactions
  - Manages gas estimation and fee calculation

- **CosmosMonitor** ([src/cosmos/monitor.ts](src/cosmos/monitor.ts)): Monitors Cosmos Hub for `MintCommand` events
  - Polls blocks for new events
  - Parses ECDSA signatures from event attributes

### Core Components

- **Relayer** ([src/relayer.ts](src/relayer.ts)): Main orchestrator
  - Coordinates all components
  - Implements circuit breakers for fault tolerance
  - Tracks processed events to avoid duplicates

- **Retry Logic** ([src/utils/retry.ts](src/utils/retry.ts)): Error handling and retry mechanisms
  - Exponential backoff with configurable limits
  - Network-aware and blockchain-aware retry strategies
  - Circuit breaker pattern for cascading failure prevention

## Installation

```bash
npm install
```

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

### Environment Variables

#### Besu Configuration

- `BESU_RPC_URL`: HTTP RPC endpoint (e.g., `http://localhost:8545`)
- `BESU_WS_URL`: WebSocket endpoint (optional, e.g., `ws://localhost:8546`)
- `BESU_GATEWAY_ADDRESS`: Gateway contract address
- `BESU_EXECUTOR_ADDRESS`: Executor contract address
- `BESU_PRIVATE_KEY`: Private key for signing transactions (32-byte hex)
- `BESU_START_BLOCK`: Starting block number (default: 0)
- `BESU_POLL_INTERVAL`: Polling interval in ms (default: 5000)

#### Cosmos Configuration

- `COSMOS_RPC_ENDPOINT`: Cosmos RPC endpoint (e.g., `http://localhost:26657`)
- `COSMOS_MNEMONIC`: 12 or 24-word mnemonic for validator account
- `COSMOS_GAS_PRICE`: Gas price (default: `0.025uatom`)
- `COSMOS_START_HEIGHT`: Starting block height (default: 0)
- `COSMOS_POLL_INTERVAL`: Polling interval in ms (default: 3000)

#### Logging Configuration

- `LOG_LEVEL`: Log level (`debug`, `info`, `warn`, `error`) (default: `info`)
- `LOG_FORMAT`: Log format (`json`, `simple`) (default: `json`)

#### Retry Configuration

- `RETRY_MAX_ATTEMPTS`: Maximum retry attempts (default: 3)
- `RETRY_BACKOFF_MS`: Initial backoff in ms (default: 1000)
- `RETRY_MAX_BACKOFF_MS`: Maximum backoff in ms (default: 30000)

## Usage

### Development

```bash
npm run dev
```

### Production

```bash
npm run build
npm start
```

### Testing

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage
```

### Linting and Formatting

```bash
npm run lint
npm run format
```

## Features

### Fault Tolerance

- **Circuit Breakers**: Prevents cascading failures when Besu or Cosmos become unavailable
- **Retry Logic**: Exponential backoff with configurable limits
- **Error Classification**: Distinguishes between temporary and permanent errors

### Event Processing

- **Duplicate Detection**: Tracks processed events to avoid reprocessing
- **Graceful Shutdown**: Properly cleans up connections on SIGINT/SIGTERM
- **Status Monitoring**: Periodic status logging for observability

### Security

- **Signature Verification**: Validates ECDSA signatures before executing mint commands
- **Private Key Management**: Secure handling of validator private keys
- **Gas Estimation**: Prevents overpaying for transactions

## Monitoring

The relayer logs status information every 60 seconds:

```json
{
  "besu": {
    "lastProcessedBlock": 12345,
    "executorAddress": "0x...",
    "walletAddress": "0x..."
  },
  "cosmos": {
    "lastProcessedHeight": 67890,
    "validatorAddress": "cosmos1...",
    "connected": true
  },
  "circuitBreakers": {
    "cosmos": "closed",
    "besu": "closed"
  },
  "processed": {
    "transferCount": 100,
    "commandCount": 95
  }
}
```

## Error Handling

The relayer implements comprehensive error handling:

### Permanent Errors (No Retry)

- Insufficient funds
- Invalid signatures
- Transaction reverted
- Nonce too low

### Temporary Errors (Retry with Backoff)

- Network connection errors
- RPC timeouts
- Rate limiting
- Nonce too high
- Known transaction

### Circuit Breaker States

- **Closed**: Normal operation
- **Open**: Too many failures, rejecting requests
- **Half-Open**: Testing if service recovered

## License

MIT
