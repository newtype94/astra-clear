#!/bin/bash

# Interbank Netting Engine - Development Environment Setup Script

set -e

echo "🌌 Interbank Netting Engine - Development Setup"
echo "================================================"

# Check if required tools are installed
check_tool() {
    if ! command -v $1 &> /dev/null; then
        echo "❌ $1 is not installed. Please install it first."
        exit 1
    else
        echo "✅ $1 is installed"
    fi
}

echo "Checking required tools..."
check_tool "go"
check_tool "docker"
check_tool "docker-compose"

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "Go version: $GO_VERSION"

# Setup Cosmos Hub
echo ""
echo "🚀 Setting up Cosmos Hub..."
cd cosmos

# Install Go dependencies
echo "Installing Go dependencies..."
go mod tidy

# Build the binary
echo "Building interbank-nettingd..."
make build

# Initialize the chain (if not already done)
if [ ! -d "$HOME/.interbank-netting" ]; then
    echo "Initializing blockchain..."
    ./build/interbank-nettingd init mynode --chain-id interbank-netting
    
    # Create validator key
    echo "Creating validator key..."
    ./build/interbank-nettingd keys add validator --keyring-backend test
    
    # Add genesis account
    echo "Adding genesis account..."
    VALIDATOR_ADDR=$(./build/interbank-nettingd keys show validator -a --keyring-backend test)
    ./build/interbank-nettingd add-genesis-account $VALIDATOR_ADDR 1000000000stake
    
    # Create genesis transaction
    echo "Creating genesis transaction..."
    ./build/interbank-nettingd gentx validator 1000000stake --chain-id interbank-netting --keyring-backend test
    
    # Collect genesis transactions
    echo "Collecting genesis transactions..."
    ./build/interbank-nettingd collect-gentxs
    
    echo "✅ Blockchain initialized successfully!"
else
    echo "✅ Blockchain already initialized"
fi

cd ..

# Setup Besu networks
echo ""
echo "🔗 Setting up Hyperledger Besu networks..."

# Check if Docker is running
if ! docker info &> /dev/null; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Start Besu networks
echo "Starting Besu networks..."
chmod +x scripts/start-besu-networks.sh
./scripts/start-besu-networks.sh

echo ""
echo "🎉 Development environment setup complete!"
echo ""
echo "📋 Next steps:"
echo "1. Start Cosmos Hub: cd cosmos && ./build/interbank-nettingd start"
echo "2. Run tests: cd cosmos && go test ./..."
echo "3. Check network status:"
echo "   - Cosmos Hub: curl http://localhost:26657/status"
echo "   - Bank A: curl -X POST --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}' http://localhost:8545"
echo "   - Bank B: curl -X POST --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}' http://localhost:8547"
echo ""
echo "📚 For more information, see README.md"