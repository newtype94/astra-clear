# Architecture

Astra Clear 시스템 아키텍처 및 흐름도

---

## 1. System Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           ASTRA CLEAR NETWORK                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────┐                              ┌─────────────────┐       │
│  │   BANK A        │                              │   BANK B        │       │
│  │   (Besu Chain)  │                              │   (Besu Chain)  │       │
│  │                 │                              │                 │       │
│  │  ┌───────────┐  │                              │  ┌───────────┐  │       │
│  │  │ Gateway   │  │         RELAYER              │  │ Gateway   │  │       │
│  │  │ Contract  │──┼──────────────────────────────┼──│ Contract  │  │       │
│  │  └───────────┘  │              │               │  └───────────┘  │       │
│  │  ┌───────────┐  │              │               │  ┌───────────┐  │       │
│  │  │ Executor  │  │              │               │  │ Executor  │  │       │
│  │  │ Contract  │◀─┼──────────────┼───────────────┼──│ Contract  │  │       │
│  │  └───────────┘  │              │               │  └───────────┘  │       │
│  │  ┌───────────┐  │              │               │  ┌───────────┐  │       │
│  │  │ BankToken │  │              │               │  │ BankToken │  │       │
│  │  └───────────┘  │              │               │  └───────────┘  │       │
│  └─────────────────┘              │               └─────────────────┘       │
│                                   │                                          │
│                          ┌────────▼────────┐                                │
│                          │   COSMOS HUB    │                                │
│                          │                 │                                │
│                          │  ┌───────────┐  │                                │
│                          │  │  Oracle   │  │                                │
│                          │  │  Module   │  │                                │
│                          │  └─────┬─────┘  │                                │
│                          │        │        │                                │
│                          │  ┌─────▼─────┐  │                                │
│                          │  │  Netting  │  │                                │
│                          │  │  Module   │  │                                │
│                          │  └─────┬─────┘  │                                │
│                          │        │        │                                │
│                          │  ┌─────▼─────┐  │                                │
│                          │  │ Multisig  │  │                                │
│                          │  │  Module   │  │                                │
│                          │  └───────────┘  │                                │
│                          └─────────────────┘                                │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Component Details

### 2.1 Cosmos Hub Modules

```
cosmos/x/
├── oracle/         # 크로스체인 이벤트 투표
├── netting/        # 신용 토큰 및 상계 처리
└── multisig/       # 서명 집계 및 검증
```

**x/oracle**
- External chain 이벤트 감시
- Validator 투표 수집
- 2/3 합의 도달 시 이벤트 확정
- Mint 명령 생성 트리거

**x/netting**
- Credit Token (IOU) 발행/소각
- Bank 간 잔액 관리
- Bilateral Netting 실행
- 상계 결과 기록

**x/multisig**
- Validator 집합 관리
- ECDSA 서명 집계
- Threshold signature 생성
- 서명 검증 로직

### 2.2 Smart Contracts

```
contracts/
├── Gateway.sol      # Source chain 진입점
├── Executor.sol     # Dest chain 실행점
└── BankToken.sol    # ERC20 스테이블코인
```

**Gateway.sol**
- 사용자 송금 요청 수신
- 토큰 Burn 처리
- TransferInitiated 이벤트 발생

**Executor.sol**
- Multi-sig 검증
- 토큰 Mint 실행
- Command 중복 처리 방지

**BankToken.sol**
- ERC20 구현
- Mint/Burn 권한 관리
- 100% 담보 기반

### 2.3 Relayer

```
relayer/src/
├── besu/
│   ├── monitor.ts    # 이벤트 감시
│   └── executor.ts   # 명령 실행
├── cosmos/
│   ├── submitter.ts  # 투표 제출
│   └── monitor.ts    # 명령 감시
└── relayer.ts        # 메인 오케스트레이터
```

- Stateless 메시지 전달
- Circuit Breaker 패턴
- Exponential Backoff 재시도

---

## 3. Data Flow

### 3.1 Cross-Chain Transfer Flow

```
User A                Bank A (Besu)              Cosmos Hub              Bank B (Besu)              User B
  │                        │                         │                        │                        │
  │  1. Transfer Request   │                         │                        │                        │
  ├───────────────────────▶│                         │                        │                        │
  │                        │                         │                        │                        │
  │                   2. Burn Token                  │                        │                        │
  │                   + Emit Event                   │                        │                        │
  │                        │                         │                        │                        │
  │                        │  3. Relayer detects     │                        │                        │
  │                        ├────────────────────────▶│                        │                        │
  │                        │     (Submit Vote)       │                        │                        │
  │                        │                         │                        │                        │
  │                        │                    4. Validators                 │                        │
  │                        │                       Vote (2/3)                 │                        │
  │                        │                         │                        │                        │
  │                        │                    5. Consensus                  │                        │
  │                        │                       Reached                    │                        │
  │                        │                         │                        │                        │
  │                        │                    6. Record IOU                 │                        │
  │                        │                    (cred-BankA)                  │                        │
  │                        │                         │                        │                        │
  │                        │                    7. Generate                   │                        │
  │                        │                    Mint Command                  │                        │
  │                        │                    + Signatures                  │                        │
  │                        │                         │                        │                        │
  │                        │                         │  8. Relayer fetches    │                        │
  │                        │                         ├───────────────────────▶│                        │
  │                        │                         │     (Execute Mint)     │                        │
  │                        │                         │                        │                        │
  │                        │                         │                   9. Verify Sigs              │
  │                        │                         │                   + Mint Token                │
  │                        │                         │                        │                        │
  │                        │                         │                        │  10. Token Received   │
  │                        │                         │                        ├───────────────────────▶│
  │                        │                         │                        │                        │
```

### 3.2 Netting Flow

```
                              COSMOS HUB
                                  │
            ┌─────────────────────┼─────────────────────┐
            │                     │                     │
            ▼                     ▼                     ▼
    ┌───────────────┐    ┌───────────────┐    ┌───────────────┐
    │   cred-A      │    │   cred-B      │    │   cred-C      │
    │   Balances    │    │   Balances    │    │   Balances    │
    │               │    │               │    │               │
    │ Bank B: 100   │    │ Bank A: 30    │    │ Bank A: 50    │
    │ Bank C: 50    │    │ Bank C: 20    │    │ Bank B: 40    │
    └───────┬───────┘    └───────┬───────┘    └───────┬───────┘
            │                     │                     │
            └─────────────────────┼─────────────────────┘
                                  │
                                  ▼
                         ┌───────────────┐
                         │   NETTING     │
                         │   PROCESS     │
                         └───────┬───────┘
                                 │
                    ┌────────────┼────────────┐
                    ▼            ▼            ▼
            ┌───────────┐ ┌───────────┐ ┌───────────┐
            │ A ↔ B     │ │ A ↔ C     │ │ B ↔ C     │
            │           │ │           │ │           │
            │ A→B: 100  │ │ A→C: 50   │ │ B→C: 40   │
            │ B→A: 30   │ │ C→A: 50   │ │ C→B: 20   │
            │ ───────── │ │ ───────── │ │ ───────── │
            │ Net: A→B  │ │ Net: 0    │ │ Net: B→C  │
            │      70   │ │           │ │      20   │
            └───────────┘ └───────────┘ └───────────┘
```

---

## 4. State Machines

### 4.1 Transfer State Machine

```
                    ┌─────────────┐
                    │   PENDING   │
                    │  (Created)  │
                    └──────┬──────┘
                           │
                    Vote submitted
                           │
                           ▼
                    ┌─────────────┐
                    │   VOTING    │
                    │  (Waiting)  │
                    └──────┬──────┘
                           │
            ┌──────────────┼──────────────┐
            │              │              │
      2/3 reached     Timeout       Rejected
            │              │              │
            ▼              ▼              ▼
     ┌───────────┐  ┌───────────┐  ┌───────────┐
     │ CONFIRMED │  │  EXPIRED  │  │ REJECTED  │
     │           │  │           │  │           │
     └─────┬─────┘  └───────────┘  └───────────┘
           │
    Mint executed
           │
           ▼
     ┌───────────┐
     │ COMPLETED │
     │           │
     └───────────┘
```

### 4.2 Netting Cycle State Machine

```
     ┌───────────────┐
     │     IDLE      │
     │ (Waiting for  │
     │  trigger)     │
     └───────┬───────┘
             │
      Trigger (block height)
             │
             ▼
     ┌───────────────┐
     │  COLLECTING   │
     │ (Gathering    │
     │  balances)    │
     └───────┬───────┘
             │
      Balances collected
             │
             ▼
     ┌───────────────┐
     │  CALCULATING  │
     │ (Computing    │
     │  net amounts) │
     └───────┬───────┘
             │
      ┌──────┴──────┐
      │             │
   Success       Error
      │             │
      ▼             ▼
┌───────────┐ ┌───────────┐
│ EXECUTING │ │ ROLLBACK  │
│           │ │           │
└─────┬─────┘ └─────┬─────┘
      │             │
      │             │
      ▼             ▼
┌───────────┐ ┌───────────┐
│ COMPLETED │ │  FAILED   │
└───────────┘ └───────────┘
```

---

## 5. Security Model

### 5.1 Trust Assumptions

```
┌─────────────────────────────────────────────────┐
│               TRUST BOUNDARY                     │
├─────────────────────────────────────────────────┤
│                                                  │
│   TRUSTED                    UNTRUSTED          │
│   ────────                   ────────           │
│                                                  │
│   • Validator Set            • External Users   │
│     (Permissioned)                              │
│                              • Network Layer    │
│   • Bank Operators                              │
│     (KYC'd)                  • Relayer          │
│                                (Stateless)      │
│   • Smart Contract                              │
│     Logic                                       │
│                                                  │
└─────────────────────────────────────────────────┘
```

### 5.2 Signature Flow

```
                         MINT COMMAND
                              │
                              ▼
         ┌─────────────────────────────────────┐
         │           MESSAGE HASH               │
         │  keccak256(commandId, recipient,     │
         │            amount, chainId)          │
         └─────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
          ▼                   ▼                   ▼
   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
   │ Validator 1 │    │ Validator 2 │    │ Validator 3 │
   │   Sign      │    │   Sign      │    │   Sign      │
   └──────┬──────┘    └──────┬──────┘    └──────┬──────┘
          │                  │                   │
          └──────────────────┼───────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │   AGGREGATED    │
                    │   SIGNATURES    │
                    │   (2/3 + 1)     │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │    EXECUTOR     │
                    │    VERIFY       │
                    │  (ecrecover)    │
                    └─────────────────┘
```

---

## 6. Network Topology

```
┌─────────────────────────────────────────────────────────────────┐
│                        PRODUCTION TOPOLOGY                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                         VALIDATOR ZONE                           │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                                                          │   │
│   │  ┌──────────┐  ┌──────────┐  ┌──────────┐              │   │
│   │  │ Val 1    │  │ Val 2    │  │ Val 3    │              │   │
│   │  │ (Bank A) │  │ (Bank B) │  │ (Bank C) │              │   │
│   │  │          │  │          │  │          │              │   │
│   │  │ Cosmos   │  │ Cosmos   │  │ Cosmos   │              │   │
│   │  │ Relayer  │  │ Relayer  │  │ Relayer  │              │   │
│   │  └────┬─────┘  └────┬─────┘  └────┬─────┘              │   │
│   │       │             │             │                     │   │
│   └───────┼─────────────┼─────────────┼─────────────────────┘   │
│           │             │             │                          │
│           └─────────────┼─────────────┘                          │
│                         │                                        │
│                         ▼                                        │
│                  ┌─────────────┐                                │
│                  │ COSMOS HUB  │                                │
│                  │ (Tendermint)│                                │
│                  └─────────────┘                                │
│                                                                  │
│   ┌──────────────────────┐    ┌──────────────────────┐         │
│   │    BANK A NETWORK    │    │    BANK B NETWORK    │         │
│   │    (Besu IBFT 2.0)   │    │    (Besu IBFT 2.0)   │         │
│   │                      │    │                      │         │
│   │  ┌────┐ ┌────┐      │    │  ┌────┐ ┌────┐      │         │
│   │  │Node│ │Node│      │    │  │Node│ │Node│      │         │
│   │  └────┘ └────┘      │    │  └────┘ └────┘      │         │
│   └──────────────────────┘    └──────────────────────┘         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 7. Error Handling

### 7.1 Recovery Mechanisms

| Layer | Error Type | Recovery |
|-------|------------|----------|
| Network | Connection timeout | Exponential backoff retry |
| Network | RPC failure | Circuit breaker + fallback |
| Consensus | Insufficient votes | Dynamic threshold |
| Consensus | Timeout | Transfer expiry |
| Netting | Calculation error | Snapshot rollback |
| Contract | Gas estimation | 20% buffer + retry |
| Contract | Signature mismatch | Validator set sync |

### 7.2 Circuit Breaker States

```
        ┌──────────────────────────────────────┐
        │                                      │
        │   CLOSED ───────────▶ OPEN          │
        │     │   (5 failures)    │            │
        │     │                   │            │
        │     │                   │ (60s)      │
        │     │                   │            │
        │     │                   ▼            │
        │     │              HALF-OPEN         │
        │     │                   │            │
        │     │      ┌────────────┤            │
        │     │      │            │            │
        │     │   Success      Failure         │
        │     │      │            │            │
        │     ◀──────┘            └─────▶ OPEN │
        │                                      │
        └──────────────────────────────────────┘
```

---

## 8. Performance Considerations

| Metric | Target | Bottleneck |
|--------|--------|------------|
| Transfer latency | < 10s | Cosmos block time (5s) |
| TPS | 100+ | Besu block gas limit |
| Netting cycle | 1 hour | Configurable |
| Signature verification | < 100ms | ecrecover |

---

## Next: [FEATURES.md](FEATURES.md) - 기능별 상세 설명
