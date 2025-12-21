# Features

Astra Clear 기능별 상세 설명

---

## 1. Cross-Chain Transfer

### 1.1 기능 개요

Bank A 네트워크의 사용자가 Bank B 네트워크의 수신자에게 토큰을 전송하는 기능.

### 1.2 처리 흐름

```
Source Chain (Bank A)         Cosmos Hub              Dest Chain (Bank B)
─────────────────────────────────────────────────────────────────────────
     │                            │                          │
     │ 1. initiateTransfer()      │                          │
     ├────────────────────────────┤                          │
     │    - Burn tokens           │                          │
     │    - Emit event            │                          │
     │                            │                          │
     │ 2. Relayer submits vote    │                          │
     ├───────────────────────────▶│                          │
     │                            │                          │
     │                            │ 3. Validators vote       │
     │                            ├──────────────────────────┤
     │                            │    - 2/3 consensus       │
     │                            │                          │
     │                            │ 4. Generate mint cmd     │
     │                            │    + Multi-sig           │
     │                            │                          │
     │                            │ 5. Relayer fetches cmd   │
     │                            ├─────────────────────────▶│
     │                            │                          │
     │                            │                          │ 6. executeMint()
     │                            │                          ├─────────────────
     │                            │                          │    - Verify sigs
     │                            │                          │    - Mint tokens
```

### 1.3 관련 코드

**Gateway.sol - 송금 시작**
```solidity
function initiateTransfer(
    address recipient,
    uint256 amount,
    string calldata destChain
) external {
    token.burn(msg.sender, amount);
    emit TransferInitiated(
        keccak256(abi.encodePacked(block.timestamp, msg.sender, recipient, amount)),
        msg.sender,
        recipient,
        amount,
        destChain
    );
}
```

**Executor.sol - 수신 실행**
```solidity
function executeMint(
    bytes32 commandId,
    address recipient,
    uint256 amount,
    bytes[] calldata signatures
) external nonReentrant {
    require(signatures.length >= threshold);
    // Verify signatures
    // Mint tokens
}
```

### 1.4 보안 고려사항

| 위험 | 대응 |
|------|------|
| Double-spending | processedCommands 맵으로 중복 처리 방지 |
| Replay attack | commandId에 timestamp 포함 |
| Signature forgery | 2/3 threshold + ecrecover 검증 |

---

## 2. Oracle Consensus

### 2.1 기능 개요

External chain 이벤트를 Validator 투표를 통해 검증하고 확정하는 기능.

### 2.2 투표 프로세스

```
┌─────────────────────────────────────────────────────────────┐
│                     VOTE AGGREGATION                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   Transfer Event: 0xabc...                                  │
│   ───────────────────────────────                           │
│                                                              │
│   Validator A: ✓ Voted (block 100)                         │
│   Validator B: ✓ Voted (block 101)                         │
│   Validator C: ✓ Voted (block 101)                         │
│   Validator D: ○ Pending                                    │
│   Validator E: ○ Pending                                    │
│                                                              │
│   Status: 3/5 votes (60%)                                   │
│   Threshold: 4/5 (67%)                                      │
│   Result: PENDING                                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 2.3 관련 코드

**x/oracle/keeper - 투표 제출**
```go
func (k Keeper) SubmitVote(ctx sdk.Context, validator sdk.AccAddress,
    txHash string, recipient string, amount math.Int, sourceChain string) error {

    // Validate voter is active validator
    if !k.IsActiveValidator(ctx, validator) {
        return types.ErrValidatorNotActive
    }

    // Check duplicate vote
    if k.HasVoted(ctx, txHash, validator) {
        return types.ErrDuplicateVote
    }

    // Record vote
    k.RecordVote(ctx, txHash, validator, recipient, amount, sourceChain)

    // Check consensus
    if k.HasReachedConsensus(ctx, txHash) {
        k.ConfirmTransfer(ctx, txHash)
    }

    return nil
}
```

### 2.4 합의 로직

- **Threshold**: 2/3 + 1 of active validators
- **Timeout**: Configurable (default 100 blocks)
- **Dynamic Threshold**: 오프라인 validator 제외 시 재계산

---

## 3. Bilateral Netting

### 3.1 기능 개요

두 은행 간 상호 채무를 상계하여 순 정산 금액만 처리하는 기능.

### 3.2 상계 예시

```
┌─────────────────────────────────────────────────────────────┐
│                    BEFORE NETTING                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   Bank A owes Bank B                                        │
│   ─────────────────                                         │
│   cred-A held by Bank B: 100,000                           │
│                                                              │
│   Bank B owes Bank A                                        │
│   ─────────────────                                         │
│   cred-B held by Bank A: 30,000                            │
│                                                              │
│   Gross Obligations: 130,000                                │
│                                                              │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     AFTER NETTING                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   Netted Amount: min(100,000, 30,000) = 30,000             │
│                                                              │
│   Burn:                                                      │
│   - cred-A: 30,000 (from Bank B)                           │
│   - cred-B: 30,000 (from Bank A)                           │
│                                                              │
│   Remaining Obligations:                                     │
│   - Bank A → Bank B: 70,000                                 │
│                                                              │
│   Reduction: 130,000 → 70,000 (46% reduction)              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 3.3 관련 코드

**x/netting/keeper - Netting 실행**
```go
func (k Keeper) ExecuteBilateralNetting(ctx sdk.Context,
    bankA, bankB string) (netAmount math.Int, netDebtor string, err error) {

    // Get mutual balances
    aOwesB := k.GetCreditBalance(ctx, bankB, "cred-"+bankA)
    bOwesA := k.GetCreditBalance(ctx, bankA, "cred-"+bankB)

    // Calculate net
    netAmount = math.MinInt(aOwesB, bOwesA)

    if netAmount.IsZero() {
        return math.ZeroInt(), "", types.ErrNettingNotRequired
    }

    // Burn netted amounts
    k.BurnCredit(ctx, bankB, "cred-"+bankA, netAmount)
    k.BurnCredit(ctx, bankA, "cred-"+bankB, netAmount)

    // Determine net debtor
    if aOwesB.GT(bOwesA) {
        return aOwesB.Sub(bOwesA), bankA, nil
    }
    return bOwesA.Sub(aOwesB), bankB, nil
}
```

### 3.4 Netting Cycle

| Parameter | Value | Description |
|-----------|-------|-------------|
| Trigger | Block height | 매 N 블록마다 실행 |
| Default interval | 720 blocks | 약 1시간 (5초 블록) |
| Pairs | All active pairs | 잔액 있는 모든 쌍 |

---

## 4. Credit Token (IOU) Management

### 4.1 기능 개요

은행 간 채무를 토큰화하여 관리하는 기능.

### 4.2 토큰 구조

```
┌─────────────────────────────────────────────────────────────┐
│                    CREDIT TOKEN MODEL                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   Token: cred-BANK_A                                        │
│   ──────────────────                                        │
│   Issuer: Bank A                                            │
│   Meaning: "Bank A owes the holder"                         │
│   Value: 1 cred = 1 Stablecoin Unit                        │
│                                                              │
│   Holders:                                                   │
│   ├── Bank B: 100,000 (Bank A owes Bank B)                 │
│   ├── Bank C: 50,000  (Bank A owes Bank C)                 │
│   └── Bank D: 25,000  (Bank A owes Bank D)                 │
│                                                              │
│   Total Supply: 175,000                                     │
│   (= Total debt of Bank A to other banks)                   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 4.3 관련 코드

**x/netting/keeper - Credit 발행**
```go
func (k Keeper) IssueCredit(ctx sdk.Context,
    issuerBank, holderBank string, amount math.Int) error {

    denom := "cred-" + issuerBank

    // Update holder balance
    currentBalance := k.GetCreditBalance(ctx, holderBank, denom)
    newBalance := currentBalance.Add(amount)
    k.SetCreditBalance(ctx, holderBank, denom, newBalance)

    // Emit event
    ctx.EventManager().EmitEvent(
        sdk.NewEvent(
            types.EventTypeCreditIssued,
            sdk.NewAttribute(types.AttributeKeyDenom, denom),
            sdk.NewAttribute(types.AttributeKeyAmount, amount.String()),
            sdk.NewAttribute(types.AttributeKeyIssuerBank, issuerBank),
            sdk.NewAttribute(types.AttributeKeyHolderBank, holderBank),
        ),
    )

    return nil
}
```

---

## 5. Multi-Signature Management

### 5.1 기능 개요

Validator 서명을 집계하여 threshold signature를 생성하는 기능.

### 5.2 서명 흐름

```
┌─────────────────────────────────────────────────────────────┐
│                   SIGNATURE AGGREGATION                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   Message: MintCommand(0xabc, 0xdef, 1000, "besu-b")       │
│   ──────────────────────────────────────────────────────    │
│                                                              │
│   Step 1: Compute Message Hash                              │
│   ────────────────────────────                              │
│   hash = keccak256(commandId || recipient || amount || chain)│
│   ethHash = "\x19Ethereum Signed Message:\n32" + hash       │
│                                                              │
│   Step 2: Collect Signatures                                │
│   ──────────────────────────                                │
│   Validator A: sig_a = sign(ethHash, privKey_a)            │
│   Validator B: sig_b = sign(ethHash, privKey_b)            │
│   Validator C: sig_c = sign(ethHash, privKey_c)            │
│                                                              │
│   Step 3: Aggregate                                         │
│   ─────────────────                                         │
│   signatures = [sig_a, sig_b, sig_c]                        │
│                                                              │
│   Step 4: Verify on Executor                                │
│   ──────────────────────────                                │
│   for each sig in signatures:                               │
│       signer = ecrecover(ethHash, sig)                      │
│       require(validators[signer] == true)                   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 5.3 관련 코드

**x/multisig/keeper - 서명 집계**
```go
func (k Keeper) AggregateSignatures(ctx sdk.Context,
    commandId string) ([][]byte, error) {

    signatures := [][]byte{}

    for _, validator := range k.GetValidators(ctx) {
        sig, found := k.GetSignature(ctx, commandId, validator)
        if found {
            signatures = append(signatures, sig)
        }
    }

    threshold := k.GetThreshold(ctx)
    if len(signatures) < int(threshold) {
        return nil, types.ErrInsufficientSignatures
    }

    return signatures, nil
}
```

---

## 6. Error Handling & Recovery

### 6.1 기능 개요

시스템 오류 발생 시 자동 복구 및 롤백 메커니즘.

### 6.2 오류 유형별 처리

| Error Type | Handler | Recovery |
|------------|---------|----------|
| Network timeout | Exponential backoff | 최대 5회 재시도 |
| RPC failure | Circuit breaker | 60초 후 half-open |
| Consensus timeout | Dynamic threshold | 오프라인 validator 제외 |
| Netting failure | Snapshot rollback | 이전 상태 복원 |
| Signature mismatch | Validator sync | 세트 버전 동기화 |

### 6.3 Netting Rollback

```
┌─────────────────────────────────────────────────────────────┐
│                    NETTING ROLLBACK                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   1. Create Snapshot                                        │
│   ──────────────────                                        │
│   snapshot = {                                              │
│       cycleId: 42,                                          │
│       balances: {                                           │
│           "bankA": {"cred-bankB": 100, "cred-bankC": 50},  │
│           "bankB": {"cred-bankA": 30, "cred-bankC": 20},   │
│       }                                                      │
│   }                                                          │
│                                                              │
│   2. Execute Netting                                        │
│   ──────────────────                                        │
│   ... processing ...                                        │
│                                                              │
│   3a. Success → Discard snapshot                            │
│   3b. Error → Rollback                                      │
│   ────────────────────                                      │
│   for bank, denoms := range snapshot.balances {            │
│       for denom, amount := range denoms {                  │
│           SetCreditBalance(bank, denom, amount)            │
│       }                                                      │
│   }                                                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 7. Audit & Logging

### 7.1 기능 개요

모든 트랜잭션 및 상태 변경에 대한 감사 로그 기록.

### 7.2 이벤트 유형

**Oracle Events**
| Event | Attributes |
|-------|------------|
| `vote_submitted` | validator, tx_hash, amount |
| `transfer_confirmed` | tx_hash, recipient, amount |
| `transfer_rejected` | tx_hash, reason |

**Netting Events**
| Event | Attributes |
|-------|------------|
| `credit_issued` | denom, amount, issuer, holder |
| `credit_burned` | denom, amount, holder |
| `netting_completed` | cycle_id, pair_count, net_amount |
| `netting_rollback` | cycle_id, reason |

### 7.3 Query API

```bash
# Vote status
interbank-nettingd query oracle vote-status <tx-hash>

# Credit balance
interbank-nettingd query netting credit-balance <bank-id> <denom>

# Netting history
interbank-nettingd query netting cycle-history <cycle-id>
```

---

## 8. Gas Estimation

### 8.1 기능 개요

Smart contract 실행 전 가스 비용을 추정하고 버퍼를 추가하는 기능.

### 8.2 추정 로직

```solidity
function estimateMintGas(
    bytes32 commandId,
    address recipient,
    uint256 amount,
    bytes[] calldata signatures
) external view returns (uint256) {
    uint256 baseGas = 50000;           // State changes
    uint256 sigGas = signatures.length * 5000;  // Per signature
    uint256 mintGas = 30000;           // Token mint

    uint256 total = baseGas + sigGas + mintGas;
    return (total * 120) / 100;        // 20% buffer
}
```

### 8.3 비용 구성

| Operation | Gas Cost | Description |
|-----------|----------|-------------|
| Base | 50,000 | 상태 변경 기본 비용 |
| Signature verify | 5,000 | ecrecover per signature |
| Token mint | 30,000 | ERC20 mint operation |
| Buffer | +20% | 안전 마진 |

---

## Next: [WHITEPAPER.md](WHITEPAPER.md) - 설계 원칙 및 기술 명세
