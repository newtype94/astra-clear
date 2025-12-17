#!/bin/bash

# Start Besu Networks for Bank A and Bank B
echo "Starting Hyperledger Besu networks..."

# Navigate to docker directory
cd "$(dirname "$0")/../docker"

# Start the networks using docker-compose
docker-compose -f docker-compose.besu.yml up -d

echo "Besu networks started successfully!"
echo "Bank A RPC: http://localhost:8545"
echo "Bank A WebSocket: ws://localhost:8546"
echo "Bank B RPC: http://localhost:8547"
echo "Bank B WebSocket: ws://localhost:8548"

# Wait for networks to be ready
echo "Waiting for networks to be ready..."
sleep 10

# Check if networks are responding
echo "Checking network connectivity..."
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H "Content-Type: application/json" http://localhost:8545
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H "Content-Type: application/json" http://localhost:8547

echo "Besu networks are ready!"