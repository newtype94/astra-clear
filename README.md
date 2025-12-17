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

> 현재: **Concept & Architecture POC 단계**

이 프로젝트는 실험적이며, 학습과 검증을 목적으로 합니다.

---

## 📜 License

MIT License

---

## 🛰 Closing Thought

**astra-clear**는 질문에서 출발합니다.

> "은행 간 결제에서 정말로 모든 송금을 즉시 정산해야 할까?"

이 프로젝트는 그 질문에 대한 하나의 기술적 실험입니다.
