# Interbank Netting Engine - Cosmos Hub

This directory contains the Cosmos SDK application that serves as the central clearing engine for the interbank netting system.

## Overview

The Cosmos Hub acts as the central clearing engine that:
- Validates cross-chain transfer events through validator consensus
- Issues and manages credit tokens representing interbank IOUs
- Performs periodic netting to minimize actual fund transfers
- Manages multi-signature operations for secure cross-chain commands

## Architecture

### Custom Modules

1. **x/oracle** - Validates external blockchain events through validator consensus
2. **x/netting** - Manages credit tokens and performs periodic netting operations  
3. **x/multisig** - Handles ECDSA multi-signature operations for cross-chain commands

### Key Components

- **Credit Tokens**: Bank-specific tokens in format `cred-{BankID}` representing IOUs
- **Validator Consensus**: 2/3 majority required for event confirmation
- **Periodic Netting**: Automatic netting every 10 blocks to minimize settlements
- **Multi-signature Commands**: ECDSA signatures for secure cross-chain operations

## Building

```bash
# Install dependencies
go mod tidy

# Build the daemon
make build
# or manually:
go build -o build/interbank-nettingd ./cmd/interbank-nettingd

# Install globally (optional)
make install
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run specific module tests
go test ./x/oracle/keeper -v
go test ./x/netting/keeper -v  
go test ./x/multisig/keeper -v

# Run only property-based tests (minimum 100 iterations each)
go test ./x/oracle/keeper -v -run TestProperty
go test ./x/netting/keeper -v -run TestProperty
go test ./x/multisig/keeper -v -run TestProperty

# Run tests with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Property-Based Testing

The system uses `gopter` for property-based testing with the following configuration:
- **Minimum iterations**: 100 per property test
- **Test library**: gopter (Go port of QuickCheck)
- **Tag format**: `**Feature: interbank-netting-engine, Property {N}: {description}**`

Each correctness property from the design document is implemented as a separate property-based test.

## Development

### Adding New Modules

1. Create module directory under `x/`
2. Implement keeper, types, and handlers
3. Register module in `app/app.go`
4. Add store keys and module manager

### Running Locally

```bash
# Build first
make build

# Initialize node
./build/interbank-nettingd init mynode --chain-id interbank-netting

# Create a key for validator
./build/interbank-nettingd keys add validator

# Add genesis account (replace with actual address)
./build/interbank-nettingd add-genesis-account $(./build/interbank-nettingd keys show validator -a) 1000000000stake

# Create genesis transaction
./build/interbank-nettingd gentx validator 1000000stake --chain-id interbank-netting

# Collect genesis transactions
./build/interbank-nettingd collect-gentxs

# Validate genesis file
./build/interbank-nettingd validate-genesis

# Start node
./build/interbank-nettingd start
```

### Useful Commands

```bash
# Check node status
curl http://localhost:26657/status

# Query account balance
./build/interbank-nettingd query bank balances $(./build/interbank-nettingd keys show validator -a)

# Send transaction (example)
./build/interbank-nettingd tx bank send validator cosmos1... 1000stake --chain-id interbank-netting

# Query module-specific data
./build/interbank-nettingd query oracle vote-status <tx-hash>
./build/interbank-nettingd query netting credit-balance <bank-id> <denom>
./build/interbank-nettingd query multisig validator-set
```

## Configuration

The application uses standard Cosmos SDK configuration:
- **Home directory**: `~/.interbank-netting`
- **Chain ID**: `interbank-netting`
- **Address prefix**: `cosmos`

## Integration

This Cosmos Hub integrates with:
- **Hyperledger Besu networks** (via relayers)
- **Smart contracts** (Gateway.sol and Executor.sol)
- **Off-chain relayers** (for event monitoring and command execution)

## Custom Types

Key data structures defined in `types/`:
- `TransferEvent` - Cross-chain transfer events
- `CreditToken` - Bank-issued IOU tokens
- `NettingCycle` - Periodic netting operations
- `MintCommand` - Multi-signed mint commands
- `ValidatorSet` - ECDSA validator management

## Interfaces

Module interfaces defined in `types/interfaces.go`:
- `OracleKeeper` - Event validation and consensus
- `NettingKeeper` - Credit token and netting operations
- `MultisigKeeper` - Multi-signature operations
- `EventEmitter` - Blockchain event emission
- `AuditLogger` - Comprehensive audit logging