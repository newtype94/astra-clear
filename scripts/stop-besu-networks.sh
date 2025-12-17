#!/bin/bash

# Stop Besu Networks
echo "Stopping Hyperledger Besu networks..."

# Navigate to docker directory
cd "$(dirname "$0")/../docker"

# Stop the networks using docker-compose
docker-compose -f docker-compose.besu.yml down

echo "Besu networks stopped successfully!"