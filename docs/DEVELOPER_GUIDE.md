# Developer Guide

<details>
<summary><b>🇺🇸 English</b></summary>

## 1. Prerequisites

### Required Tools

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.21+ | Cosmos SDK build |
| Node.js | 18+ | Smart contract development |
| Docker | 24+ | Besu network execution |
| Git | 2.40+ | Source control |

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
go mod tidy
make build
```

### Initialize Chain
```bash
./build/interbank-nettingd init mynode --chain-id interbank-netting
./build/interbank-nettingd keys add validator
./build/interbank-nettingd add-genesis-account \
  $(./build/interbank-nettingd keys show validator -a) \
  1000000000stake
./build/interbank-nettingd gentx validator 1000000stake \
  --chain-id interbank-netting
./build/interbank-nettingd collect-gentxs
```

### Run Node
```bash
./build/interbank-nettingd start
```

Verify:
```bash
curl http://localhost:26657/status
```

---

## 4. Smart Contract Setup

### Install & Compile
```bash
cd contracts
npm install
npx hardhat compile
```

### Run Tests
```bash
npx hardhat test
```

### Deploy (Local)
```bash
npx hardhat node
npx hardhat run scripts/deploy.ts --network localhost
```

---

## 5. Besu Network Setup

### Using Docker
```bash
docker-compose -f docker/docker-compose.besu.yml up -d
docker ps
```

### Network Configuration

| Network | Chain ID | RPC Port | WS Port |
|---------|----------|----------|---------|
| Bank A | 1337 | 8545 | 8546 |
| Bank B | 1338 | 8555 | 8556 |

---

## 6. Relayer Setup

### Configure
```bash
cd relayer
npm install
cp .env.example .env
```

Edit `.env`:
```env
COSMOS_RPC_ENDPOINT=http://localhost:26657
COSMOS_MNEMONIC="your validator mnemonic"
BESU_A_RPC_URL=http://localhost:8545
BESU_B_RPC_URL=http://localhost:8555
```

### Run
```bash
npm run start
```

---

## 7. Running Tests

### Cosmos
```bash
cd cosmos
go test ./...
go test ./x/oracle/keeper -v -run TestProperty
```

### Contracts
```bash
cd contracts
npx hardhat test
REPORT_GAS=true npx hardhat test
```

---

## 8. Troubleshooting

### Port Conflicts
```bash
lsof -i :26657
lsof -i :8545
kill -9 <PID>
```

### Go Module Issues
```bash
go clean -modcache
go mod tidy
```

### Docker Issues
```bash
docker-compose -f docker/docker-compose.besu.yml down -v
docker-compose -f docker/docker-compose.besu.yml up -d
```

</details>

<details open>
<summary><b>🇰🇷 한국어</b></summary>

## 1. 필수 요구사항

### 필요 도구

| 도구 | 버전 | 용도 |
|------|------|------|
| Go | 1.21+ | Cosmos SDK 빌드 |
| Node.js | 18+ | 스마트 컨트랙트 개발 |
| Docker | 24+ | Besu 네트워크 실행 |
| Git | 2.40+ | 소스 관리 |

### 설치

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
choco install golang nodejs docker-desktop git
```

---

## 2. 프로젝트 설정

### 저장소 클론
```bash
git clone https://github.com/[org]/astra-clear.git
cd astra-clear
```

### 디렉토리 구조
```
astra-clear/
├── cosmos/          # Cosmos SDK Hub
├── contracts/       # Solidity 컨트랙트
├── relayer/         # 크로스체인 릴레이어
├── docker/          # Docker 설정
├── scripts/         # 유틸리티 스크립트
└── docs/            # 문서
```

---

## 3. Cosmos Hub 설정

### 빌드
```bash
cd cosmos
go mod tidy
make build
```

### 체인 초기화
```bash
./build/interbank-nettingd init mynode --chain-id interbank-netting
./build/interbank-nettingd keys add validator
./build/interbank-nettingd add-genesis-account \
  $(./build/interbank-nettingd keys show validator -a) \
  1000000000stake
./build/interbank-nettingd gentx validator 1000000stake \
  --chain-id interbank-netting
./build/interbank-nettingd collect-gentxs
```

### 노드 실행
```bash
./build/interbank-nettingd start
```

확인:
```bash
curl http://localhost:26657/status
```

---

## 4. 스마트 컨트랙트 설정

### 설치 및 컴파일
```bash
cd contracts
npm install
npx hardhat compile
```

### 테스트 실행
```bash
npx hardhat test
```

### 배포 (로컬)
```bash
npx hardhat node
npx hardhat run scripts/deploy.ts --network localhost
```

---

## 5. Besu 네트워크 설정

### Docker 사용
```bash
docker-compose -f docker/docker-compose.besu.yml up -d
docker ps
```

### 네트워크 구성

| 네트워크 | Chain ID | RPC 포트 | WS 포트 |
|----------|----------|----------|---------|
| Bank A | 1337 | 8545 | 8546 |
| Bank B | 1338 | 8555 | 8556 |

---

## 6. Relayer 설정

### 구성
```bash
cd relayer
npm install
cp .env.example .env
```

`.env` 수정:
```env
COSMOS_RPC_ENDPOINT=http://localhost:26657
COSMOS_MNEMONIC="your validator mnemonic"
BESU_A_RPC_URL=http://localhost:8545
BESU_B_RPC_URL=http://localhost:8555
```

### 실행
```bash
npm run start
```

---

## 7. 테스트 실행

### Cosmos
```bash
cd cosmos
go test ./...
go test ./x/oracle/keeper -v -run TestProperty
```

### Contracts
```bash
cd contracts
npx hardhat test
REPORT_GAS=true npx hardhat test
```

---

## 8. 트러블슈팅

### 포트 충돌
```bash
lsof -i :26657
lsof -i :8545
kill -9 <PID>
```

### Go 모듈 문제
```bash
go clean -modcache
go mod tidy
```

### Docker 문제
```bash
docker-compose -f docker/docker-compose.besu.yml down -v
docker-compose -f docker/docker-compose.besu.yml up -d
```

</details>
