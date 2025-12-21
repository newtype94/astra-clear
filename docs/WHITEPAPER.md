# Astra Clear: Technical Whitepaper

Interbank Netting & Clearing Engine for Permissioned Stablecoin Networks

---

## Abstract

Astra Clear는 허가형 금융기관 컨소시엄 환경에서 은행 간 결제를 효율화하는 청산 엔진이다. 기존 RTGS(Real-Time Gross Settlement) 시스템은 모든 거래를 개별 정산하여 높은 유동성 비용과 처리 지연을 유발한다. Astra Clear는 은행 간 채권/채무를 토큰화(IOU)하고 주기적 상계(Netting)를 통해 실제 자금 이동을 최소화한다. 사용자에게는 즉시 지급을 제공하면서 은행 간 정산은 효율적으로 처리한다.

---

## 1. Introduction

### 1.1 Problem Statement

현행 은행 간 결제 시스템의 비효율:

1. **Gross Settlement Overhead**
   - 모든 송금건이 개별 처리됨
   - 은행 A→B 100건, B→A 80건이 있어도 180건 모두 정산

2. **Liquidity Lock-up**
   - 일중 유동성 확보를 위한 담보 요구
   - 노스트로/보스트로 계좌 유지 비용

3. **Settlement Delay**
   - 사용자 관점 즉시 송금 기대
   - 실제 정산은 T+1 또는 T+2

4. **Operational Cost**
   - 건당 수수료
   - SWIFT/대외계 인프라 비용

### 1.2 Proposed Solution

Astra Clear의 접근:

1. **Deferred Net Settlement (DNS)**
   - 사용자에게는 즉시 지급 (토큰 Mint)
   - 은행 간 정산은 Netting 후 처리

2. **IOU Tokenization**
   - 은행 간 채무를 블록체인 토큰으로 표현
   - 투명한 잔액 추적 및 감사

3. **BFT Consensus**
   - 크로스체인 이벤트 검증
   - 2/3 Validator 합의

4. **Permissioned Network**
   - KYC된 금융기관만 참여
   - 규제 준수 용이

---

## 2. System Design

### 2.1 Design Principles

| Principle | Description |
|-----------|-------------|
| **Separation of Concerns** | 사용자 지급과 은행 간 정산 분리 |
| **Atomicity** | 크로스체인 전송의 원자성 보장 |
| **Finality** | BFT 합의를 통한 즉시 완결성 |
| **Auditability** | 모든 상태 변경 추적 가능 |
| **Fault Tolerance** | 부분 장애 시에도 시스템 운영 |

### 2.2 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         ASTRA CLEAR                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   APPLICATION LAYER                                             │
│   ─────────────────                                             │
│   • User Interface (Bank Apps)                                  │
│   • Admin Dashboard                                             │
│                                                                  │
│   COORDINATION LAYER (Cosmos Hub)                               │
│   ───────────────────────────────                               │
│   • Oracle Module - Cross-chain event voting                    │
│   • Netting Module - IOU management & settlement                │
│   • Multisig Module - Signature aggregation                     │
│                                                                  │
│   EXECUTION LAYER (Hyperledger Besu)                            │
│   ──────────────────────────────────                            │
│   • Gateway Contract - Transfer initiation                      │
│   • Executor Contract - Mint execution                          │
│   • BankToken Contract - Stablecoin                             │
│                                                                  │
│   TRANSPORT LAYER                                               │
│   ───────────────                                               │
│   • Relayer Service - Event relay                               │
│   • WebSocket/RPC Connections                                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.3 Component Responsibilities

**Cosmos Hub (Coordination)**
- 크로스체인 이벤트 합의
- 은행 간 채무 기록
- Netting 실행 및 기록
- Multi-sig 집계

**Hyperledger Besu (Execution)**
- 사용자 토큰 관리
- 송금 시작 (Burn)
- 수신 완료 (Mint)
- 서명 검증

**Relayer (Transport)**
- 이벤트 감지 및 전달
- Stateless 설계
- 장애 복구 지원

---

## 3. IOU Token Model

### 3.1 Concept

IOU(I Owe You) 토큰은 발행 은행의 채무를 나타낸다.

```
Token: cred-BANK_A
Meaning: "Bank A owes the holder this amount"
Value: 1 cred-BANK_A = 1 Stablecoin Unit
```

### 3.2 Properties

| Property | Description |
|----------|-------------|
| **Issuer-specific** | 각 은행별 독립 토큰 |
| **Fungible** | 동일 발행자 토큰은 대체 가능 |
| **Burnable** | Netting 시 소각 |
| **Non-transferable** | 은행 간 직접 이전 불가 (Netting만 가능) |

### 3.3 Lifecycle

```
1. ISSUANCE
   ─────────
   사용자가 Bank A → Bank B 송금 시
   Bank B에게 cred-A 발행

2. ACCUMULATION
   ─────────────
   여러 거래에 따라 잔액 누적
   Bank B holds: cred-A 100, cred-C 50

3. NETTING
   ────────
   상호 채무 상계
   cred-A 30 + cred-B 30 → Burn (상계)

4. SETTLEMENT
   ───────────
   순 채무에 대해 실제 정산
   Net: Bank A → Bank B = 70
```

### 3.4 vs Central Liquidity Pool

| Aspect | IOU Model | Central Pool |
|--------|-----------|--------------|
| Counterparty Risk | Issuer-specific | Pool operator |
| Transparency | On-chain tracking | Opaque |
| Netting | Bilateral | N/A |
| Liquidity | Distributed | Concentrated |

---

## 4. Cross-Chain Transfer Protocol

### 4.1 Protocol Steps

**Phase 1: Initiation (Source Chain)**
```
1. User calls Gateway.initiateTransfer(recipient, amount, destChain)
2. Gateway burns user's tokens
3. Gateway emits TransferInitiated event
4. Relayer detects event
```

**Phase 2: Consensus (Cosmos Hub)**
```
5. Relayer submits vote to Oracle module
6. Validators independently verify and vote
7. Upon 2/3 consensus, transfer is confirmed
8. Netting module records IOU (cred-sourceBank to destBank)
9. Multisig module generates mint command with signatures
```

**Phase 3: Execution (Destination Chain)**
```
10. Relayer fetches mint command
11. Relayer calls Executor.executeMint with signatures
12. Executor verifies signatures (threshold check)
13. Executor mints tokens to recipient
14. Executor marks command as processed
```

### 4.2 Message Format

**TransferInitiated Event**
```solidity
event TransferInitiated(
    bytes32 indexed transferId,
    address indexed sender,
    address recipient,
    uint256 amount,
    string destChain
);
```

**MintCommand**
```go
type MintCommand struct {
    CommandId   string
    Recipient   string
    Amount      math.Int
    TargetChain string
    Signatures  [][]byte
}
```

### 4.3 Security Guarantees

| Property | Mechanism |
|----------|-----------|
| **No double-spend** | processedCommands mapping |
| **Authenticity** | 2/3 validator signatures |
| **Non-repudiation** | On-chain event records |
| **Atomicity** | State rollback on failure |

---

## 5. Netting Mechanism

### 5.1 Bilateral Netting

두 은행 간 상호 채무를 상계:

```
Before:
  Bank A → Bank B: 100 (cred-A held by B)
  Bank B → Bank A:  30 (cred-B held by A)

Netting:
  Burn min(100, 30) = 30 from each

After:
  Net: Bank A → Bank B: 70
  Gross reduction: 130 → 70 (46%)
```

### 5.2 Netting Cycle

```
┌─────────────────────────────────────────┐
│             NETTING CYCLE               │
├─────────────────────────────────────────┤
│                                         │
│  Trigger: Every N blocks (e.g., 720)   │
│                                         │
│  1. Identify all bank pairs with       │
│     mutual obligations                  │
│                                         │
│  2. For each pair (A, B):              │
│     - Get cred-A balance of B          │
│     - Get cred-B balance of A          │
│     - Calculate net = min(A→B, B→A)    │
│     - Burn net amount from each        │
│                                         │
│  3. Record netting results             │
│     - Cycle ID                         │
│     - Pairs processed                  │
│     - Total netted amount              │
│                                         │
└─────────────────────────────────────────┘
```

### 5.3 Netting Efficiency

가정: 5개 은행, 각 쌍 간 양방향 거래

| Metric | Without Netting | With Netting | Reduction |
|--------|-----------------|--------------|-----------|
| Gross obligations | 1,000,000 | - | - |
| Net obligations | - | 400,000 | 60% |
| Settlement txns | 20 | 10 | 50% |

### 5.4 Rollback Mechanism

Netting 실패 시 원자적 롤백:

```go
func ExecuteNettingWithRollback(ctx, pairs) error {
    // 1. Create snapshot
    snapshot := CreateNettingSnapshot(ctx, pairs)

    // 2. Execute netting
    err := ExecuteBilateralNetting(ctx, pairs)

    // 3. Rollback on error
    if err != nil {
        RollbackNetting(ctx, snapshot)
        return err
    }

    return nil
}
```

---

## 6. Consensus Mechanism

### 6.1 Oracle Voting

```
┌─────────────────────────────────────────┐
│           VOTE AGGREGATION              │
├─────────────────────────────────────────┤
│                                         │
│  Transfer: 0xabc...                     │
│                                         │
│  Validator Votes:                       │
│  ├── V1: ✓ (block 100)                 │
│  ├── V2: ✓ (block 101)                 │
│  ├── V3: ✓ (block 101)                 │
│  ├── V4: ○ (pending)                   │
│  └── V5: ○ (pending)                   │
│                                         │
│  Threshold: 2/3 + 1 = 4/5              │
│  Current: 3/5                          │
│  Status: PENDING                        │
│                                         │
└─────────────────────────────────────────┘
```

### 6.2 Threshold Calculation

```
threshold = (validatorCount * 2 + 2) / 3

Examples:
- 3 validators: (3*2+2)/3 = 3 (100%)
- 5 validators: (5*2+2)/3 = 4 (80%)
- 7 validators: (7*2+2)/3 = 5 (71%)
```

### 6.3 Dynamic Threshold

오프라인 Validator 처리:

```go
func GetDynamicThreshold(ctx) (threshold, activeCount) {
    validators := GetAllValidators(ctx)
    activeCount := 0

    for _, v := range validators {
        if IsValidatorActive(ctx, v) {
            activeCount++
        }
    }

    threshold = (activeCount * 2 + 2) / 3
    return threshold, activeCount
}
```

### 6.4 Signature Scheme

ECDSA (secp256k1) 기반:

```
1. Message Construction
   ─────────────────────
   messageHash = keccak256(commandId || recipient || amount || chainId)

2. Ethereum Signed Message
   ────────────────────────
   ethHash = keccak256("\x19Ethereum Signed Message:\n32" || messageHash)

3. Signature Generation
   ─────────────────────
   sig = secp256k1_sign(ethHash, validatorPrivateKey)

4. Signature Verification
   ───────────────────────
   recoveredAddr = ecrecover(ethHash, sig)
   require(validators[recoveredAddr] == true)
```

---

## 7. Security Considerations

### 7.1 Threat Model

| Threat | Mitigation |
|--------|------------|
| **Byzantine Validators** | 2/3 threshold |
| **Double Spending** | Command ID tracking |
| **Replay Attack** | Chain-specific command ID |
| **Signature Forgery** | ECDSA + ecrecover |
| **Network Partition** | Timeout + dynamic threshold |

### 7.2 Trust Assumptions

```
TRUSTED:
├── Validator Set (permissioned, KYC'd)
├── Smart Contract Logic (audited)
└── Cryptographic Primitives (secp256k1)

UNTRUSTED:
├── Network Layer (public internet)
├── Relayer (stateless, replaceable)
└── External Users (public access)
```

### 7.3 Attack Vectors & Defenses

**1. Validator Collusion**
- Risk: 2/3 validators collude to forge transfers
- Defense: Permissioned set, economic incentives, audit logs

**2. Replay Attack**
- Risk: Reuse of valid signatures on different chain
- Defense: Chain ID included in message hash

**3. Front-running**
- Risk: Validator extracts value by reordering
- Defense: FIFO processing, no MEV extraction possible

**4. Denial of Service**
- Risk: Flood network with invalid requests
- Defense: Rate limiting, gas costs, permissioned access

---

## 8. Performance Analysis

### 8.1 Latency Breakdown

| Phase | Latency | Notes |
|-------|---------|-------|
| Source Chain | 2-5s | Besu IBFT block time |
| Relayer Detection | 1-2s | Event polling |
| Cosmos Voting | 5-10s | BFT consensus |
| Relayer Fetch | 1-2s | Command polling |
| Dest Chain | 2-5s | Besu IBFT block time |
| **Total** | **11-24s** | End-to-end |

### 8.2 Throughput

| Bottleneck | Capacity |
|------------|----------|
| Besu Block | 30M gas / 5s = ~1000 TPS |
| Cosmos Hub | ~10,000 TPS (Tendermint) |
| Relayer | Limited by RPC connections |

### 8.3 Scalability

| Dimension | Approach |
|-----------|----------|
| More Banks | Linear validator set growth |
| More Volume | Horizontal relayer scaling |
| More Chains | Additional Besu networks |

---

## 9. Comparison with Alternatives

### 9.1 vs RTGS (Real-Time Gross Settlement)

| Aspect | RTGS | Astra Clear |
|--------|------|-------------|
| Settlement | Per-transaction | Net basis |
| Liquidity | High requirement | Reduced by netting |
| Latency | Varies (mins-hours) | ~15 seconds |
| Cost | Per-txn fees | Reduced by netting |

### 9.2 vs Correspondent Banking

| Aspect | Correspondent | Astra Clear |
|--------|---------------|-------------|
| Intermediaries | Multiple | None |
| Transparency | Limited | Full on-chain |
| Settlement | T+1 to T+3 | Near real-time |
| Trust | Bilateral | Consortium |

### 9.3 vs Public Blockchain

| Aspect | Public Chain | Astra Clear |
|--------|--------------|-------------|
| Access | Permissionless | Permissioned |
| Finality | Probabilistic | Deterministic |
| Compliance | Difficult | Built-in |
| Throughput | Limited | Higher |

---

## 10. Future Work

### 10.1 Planned Enhancements

| Feature | Description | Priority |
|---------|-------------|----------|
| Multilateral Netting | N-party cycle detection | High |
| Credit Limits | Per-bank exposure caps | High |
| FX Support | Multi-currency netting | Medium |
| Privacy | Zero-knowledge proofs | Medium |
| Interoperability | IBC protocol | Low |

### 10.2 Research Directions

1. **Optimistic Execution**
   - Execute first, verify later
   - Fraud proof window

2. **Threshold Signatures**
   - Replace multi-sig with TSS
   - Constant-size signatures

3. **Recursive ZK Proofs**
   - Prove netting correctness
   - Privacy-preserving settlement

---

## 11. Conclusion

Astra Clear는 허가형 금융 네트워크에서 은행 간 결제 효율을 개선하는 실험적 시스템이다. IOU 토큰화와 Bilateral Netting을 통해 정산 건수를 줄이고, BFT 합의로 크로스체인 보안을 확보한다. 본 POC는 기술적 타당성을 검증하며, 프로덕션 적용 시 추가 검토가 필요하다.

---

## References

1. Tendermint: Byzantine Fault Tolerant State Machine Replication
2. ECDSA: Elliptic Curve Digital Signature Algorithm (NIST FIPS 186-4)
3. ERC-20: Token Standard (Ethereum Improvement Proposals)
4. Hyperledger Besu: Enterprise Ethereum Client
5. Cosmos SDK: Application Blockchain Interface

---

## Appendix A: Glossary

| Term | Definition |
|------|------------|
| **IOU** | I Owe You; 채무를 나타내는 토큰 |
| **Netting** | 상호 채무 상계 |
| **RTGS** | Real-Time Gross Settlement |
| **BFT** | Byzantine Fault Tolerant |
| **Validator** | 합의에 참여하는 노드 |
| **Threshold** | 합의에 필요한 최소 서명 수 |

---

## Appendix B: Configuration Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `block_time` | 5s | Cosmos block interval |
| `voting_timeout` | 100 blocks | Vote expiration |
| `netting_interval` | 720 blocks | Netting cycle (~1 hour) |
| `threshold_ratio` | 2/3 | Signature threshold |
| `retry_max` | 5 | Max retry attempts |
| `circuit_breaker_threshold` | 5 | Failures before open |
| `circuit_breaker_timeout` | 60s | Time in open state |
