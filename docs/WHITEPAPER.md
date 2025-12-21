# Astra Clear: Technical Whitepaper

<details>
<summary><b>🇺🇸 English</b></summary>

## Abstract

Astra Clear is a clearing engine that optimizes interbank settlements in permissioned financial institution consortiums. Traditional RTGS systems settle every transaction individually, causing high liquidity costs and processing delays. Astra Clear tokenizes interbank obligations (IOU) and minimizes actual fund movements through periodic netting. Users receive instant payments while interbank settlement is processed efficiently.

---

## 1. Introduction

### 1.1 Problem Statement

Inefficiencies in current interbank payment systems:

| Issue | Description |
|-------|-------------|
| Gross Settlement | Every transfer processed individually |
| Liquidity Lock-up | Collateral required for intraday liquidity |
| Settlement Delay | T+1 or T+2 settlement cycles |
| Operational Cost | Per-transaction fees, nostro account maintenance |

### 1.2 Proposed Solution

| Approach | Description |
|----------|-------------|
| Deferred Net Settlement | Instant user payment, netted bank settlement |
| IOU Tokenization | Transparent on-chain debt tracking |
| BFT Consensus | Cross-chain event verification |
| Permissioned Network | KYC'd financial institutions only |

---

## 2. System Design

### 2.1 Design Principles

| Principle | Description |
|-----------|-------------|
| Separation of Concerns | User payment vs bank settlement |
| Atomicity | Cross-chain transfer guarantees |
| Finality | Immediate BFT consensus |
| Auditability | All state changes tracked |
| Fault Tolerance | Partial failure resilience |

### 2.2 Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     ASTRA CLEAR                          │
├─────────────────────────────────────────────────────────┤
│  APPLICATION LAYER                                       │
│  • User Interface, Admin Dashboard                      │
│                                                          │
│  COORDINATION LAYER (Cosmos Hub)                        │
│  • Oracle - Event voting                                │
│  • Netting - IOU management                             │
│  • Multisig - Signature aggregation                     │
│                                                          │
│  EXECUTION LAYER (Hyperledger Besu)                     │
│  • Gateway - Transfer initiation                        │
│  • Executor - Mint execution                            │
│  • BankToken - Stablecoin                               │
│                                                          │
│  TRANSPORT LAYER                                         │
│  • Relayer Service                                       │
└─────────────────────────────────────────────────────────┘
```

---

## 3. IOU Token Model

### 3.1 Concept

```
Token: cred-BANK_A
Meaning: "Bank A owes the holder this amount"
Value: 1 cred = 1 Stablecoin Unit
```

### 3.2 Properties

| Property | Description |
|----------|-------------|
| Issuer-specific | Separate token per bank |
| Fungible | Same issuer tokens interchangeable |
| Burnable | Destroyed during netting |
| Non-transferable | Only netted, not traded |

### 3.3 Lifecycle

```
1. ISSUANCE    - User A→B transfer creates cred-A for B
2. ACCUMULATION - Multiple transfers accumulate balances
3. NETTING     - Mutual obligations offset and burn
4. SETTLEMENT  - Net obligations settled externally
```

---

## 4. Cross-Chain Transfer Protocol

### 4.1 Protocol Phases

**Phase 1: Initiation**
- User calls Gateway.initiateTransfer()
- Tokens burned, event emitted
- Relayer detects event

**Phase 2: Consensus**
- Relayer submits vote to Oracle
- Validators verify and vote
- 2/3 consensus confirms transfer
- IOU recorded, mint command generated

**Phase 3: Execution**
- Relayer fetches mint command
- Executor verifies signatures
- Tokens minted to recipient

### 4.2 Security Guarantees

| Property | Mechanism |
|----------|-----------|
| No double-spend | processedCommands mapping |
| Authenticity | 2/3 validator signatures |
| Non-repudiation | On-chain event records |
| Atomicity | State rollback on failure |

---

## 5. Netting Mechanism

### 5.1 Bilateral Netting

```
Before:
  A → B: 100, B → A: 30
  Gross: 130

After:
  Burn min(100,30) = 30 each
  Net: A → B: 70
  Reduction: 46%
```

### 5.2 Netting Efficiency

| Metric | Without | With | Reduction |
|--------|---------|------|-----------|
| Gross | 1,000,000 | - | - |
| Net | - | 400,000 | 60% |
| Txns | 20 | 10 | 50% |

---

## 6. Consensus Mechanism

### 6.1 Threshold Calculation

```
threshold = (validatorCount * 2 + 2) / 3

3 validators: 3 required (100%)
5 validators: 4 required (80%)
7 validators: 5 required (71%)
```

### 6.2 Signature Scheme

ECDSA (secp256k1):
```
1. hash = keccak256(commandId || recipient || amount || chainId)
2. ethHash = "\x19Ethereum Signed Message:\n32" || hash
3. sig = secp256k1_sign(ethHash, privateKey)
4. verify: ecrecover(ethHash, sig) == validator
```

---

## 7. Security Considerations

### 7.1 Trust Assumptions

| Trusted | Untrusted |
|---------|-----------|
| Validator Set | External Users |
| Bank Operators | Network Layer |
| Smart Contracts | Relayer |

### 7.2 Attack Mitigation

| Attack | Defense |
|--------|---------|
| Validator Collusion | Permissioned set, audit logs |
| Replay Attack | Chain ID in message hash |
| Front-running | FIFO processing |
| DoS | Rate limiting, permissioned access |

---

## 8. Performance

### 8.1 Latency

| Phase | Latency |
|-------|---------|
| Source Chain | 2-5s |
| Relayer | 1-2s |
| Cosmos Voting | 5-10s |
| Dest Chain | 2-5s |
| **Total** | **11-24s** |

### 8.2 Throughput

| Component | Capacity |
|-----------|----------|
| Besu | ~1000 TPS |
| Cosmos | ~10,000 TPS |

---

## 9. Comparison

### vs RTGS

| Aspect | RTGS | Astra Clear |
|--------|------|-------------|
| Settlement | Per-txn | Net basis |
| Liquidity | High | Reduced |
| Latency | Minutes-hours | ~15 seconds |

### vs Correspondent Banking

| Aspect | Correspondent | Astra Clear |
|--------|---------------|-------------|
| Intermediaries | Multiple | None |
| Settlement | T+1 to T+3 | Near real-time |

---

## 10. Conclusion

Astra Clear demonstrates a technical approach to optimizing interbank settlements through IOU tokenization and bilateral netting. This POC validates the concept; production deployment requires additional review.

---

## Appendix: Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| block_time | 5s | Cosmos block interval |
| voting_timeout | 100 blocks | Vote expiration |
| netting_interval | 720 blocks | ~1 hour cycle |
| threshold_ratio | 2/3 | Signature threshold |
| retry_max | 5 | Max retries |

</details>

<details open>
<summary><b>🇰🇷 한국어</b></summary>

## 개요

Astra Clear는 허가형 금융기관 컨소시엄 환경에서 은행 간 결제를 효율화하는 청산 엔진이다. 기존 RTGS 시스템은 모든 거래를 개별 정산하여 높은 유동성 비용과 처리 지연을 유발한다. Astra Clear는 은행 간 채권/채무를 토큰화(IOU)하고 주기적 상계(Netting)를 통해 실제 자금 이동을 최소화한다. 사용자에게는 즉시 지급을 제공하면서 은행 간 정산은 효율적으로 처리한다.

---

## 1. 서론

### 1.1 문제 정의

현행 은행 간 결제 시스템의 비효율:

| 문제 | 설명 |
|------|------|
| 개별 정산 | 모든 송금건 개별 처리 |
| 유동성 잠김 | 일중 유동성 담보 요구 |
| 정산 지연 | T+1 또는 T+2 사이클 |
| 운영 비용 | 건당 수수료, 노스트로 유지비 |

### 1.2 제안 솔루션

| 접근법 | 설명 |
|--------|------|
| 지연 순정산 | 사용자 즉시 지급, 은행 간 Netting 처리 |
| IOU 토큰화 | 투명한 온체인 채무 추적 |
| BFT 합의 | 크로스체인 이벤트 검증 |
| 허가형 네트워크 | KYC된 금융기관만 참여 |

---

## 2. 시스템 설계

### 2.1 설계 원칙

| 원칙 | 설명 |
|------|------|
| 관심사 분리 | 사용자 지급 vs 은행 정산 |
| 원자성 | 크로스체인 전송 보장 |
| 완결성 | 즉시 BFT 합의 |
| 감사 가능성 | 모든 상태 변경 추적 |
| 장애 허용 | 부분 장애 복원력 |

### 2.2 아키텍처

```
┌─────────────────────────────────────────────────────────┐
│                     ASTRA CLEAR                          │
├─────────────────────────────────────────────────────────┤
│  애플리케이션 계층                                       │
│  • 사용자 인터페이스, 관리자 대시보드                   │
│                                                          │
│  조정 계층 (Cosmos Hub)                                  │
│  • Oracle - 이벤트 투표                                 │
│  • Netting - IOU 관리                                   │
│  • Multisig - 서명 집계                                 │
│                                                          │
│  실행 계층 (Hyperledger Besu)                           │
│  • Gateway - 전송 시작                                  │
│  • Executor - Mint 실행                                 │
│  • BankToken - 스테이블코인                             │
│                                                          │
│  전송 계층                                               │
│  • Relayer 서비스                                        │
└─────────────────────────────────────────────────────────┘
```

---

## 3. IOU 토큰 모델

### 3.1 개념

```
토큰: cred-BANK_A
의미: "Bank A가 보유자에게 갚아야 할 금액"
가치: 1 cred = 1 스테이블코인 단위
```

### 3.2 속성

| 속성 | 설명 |
|------|------|
| 발행자별 분리 | 은행별 독립 토큰 |
| 대체 가능 | 동일 발행자 토큰 교환 가능 |
| 소각 가능 | Netting 시 소각 |
| 양도 불가 | 거래 불가, Netting만 가능 |

### 3.3 생명주기

```
1. 발행     - 사용자 A→B 송금 시 B에게 cred-A 발행
2. 누적     - 여러 거래로 잔액 누적
3. 상계     - 상호 채무 상계 후 소각
4. 정산     - 순 채무에 대해 외부 정산
```

---

## 4. 크로스체인 전송 프로토콜

### 4.1 프로토콜 단계

**1단계: 시작**
- 사용자가 Gateway.initiateTransfer() 호출
- 토큰 소각, 이벤트 발생
- Relayer가 이벤트 감지

**2단계: 합의**
- Relayer가 Oracle에 투표 제출
- Validator들이 검증 후 투표
- 2/3 합의 시 전송 확정
- IOU 기록, Mint 명령 생성

**3단계: 실행**
- Relayer가 Mint 명령 가져옴
- Executor가 서명 검증
- 수신자에게 토큰 발행

### 4.2 보안 보장

| 속성 | 메커니즘 |
|------|----------|
| 이중 지불 방지 | processedCommands 맵 |
| 진위성 | 2/3 Validator 서명 |
| 부인 방지 | 온체인 이벤트 기록 |
| 원자성 | 실패 시 상태 롤백 |

---

## 5. Netting 메커니즘

### 5.1 양방향 Netting

```
전:
  A → B: 100, B → A: 30
  총 채무: 130

후:
  min(100,30) = 30씩 소각
  순 채무: A → B: 70
  감소율: 46%
```

### 5.2 Netting 효율

| 지표 | Netting 전 | Netting 후 | 감소율 |
|------|-----------|-----------|--------|
| 총 채무 | 1,000,000 | - | - |
| 순 채무 | - | 400,000 | 60% |
| 거래 수 | 20 | 10 | 50% |

---

## 6. 합의 메커니즘

### 6.1 임계값 계산

```
threshold = (validatorCount * 2 + 2) / 3

3명: 3명 필요 (100%)
5명: 4명 필요 (80%)
7명: 5명 필요 (71%)
```

### 6.2 서명 체계

ECDSA (secp256k1):
```
1. hash = keccak256(commandId || recipient || amount || chainId)
2. ethHash = "\x19Ethereum Signed Message:\n32" || hash
3. sig = secp256k1_sign(ethHash, privateKey)
4. 검증: ecrecover(ethHash, sig) == validator
```

---

## 7. 보안 고려사항

### 7.1 신뢰 가정

| 신뢰됨 | 비신뢰 |
|--------|--------|
| Validator 집합 | 외부 사용자 |
| 은행 운영자 | 네트워크 계층 |
| 스마트 컨트랙트 | Relayer |

### 7.2 공격 대응

| 공격 | 방어 |
|------|------|
| Validator 공모 | 허가형 집합, 감사 로그 |
| 재전송 공격 | 메시지 해시에 Chain ID 포함 |
| 프론트러닝 | FIFO 처리 |
| DoS | 속도 제한, 허가형 접근 |

---

## 8. 성능

### 8.1 지연시간

| 단계 | 지연시간 |
|------|----------|
| Source Chain | 2-5초 |
| Relayer | 1-2초 |
| Cosmos 투표 | 5-10초 |
| Dest Chain | 2-5초 |
| **총** | **11-24초** |

### 8.2 처리량

| 컴포넌트 | 용량 |
|----------|------|
| Besu | ~1000 TPS |
| Cosmos | ~10,000 TPS |

---

## 9. 비교

### vs RTGS

| 항목 | RTGS | Astra Clear |
|------|------|-------------|
| 정산 | 건별 | 순 기준 |
| 유동성 | 높음 | 감소 |
| 지연시간 | 분~시간 | ~15초 |

### vs 환거래 은행

| 항목 | 환거래 | Astra Clear |
|------|--------|-------------|
| 중개자 | 다수 | 없음 |
| 정산 | T+1~T+3 | 준실시간 |

---

## 10. 결론

Astra Clear는 IOU 토큰화와 양방향 Netting을 통해 은행 간 정산을 최적화하는 기술적 접근을 시연한다. 본 POC는 개념을 검증하며, 프로덕션 배포 시 추가 검토가 필요하다.

---

## 부록: 파라미터

| 파라미터 | 기본값 | 설명 |
|----------|--------|------|
| block_time | 5초 | Cosmos 블록 주기 |
| voting_timeout | 100 블록 | 투표 만료 |
| netting_interval | 720 블록 | ~1시간 사이클 |
| threshold_ratio | 2/3 | 서명 임계값 |
| retry_max | 5 | 최대 재시도 |

</details>
