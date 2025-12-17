# astra-clear

# 🌌 astra-clear

**astra-clear**는 허가형 컨소시움 환경에서 스테이블코인 기반 결제를 실험하기 위한 **Interbank Netting & Clearing Engine POC**입니다.

이 프로젝트의 핵심 아이디어는 단순합니다.

> **사용자는 즉시 지급을 받고, 은행 간 정산은 나중에, 최소한으로 처리한다.**

Cosmos SDK를 중심 허브로 사용하여 은행 간 부채(IOU)를 토큰화하고 상계(Netting)함으로써, 실제 자금 이동을 최대한 압축합니다.

---

## ✨ What This Project Is (and Is Not)

### ✔ This is

* 개인 사이드 프로젝트이자 기술 **POC (Proof of Concept)**
* **100% 담보 스테이블코인** 환경 가정
* **완전 허가형 컨소시움** (신뢰된 금융기관만 참여)
* 실시간 사용자 지급 + 비동기 은행 간 정산 구조 실험

### ✖ This is NOT

* 프로덕션 레디 결제 네트워크
* 무담보 DeFi 프로토콜
* 파산, 디폴트, 리스크 엔진을 포함한 완전한 금융 시스템

---

## 🧠 Core Concept

### 1. Issuer-based Credit Token (IOU)

각 참여 은행은 Cosmos Hub 상에서 자신의 신용을 나타내는 부채 토큰을 발행합니다.

* 형식: `cred-{BankID}`
* 의미: "이 은행이 다른 은행에게 갚아야 할 돈"
* 가치: 1 `cred` = 1 Stablecoin Unit

중앙 유동성 풀은 존재하지 않으며, 모든 부채는 **발행자 기준(IOU)** 으로 명확히 분리됩니다.

---

### 2. Real-time User Payment, Deferred Settlement

* 송금인 체인에서는 토큰이 **Burn**
* 수신인 체인에서는 토큰이 **즉시 Mint**
* 은행 간 채권/채무는 Cosmos Hub에 기록
* Netting은 백그라운드에서 주기적으로 실행

사용자는 기다리지 않고, 은행은 효율적으로 정산합니다.

---

### 3. Netting via Token Burn

상호 보유 중인 `cred` 토큰은 주기적으로 상계됩니다.

예시:

* Bank A → Bank B: 100
* Bank B → Bank A: 30

결과:

* `cred-A` 30 Burn
* `cred-B` 30 Burn
* 순 부채: Bank A → Bank B = 70

---

## 🏗 Architecture Overview

```
┌─────────────┐        Events        ┌──────────────┐
│  Besu A     │ ──────────────────▶ │              │
│ (Source)    │                     │              │
└─────────────┘                     │              │
                                     │   Cosmos     │
┌─────────────┐        Commands      │     Hub      │
│  Besu B     │ ◀────────────────── │              │
│ (Destination│                     │              │
└─────────────┘                     └──────────────┘
```

### Components

* **Cosmos SDK Hub**

  * `x/oracle`: 외부 체인 이벤트 투표 및 확정
  * `x/netting`: cred 토큰 발행/소각 및 상계 로직
  * `x/multisig`: ECDSA 기반 서명 관리

* **Hyperledger Besu**

  * `Gateway.sol`: Source 체인 Burn + 이벤트 방출
  * `Executor.sol`: Destination 체인 Mint + 서명 검증

* **Relayer**

  * Besu ↔ Cosmos 간 이벤트/명령 전달
  * 비즈니스 로직 없음 (stateless)

---

## 🔄 End-to-End Flow (Simplified)

1. 사용자가 Source 체인에서 송금 요청
2. 토큰 Burn + 이벤트 발생
3. Cosmos Hub에서 Validator 합의
4. 수신 체인으로 Mint 명령 서명
5. Destination 체인에서 즉시 Mint
6. 은행 간 부채는 Cosmos에서 Netting

---

## 📦 Repository Structure (Planned)

```
astra-clear/
 ├─ cosmos/
 │   ├─ x/oracle/
 │   ├─ x/netting/
 │   └─ x/multisig/
 ├─ contracts/
 │   ├─ gateway.sol
 │   └─ executor.sol
 ├─ relayer/
 └─ docs/
```

---

## 🎯 MVP Scope

### Included

* cred 토큰 발행 / 소각
* 단순 양방향 Netting
* Oracle 투표 기반 이벤트 확정
* ECDSA Multisig Mint 명령 실행

### Explicitly Out of Scope

* 신용 한도 관리
* 디폴트 / 파산 처리
* 이자, FX, 수수료 모델
* 규제 및 법적 프레임워크

---

## 🚧 Status

> 현재: **Core Modules Implementation 완료**

Cosmos Hub의 핵심 모듈(oracle, netting, multisig)이 구현되었으며, 속성 기반 테스트가 포함되어 있습니다.

---

## 🛠 환경 설정 및 실행 방법

### 필수 요구사항

#### 1. Go 설치 (v1.21+)
```bash
# Windows (Chocolatey 사용)
choco install golang

# macOS (Homebrew 사용)
brew install go

# Linux (Ubuntu/Debian)
sudo apt update
sudo apt install golang-go

# 설치 확인
go version
```

#### 2. Node.js 설치 (v18+)
```bash
# Windows (Chocolatey 사용)
choco install nodejs

# macOS (Homebrew 사용)
brew install node

# Linux (Ubuntu/Debian)
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# 설치 확인
node --version
npm --version
```

#### 3. Docker 설치 (Besu 네트워크용)
```bash
# Windows: Docker Desktop 다운로드 및 설치
# https://www.docker.com/products/docker-desktop

# macOS (Homebrew 사용)
brew install --cask docker

# Linux (Ubuntu/Debian)
sudo apt update
sudo apt install docker.io docker-compose

# 설치 확인
docker --version
docker-compose --version
```

### 프로젝트 설정

#### 1. 저장소 클론
```bash
git clone <repository-url>
cd astra-clear
```

#### 2. 자동 개발 환경 설정 (권장)
```bash
# Linux/macOS
chmod +x setup-dev.sh
./setup-dev.sh

# Windows
setup-dev.bat
```

이 스크립트는 다음을 자동으로 수행합니다:
- 필수 도구 설치 확인 (Go, Docker, Docker Compose)
- Cosmos Hub 초기화 및 빌드
- Hyperledger Besu 네트워크 시작
- 개발 환경 준비 완료

#### 3. 수동 Cosmos Hub 설정 및 실행 (자동 설정을 사용하지 않는 경우)
```bash
# Cosmos 디렉토리로 이동
cd cosmos

# Go 모듈 의존성 설치
go mod tidy

# 바이너리 빌드
make build

# 또는 직접 빌드
go build -o build/interbank-nettingd ./cmd/interbank-nettingd

# 체인 초기화
./build/interbank-nettingd init mynode --chain-id interbank-netting

# 제네시스 계정 추가
./build/interbank-nettingd keys add validator
./build/interbank-nettingd add-genesis-account $(./build/interbank-nettingd keys show validator -a) 1000000000stake

# 제네시스 트랜잭션 생성
./build/interbank-nettingd gentx validator 1000000stake --chain-id interbank-netting

# 제네시스 파일 수집
./build/interbank-nettingd collect-gentxs

# 체인 시작
./build/interbank-nettingd start
```

#### 4. 수동 Hyperledger Besu 네트워크 실행 (자동 설정을 사용하지 않는 경우)
```bash
# 프로젝트 루트로 돌아가기
cd ..

# Besu 네트워크 시작 (Docker 사용)
# Windows
scripts/start-besu-networks.bat

# Linux/macOS
chmod +x scripts/start-besu-networks.sh
./scripts/start-besu-networks.sh

# 또는 Docker Compose 직접 사용
docker-compose -f docker/docker-compose.besu.yml up -d
```

#### 5. 스마트 컨트랙트 배포 (향후 구현)
```bash
cd contracts

# 의존성 설치
npm install

# 컨트랙트 컴파일
npx hardhat compile

# 로컬 네트워크에 배포
npx hardhat run scripts/deploy.js --network localhost
```

### 테스트 실행

#### 1. 속성 기반 테스트 (Property-Based Tests)
```bash
cd cosmos

# 모든 테스트 실행
go test ./...

# 특정 모듈 테스트
go test ./x/oracle/keeper -v
go test ./x/netting/keeper -v
go test ./x/multisig/keeper -v

# 속성 테스트만 실행
go test ./x/oracle/keeper -v -run TestProperty
go test ./x/netting/keeper -v -run TestProperty
go test ./x/multisig/keeper -v -run TestProperty
```

#### 2. 통합 테스트 (향후 구현)
```bash
# 전체 시스템 통합 테스트
make test-integration

# 특정 시나리오 테스트
make test-scenario-basic-transfer
make test-scenario-netting
```

### 개발 도구

#### 1. 코드 포맷팅
```bash
# Go 코드 포맷팅
go fmt ./...

# Solidity 코드 포맷팅 (contracts 디렉토리에서)
npx prettier --write contracts/**/*.sol
```

#### 2. 린팅
```bash
# Go 린팅
golangci-lint run

# Solidity 린팅
npx solhint contracts/**/*.sol
```

### 네트워크 상태 확인

#### 1. Cosmos Hub 상태
```bash
# 노드 상태 확인
curl http://localhost:26657/status

# 계정 잔액 확인
./build/interbank-nettingd query bank balances $(./build/interbank-nettingd keys show validator -a)

# 모듈별 상태 확인
./build/interbank-nettingd query oracle vote-status <tx-hash>
./build/interbank-nettingd query netting credit-balance <bank-id> <denom>
./build/interbank-nettingd query multisig validator-set
```

#### 2. Besu 네트워크 상태
```bash
# Bank A 네트워크 상태
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' http://localhost:8545

# Bank B 네트워크 상태
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' http://localhost:8546

# 네트워크 피어 확인
curl -X POST --data '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":1}' http://localhost:8545
```

### 트러블슈팅

#### 1. 포트 충돌
```bash
# 사용 중인 포트 확인
# Windows
netstat -ano | findstr :26657
netstat -ano | findstr :8545

# Linux/macOS
lsof -i :26657
lsof -i :8545

# 프로세스 종료 후 재시작
```

#### 2. Docker 관련 문제
```bash
# Docker 컨테이너 상태 확인
docker ps -a

# 로그 확인
docker logs <container-name>

# 컨테이너 재시작
docker-compose -f docker/docker-compose.besu.yml restart
```

#### 3. Go 모듈 문제
```bash
# 모듈 캐시 정리
go clean -modcache

# 의존성 재설치
go mod tidy
go mod download
```

### 다음 단계

1. **Relayer 구현**: Cosmos Hub와 Besu 네트워크 간 이벤트 전달
2. **스마트 컨트랙트 완성**: Gateway.sol, Executor.sol 구현
3. **통합 테스트**: 전체 시스템 End-to-End 테스트
4. **성능 최적화**: 처리량 및 지연시간 개선
5. **모니터링**: 메트릭 수집 및 대시보드 구성

---

## 📜 License

MIT License

---

## 🛰 Closing Thought

**astra-clear**는 질문에서 출발합니다.

> "은행 간 결제에서 정말로 모든 송금을 즉시 정산해야 할까?"

이 프로젝트는 그 질문에 대한 하나의 기술적 실험입니다.
