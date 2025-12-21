# Developer Guide

Astra Clear 개발 환경 설정 및 실행 가이드

---

## 1. Prerequisites

### Required Tools

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.21+ | Cosmos SDK 빌드 |
| Node.js | 18+ | Smart Contract 개발 |
| Docker | 24+ | Besu 네트워크 실행 |
| Git | 2.40+ | 소스 관리 |

### Installation

**macOS (Homebrew)**
```bash
brew install go node docker git
```

**Ubuntu/Debian**
```bash
# Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# Node.js
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# Docker
sudo apt-get install docker.io docker-compose
```

**Windows**
```powershell
# Chocolatey 사용
choco install golang nodejs docker-desktop git
```

---

## 2. Project Setup

### Clone Repository
```bash
git clone https://github.com/[org]/astra-clear.git
cd astra-clear
```

### Directory Structure
```
astra-clear/
├── cosmos/          # Cosmos SDK Hub
├── contracts/       # Solidity Contracts
├── relayer/         # Cross-chain Relayer
├── docker/          # Docker configurations
├── scripts/         # Utility scripts
└── docs/            # Documentation
```

---

## 3. Cosmos Hub Setup

### Build
```bash
cd cosmos

# Install dependencies
go mod tidy

# Build binary
make build
# 또는
go build -o build/interbank-nettingd ./cmd/interbank-nettingd
```

### Initialize Chain
```bash
# Init node
./build/interbank-nettingd init mynode --chain-id interbank-netting

# Create validator key
./build/interbank-nettingd keys add validator

# Add genesis account
./build/interbank-nettingd add-genesis-account \
  $(./build/interbank-nettingd keys show validator -a) \
  1000000000stake

# Create genesis transaction
./build/interbank-nettingd gentx validator 1000000stake \
  --chain-id interbank-netting

# Collect genesis transactions
./build/interbank-nettingd collect-gentxs
```

### Run Node
```bash
./build/interbank-nettingd start
```

Node 상태 확인:
```bash
curl http://localhost:26657/status
```

---

## 4. Smart Contract Setup

### Install Dependencies
```bash
cd contracts
npm install
```

### Compile Contracts
```bash
npx hardhat compile
```

### Run Tests
```bash
npx hardhat test
```

### Deploy (Local Network)
```bash
# Start local Hardhat node
npx hardhat node

# Deploy contracts (new terminal)
npx hardhat run scripts/deploy.ts --network localhost
```

---

## 5. Besu Network Setup

### Using Docker Compose
```bash
# Start Besu networks
docker-compose -f docker/docker-compose.besu.yml up -d

# Check status
docker ps

# View logs
docker logs besu-bank-a
docker logs besu-bank-b
```

### Network Configuration

| Network | Chain ID | RPC Port | WS Port |
|---------|----------|----------|---------|
| Bank A | 1337 | 8545 | 8546 |
| Bank B | 1338 | 8555 | 8556 |

### Verify Connection
```bash
# Bank A
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:8545

# Bank B
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:8555
```

---

## 6. Relayer Setup

### Install Dependencies
```bash
cd relayer
npm install
```

### Configuration
```bash
cp .env.example .env
```

`.env` 파일 수정:
```env
# Cosmos Hub
COSMOS_RPC_ENDPOINT=http://localhost:26657
COSMOS_MNEMONIC="your validator mnemonic"

# Besu Networks
BESU_A_RPC_URL=http://localhost:8545
BESU_A_WS_URL=ws://localhost:8546
BESU_B_RPC_URL=http://localhost:8555
BESU_B_WS_URL=ws://localhost:8556

# Contract Addresses (after deployment)
GATEWAY_ADDRESS=0x...
EXECUTOR_ADDRESS=0x...
```

### Run Relayer
```bash
npm run start
```

---

## 7. Running Tests

### Cosmos Module Tests
```bash
cd cosmos

# All tests
go test ./...

# Specific module
go test ./x/oracle/keeper -v
go test ./x/netting/keeper -v
go test ./x/multisig/keeper -v

# Property-based tests only
go test ./x/oracle/keeper -v -run TestProperty
go test ./x/netting/keeper -v -run TestProperty
```

### Smart Contract Tests
```bash
cd contracts

# All tests
npx hardhat test

# Specific test file
npx hardhat test test/Gateway.test.ts
npx hardhat test test/Executor.test.ts

# With gas reporting
REPORT_GAS=true npx hardhat test
```

### Integration Tests
```bash
# Full system test (requires all components running)
make test-integration
```

---

## 8. Development Workflow

### Code Formatting

**Go**
```bash
cd cosmos
go fmt ./...
```

**TypeScript/Solidity**
```bash
cd contracts
npx prettier --write .
```

### Linting

**Go**
```bash
golangci-lint run
```

**Solidity**
```bash
npx solhint 'contracts/**/*.sol'
```

### Pre-commit Checks
```bash
# Go
cd cosmos && go test ./... && go fmt ./...

# Contracts
cd contracts && npx hardhat compile && npx hardhat test
```

---

## 9. Debugging

### Cosmos Hub Logs
```bash
# Verbose logging
./build/interbank-nettingd start --log_level debug

# Query specific module state
./build/interbank-nettingd query oracle vote-status <tx-hash>
./build/interbank-nettingd query netting credit-balance <bank-id> <denom>
```

### Besu Logs
```bash
docker logs -f besu-bank-a
docker logs -f besu-bank-b
```

### Relayer Logs
```bash
# Enable debug logging
LOG_LEVEL=debug npm run start
```

---

## 10. Common Issues

### Port Conflicts
```bash
# Check port usage
lsof -i :26657  # Cosmos
lsof -i :8545   # Besu A
lsof -i :8555   # Besu B

# Kill process
kill -9 <PID>
```

### Go Module Issues
```bash
go clean -modcache
go mod tidy
go mod download
```

### Docker Issues
```bash
# Restart Docker
docker-compose -f docker/docker-compose.besu.yml down
docker-compose -f docker/docker-compose.besu.yml up -d

# Clean volumes
docker-compose -f docker/docker-compose.besu.yml down -v
```

### Contract Compilation Errors
```bash
# Clean artifacts
npx hardhat clean
npx hardhat compile
```

---

## 11. Environment Variables Reference

| Variable | Description | Default |
|----------|-------------|---------|
| `COSMOS_RPC_ENDPOINT` | Cosmos Hub RPC | `http://localhost:26657` |
| `COSMOS_MNEMONIC` | Validator mnemonic | - |
| `BESU_A_RPC_URL` | Bank A RPC endpoint | `http://localhost:8545` |
| `BESU_B_RPC_URL` | Bank B RPC endpoint | `http://localhost:8555` |
| `GATEWAY_ADDRESS` | Gateway contract address | - |
| `EXECUTOR_ADDRESS` | Executor contract address | - |
| `LOG_LEVEL` | Logging level | `info` |

---

## 12. Next Steps

1. [ARCHITECTURE.md](ARCHITECTURE.md) - 시스템 구조 이해
2. [FEATURES.md](FEATURES.md) - 기능별 상세 설명
3. [WHITEPAPER.md](WHITEPAPER.md) - 설계 원칙 및 기술 명세
