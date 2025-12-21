# Astra Clear

Interbank Netting & Clearing Engine for Permissioned Stablecoin Networks

<details>
<summary><b>🇺🇸 English</b></summary>

## Overview

Astra Clear is a clearing engine that optimizes interbank settlements in permissioned financial institution consortiums. The core concept is simple:

> **Instant payment to users, minimize interbank settlement through netting**

Traditional payment systems settle every transaction individually. Astra Clear tokenizes interbank obligations (IOU) and compresses actual fund movements through netting.

---

## Problem Statement

Inefficiencies in current interbank payment systems:

| Problem | Description |
|---------|-------------|
| Gross Settlement | Every transfer processed individually |
| Liquidity Lock-up | Collateral required for intraday liquidity |
| Delay | T+1 or T+2 settlement cycles |
| Cost | Per-transaction fees, nostro account maintenance |

---

## Solution

```
┌─────────────────────────────────────────────────────────────────┐
│                      Astra Clear Architecture                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│   Bank A Network          Cosmos Hub           Bank B Network    │
│   (Hyperledger Besu)      (Coordinator)        (Hyperledger Besu)│
│                                                                   │
│   ┌──────────┐           ┌───────────┐         ┌──────────┐     │
│   │ Gateway  │──Events──▶│  Oracle   │         │ Gateway  │     │
│   │ Contract │           │  Module   │         │ Contract │     │
│   └──────────┘           └─────┬─────┘         └──────────┘     │
│                                │                                  │
│   ┌──────────┐           ┌─────▼─────┐         ┌──────────┐     │
│   │ Executor │◀─Commands─│  Netting  │─────────│ Executor │     │
│   │ Contract │           │  Module   │         │ Contract │     │
│   └──────────┘           └───────────┘         └──────────┘     │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

**Flow:**

1. User requests transfer from Bank A to Bank B recipient
2. Gateway on Bank A burns tokens + emits event
3. Validators detect event and vote on Cosmos Hub
4. Upon 2/3 consensus, mint command is signed for Bank B
5. Executor on Bank B mints tokens to recipient instantly
6. Interbank obligations are netted on Cosmos Hub

---

## Key Features

| Feature | Description |
|---------|-------------|
| **Bilateral Netting** | Mutual obligation offset reduces settlement count |
| **IOU Token Model** | Issuer-based debt tokens (`cred-{BankID}`) |
| **Oracle Consensus** | BFT-based cross-chain event verification |
| **Multi-Signature Execution** | 2/3 validator signatures for mint execution |
| **Atomic Cross-Chain Transfer** | Instant transfer from user perspective |

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Settlement Hub | Cosmos SDK (Go) |
| Bank Networks | Hyperledger Besu (EVM) |
| Smart Contracts | Solidity 0.8.24 |
| Relayer | TypeScript |
| Signing | ECDSA (secp256k1) |

---

## Repository Structure

```
astra-clear/
├── cosmos/                 # Cosmos SDK Hub
│   ├── x/oracle/          # Cross-chain event voting
│   ├── x/netting/         # Credit token & netting logic
│   └── x/multisig/        # Validator signature aggregation
├── contracts/             # Solidity smart contracts
│   ├── Gateway.sol        # Source chain burn & event
│   ├── Executor.sol       # Dest chain signature verify & mint
│   └── BankToken.sol      # ERC20 stablecoin implementation
├── relayer/               # Event relay service
└── docs/                  # Documentation
```

---

## Quick Start

```bash
# Clone
git clone https://github.com/[org]/astra-clear.git
cd astra-clear

# Cosmos Hub
cd cosmos && go mod tidy && make build

# Smart Contracts
cd ../contracts && npm install && npx hardhat compile

# Run Tests
cd ../cosmos && go test ./...
cd ../contracts && npx hardhat test
```

See [docs/DEVELOPER_GUIDE.md](docs/DEVELOPER_GUIDE.md) for detailed setup instructions.

---

## Documentation

| Document | Description |
|----------|-------------|
| [DEVELOPER_GUIDE.md](docs/DEVELOPER_GUIDE.md) | Installation, build, run guide |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | System architecture & flow diagrams |
| [WHITEPAPER.md](docs/WHITEPAPER.md) | Design principles & technical specs |
| [FEATURES.md](docs/FEATURES.md) | Feature-by-feature explanation |

---

## Project Status

| Component | Status |
|-----------|--------|
| Cosmos x/oracle | Implemented |
| Cosmos x/netting | Implemented |
| Cosmos x/multisig | Implemented |
| Gateway.sol | Implemented |
| Executor.sol | Implemented |
| Relayer | Implemented |
| Property-Based Tests | 100+ test cases |
| Integration Tests | In Progress |

---

## Scope & Limitations

**In Scope (MVP)**
- 100% collateralized stablecoin environment
- Permissioned financial institution consortium
- Bilateral netting
- ECDSA-based signature verification

**Out of Scope**
- Credit limit management
- Default/bankruptcy handling
- Multilateral netting
- Interest/FX/fee models
- Regulatory framework

---

## License

MIT License

---

## Contact

This project is a POC for technical validation.
Separate review is required for production deployment.

</details>

<details open>
<summary><b>🇰🇷 한국어</b></summary>

## 개요

Astra Clear는 허가형 금융기관 컨소시엄 환경에서 은행 간 결제를 효율화하는 청산 엔진이다. 핵심 개념은 단순하다:

> **사용자에게는 즉시 지급, 은행 간 정산은 Netting으로 최소화**

기존 결제 시스템은 모든 거래를 개별 정산한다. Astra Clear는 은행 간 채권/채무를 토큰화하고, 상계(Netting)를 통해 실제 자금 이동을 압축한다.

---

## 문제 정의

현행 은행 간 결제 구조의 비효율:

| 문제 | 설명 |
|------|------|
| 개별 정산 | 모든 송금건이 RTGS/대외계로 개별 처리 |
| 유동성 잠김 | 일중 유동성 확보를 위한 담보 묶임 |
| 지연 | T+1 또는 T+2 정산 사이클 |
| 비용 | 건당 수수료, 노스트로 계좌 유지비용 |

---

## 솔루션

```
┌─────────────────────────────────────────────────────────────────┐
│                      Astra Clear Architecture                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│   Bank A Network          Cosmos Hub           Bank B Network    │
│   (Hyperledger Besu)      (Coordinator)        (Hyperledger Besu)│
│                                                                   │
│   ┌──────────┐           ┌───────────┐         ┌──────────┐     │
│   │ Gateway  │──Events──▶│  Oracle   │         │ Gateway  │     │
│   │ Contract │           │  Module   │         │ Contract │     │
│   └──────────┘           └─────┬─────┘         └──────────┘     │
│                                │                                  │
│   ┌──────────┐           ┌─────▼─────┐         ┌──────────┐     │
│   │ Executor │◀─Commands─│  Netting  │─────────│ Executor │     │
│   │ Contract │           │  Module   │         │ Contract │     │
│   └──────────┘           └───────────┘         └──────────┘     │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

**동작 흐름:**

1. 사용자가 Bank A에서 Bank B 수신자에게 송금 요청
2. Bank A의 Gateway에서 토큰 Burn + 이벤트 발생
3. Validator들이 이벤트를 감지하고 Cosmos Hub에 투표
4. 2/3 합의 도달 시 Bank B로 Mint 명령 서명
5. Bank B의 Executor가 수신자에게 즉시 Mint
6. 은행 간 채무는 Cosmos Hub에서 Netting 처리

---

## 주요 기능

| 기능 | 설명 |
|------|------|
| **Bilateral Netting** | 양방향 채무 상계로 정산 건수 감소 |
| **IOU Token Model** | 발행자 기준 부채 토큰 (`cred-{BankID}`) |
| **Oracle Consensus** | BFT 기반 크로스체인 이벤트 검증 |
| **Multi-Signature Execution** | 2/3 Validator 서명으로 Mint 명령 실행 |
| **Atomic Cross-Chain Transfer** | 사용자 관점 즉시 송금 완료 |

---

## 기술 스택

| Layer | Technology |
|-------|------------|
| Settlement Hub | Cosmos SDK (Go) |
| Bank Networks | Hyperledger Besu (EVM) |
| Smart Contracts | Solidity 0.8.24 |
| Relayer | TypeScript |
| Signing | ECDSA (secp256k1) |

---

## 저장소 구조

```
astra-clear/
├── cosmos/                 # Cosmos SDK Hub
│   ├── x/oracle/          # 크로스체인 이벤트 투표
│   ├── x/netting/         # 신용 토큰 및 상계 로직
│   └── x/multisig/        # Validator 서명 집계
├── contracts/             # Solidity 스마트 컨트랙트
│   ├── Gateway.sol        # Source 체인 Burn + 이벤트
│   ├── Executor.sol       # Dest 체인 서명 검증 + Mint
│   └── BankToken.sol      # ERC20 스테이블코인
├── relayer/               # 이벤트 릴레이 서비스
└── docs/                  # 문서
```

---

## 빠른 시작

```bash
# Clone
git clone https://github.com/[org]/astra-clear.git
cd astra-clear

# Cosmos Hub
cd cosmos && go mod tidy && make build

# Smart Contracts
cd ../contracts && npm install && npx hardhat compile

# Run Tests
cd ../cosmos && go test ./...
cd ../contracts && npx hardhat test
```

상세 설치 및 실행 가이드: [docs/DEVELOPER_GUIDE.md](docs/DEVELOPER_GUIDE.md)

---

## 문서

| 문서 | 설명 |
|------|------|
| [DEVELOPER_GUIDE.md](docs/DEVELOPER_GUIDE.md) | 설치, 빌드, 실행 가이드 |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | 시스템 아키텍처 및 흐름도 |
| [WHITEPAPER.md](docs/WHITEPAPER.md) | 설계 원칙 및 기술 명세 |
| [FEATURES.md](docs/FEATURES.md) | 기능별 상세 설명 |

---

## 프로젝트 상태

| 컴포넌트 | 상태 |
|----------|------|
| Cosmos x/oracle | 구현 완료 |
| Cosmos x/netting | 구현 완료 |
| Cosmos x/multisig | 구현 완료 |
| Gateway.sol | 구현 완료 |
| Executor.sol | 구현 완료 |
| Relayer | 구현 완료 |
| Property-Based Tests | 100+ 테스트 케이스 |
| Integration Tests | 진행 중 |

---

## 범위 및 제한사항

**In Scope (MVP)**
- 100% 담보 스테이블코인 환경
- 허가형 금융기관 컨소시엄
- 양방향 Netting (Bilateral)
- ECDSA 기반 서명 검증

**Out of Scope**
- 신용 한도 관리
- 파산/디폴트 처리
- 다자간 Netting (Multilateral)
- 이자/FX/수수료 모델
- 규제 프레임워크

---

## 라이센스

MIT License

---

## 연락처

이 프로젝트는 기술 검증 목적의 POC입니다.
프로덕션 환경 적용 시 별도 검토가 필요합니다.

</details>
