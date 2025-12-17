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
go mod download

# Build the daemon
make build

# Install globally
make install
```

## Testing

```bash
# Run unit tests
make test-unit

# Run property-based tests (minimum 100 iterations each)
make test-property

# Run all tests
make test-all
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
# Initialize node
interbank-nettingd init mynode --chain-id interbank-netting

# Add genesis account
interbank-nettingd add-genesis-account cosmos1... 1000000000stake

# Create genesis transaction
interbank-nettingd gentx mykey 1000000stake --chain-id interbank-netting

# Collect genesis transactions
interbank-nettingd collect-gentxs

# Start node
interbank-nettingd start
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