# Hyperledger Besu Networks Setup

This directory contains the configuration for two Hyperledger Besu networks representing Bank A and Bank B.

## Network Configuration

### Bank A Network
- **RPC Port**: 8545
- **WebSocket Port**: 8546  
- **P2P Port**: 30303
- **Coinbase**: 0xfe3b557e8fb62b89f4916b721be55ceb828dbd73

### Bank B Network
- **RPC Port**: 8547 (mapped from 8546 to avoid conflicts)
- **WebSocket Port**: 8548 (mapped from 8547 to avoid conflicts)
- **P2P Port**: 30304
- **Coinbase**: 0x627306090abaB3A6e1400e9345bC60c78a8BEf57

## Genesis Configuration

Both networks use the same genesis block configuration with:
- **Chain ID**: 1337
- **Consensus**: Clique (Proof of Authority)
- **Block Time**: 15 seconds
- **Gas Limit**: 0x8000000

## Pre-funded Accounts

The genesis block includes three pre-funded accounts:
1. `0xfe3b557e8fb62b89f4916b721be55ceb828dbd73` (Bank A Coinbase)
2. `0x627306090abaB3A6e1400e9345bC60c78a8BEf57` (Bank B Coinbase)  
3. `0xf17f52151EbEF6C7334FAD080c5704D77216b732` (Additional account)

Each account is pre-funded with a large amount of ETH for testing purposes.

## Starting the Networks

Use the provided scripts to start/stop the networks:

### Windows
```bash
scripts\start-besu-networks.bat
scripts\stop-besu-networks.bat
```

### Linux/macOS
```bash
scripts/start-besu-networks.sh
scripts/stop-besu-networks.sh
```

## Testing Connectivity

After starting the networks, you can test connectivity using curl:

```bash
# Test Bank A
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H "Content-Type: application/json" http://localhost:8545

# Test Bank B  
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H "Content-Type: application/json" http://localhost:8547
```