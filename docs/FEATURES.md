# Features

<details>
<summary><b>🇺🇸 English</b></summary>

## 1. Cross-Chain Transfer

### Overview
Transfer tokens from Bank A network to Bank B network recipient.

### Flow
```
Source Chain                Cosmos Hub              Dest Chain
─────────────────────────────────────────────────────────────
     │                           │                       │
     │ 1. initiateTransfer()     │                       │
     ├───────────────────────────┤                       │
     │    Burn + Emit event      │                       │
     │                           │                       │
     │ 2. Validators vote        │                       │
     │    ───────────────────────┤                       │
     │                           │ 3. 2/3 consensus      │
     │                           │                       │
     │                           │ 4. Mint command       │
     │                           ├──────────────────────▶│
     │                           │                       │
     │                           │    5. Verify + Mint   │
```

### Code Reference

**Gateway.sol**
```solidity
function initiateTransfer(
    address recipient,
    uint256 amount,
    string calldata destChain
) external {
    token.burn(msg.sender, amount);
    emit TransferInitiated(transferId, msg.sender, recipient, amount, destChain);
}
```

**Executor.sol**
```solidity
function executeMint(
    bytes32 commandId,
    address recipient,
    uint256 amount,
    bytes[] calldata signatures
) external nonReentrant {
    require(signatures.length >= threshold);
    // Verify signatures, mint tokens
}
```

### Security

| Risk | Mitigation |
|------|------------|
| Double-spending | processedCommands map |
| Replay attack | commandId includes timestamp |
| Signature forgery | 2/3 threshold + ecrecover |

---

## 2. Oracle Consensus

### Overview
Verify external chain events through validator voting.

### Voting Process
```
Transfer Event: 0xabc...

Validator A: ✓ Voted (block 100)
Validator B: ✓ Voted (block 101)
Validator C: ✓ Voted (block 101)
Validator D: ○ Pending
Validator E: ○ Pending

Status: 3/5 (60%)
Threshold: 4/5 (67%)
Result: PENDING
```

### Threshold Calculation
```
threshold = (validatorCount * 2 + 2) / 3

3 validators: 3 required (100%)
5 validators: 4 required (80%)
7 validators: 5 required (71%)
```

### Dynamic Threshold
Excludes offline validators:
```go
func GetDynamicThreshold(ctx) (threshold, activeCount) {
    activeCount := countActiveValidators(ctx)
    threshold = (activeCount * 2 + 2) / 3
    return threshold, activeCount
}
```

---

## 3. Bilateral Netting

### Overview
Offset mutual obligations between two banks.

### Example
```
BEFORE NETTING
──────────────
Bank A → Bank B: 100 (cred-A held by B)
Bank B → Bank A:  30 (cred-B held by A)
Gross: 130

NETTING
───────
Net = min(100, 30) = 30
Burn cred-A: 30, Burn cred-B: 30

AFTER NETTING
─────────────
Net: Bank A → Bank B: 70
Reduction: 130 → 70 (46%)
```

### Code Reference
```go
func ExecuteBilateralNetting(ctx, bankA, bankB string) (netAmount, netDebtor, error) {
    aOwesB := GetCreditBalance(ctx, bankB, "cred-"+bankA)
    bOwesA := GetCreditBalance(ctx, bankA, "cred-"+bankB)

    netAmount = math.MinInt(aOwesB, bOwesA)

    BurnCredit(ctx, bankB, "cred-"+bankA, netAmount)
    BurnCredit(ctx, bankA, "cred-"+bankB, netAmount)

    if aOwesB.GT(bOwesA) {
        return aOwesB.Sub(bOwesA), bankA, nil
    }
    return bOwesA.Sub(aOwesB), bankB, nil
}
```

### Netting Cycle

| Parameter | Default | Description |
|-----------|---------|-------------|
| Trigger | Block height | Every N blocks |
| Interval | 720 blocks | ~1 hour (5s blocks) |
| Pairs | All active | Pairs with balances |

---

## 4. Credit Token (IOU)

### Token Model
```
Token: cred-BANK_A
Issuer: Bank A
Meaning: "Bank A owes the holder"
Value: 1 cred = 1 Stablecoin Unit

Holders:
├── Bank B: 100,000
├── Bank C:  50,000
└── Bank D:  25,000

Total Supply: 175,000 (Bank A's total debt)
```

### Properties

| Property | Description |
|----------|-------------|
| Issuer-specific | Separate token per bank |
| Fungible | Same issuer tokens interchangeable |
| Burnable | Destroyed during netting |
| Non-transferable | Only netted, not traded |

---

## 5. Multi-Signature

### Signature Aggregation
```
Message: MintCommand(0xabc, 0xdef, 1000, "besu-b")

Step 1: Compute Hash
  hash = keccak256(commandId || recipient || amount || chain)
  ethHash = "\x19Ethereum Signed Message:\n32" + hash

Step 2: Collect Signatures
  Validator A: sig_a = sign(ethHash, privKey_a)
  Validator B: sig_b = sign(ethHash, privKey_b)
  Validator C: sig_c = sign(ethHash, privKey_c)

Step 3: Aggregate
  signatures = [sig_a, sig_b, sig_c]

Step 4: Verify on Executor
  for sig in signatures:
      signer = ecrecover(ethHash, sig)
      require(validators[signer] == true)
```

---

## 6. Error Handling

### Recovery Mechanisms

| Error | Handler | Recovery |
|-------|---------|----------|
| Network timeout | Exponential backoff | Max 5 retries |
| RPC failure | Circuit breaker | 60s open state |
| Consensus timeout | Dynamic threshold | Exclude offline |
| Netting failure | Snapshot | Rollback |
| Signature mismatch | Validator sync | Version check |

### Netting Rollback
```go
func ExecuteNettingWithRollback(ctx, pairs) error {
    snapshot := CreateSnapshot(ctx, pairs)

    err := ExecuteNetting(ctx, pairs)
    if err != nil {
        RollbackNetting(ctx, snapshot)
        return err
    }

    return nil
}
```

---

## 7. Gas Estimation

### Estimation Logic
```solidity
function estimateMintGas(signatures) returns (uint256) {
    uint256 baseGas = 50000;           // State changes
    uint256 sigGas = signatures.length * 5000;
    uint256 mintGas = 30000;

    uint256 total = baseGas + sigGas + mintGas;
    return (total * 120) / 100;        // 20% buffer
}
```

### Cost Breakdown

| Operation | Gas | Description |
|-----------|-----|-------------|
| Base | 50,000 | State change overhead |
| Sig verify | 5,000 | Per signature ecrecover |
| Token mint | 30,000 | ERC20 mint |
| Buffer | +20% | Safety margin |

</details>

<details open>
<summary><b>🇰🇷 한국어</b></summary>

## 1. 크로스체인 전송

### 개요
Bank A 네트워크에서 Bank B 네트워크 수신자에게 토큰 전송.

### 흐름
```
Source Chain                Cosmos Hub              Dest Chain
─────────────────────────────────────────────────────────────
     │                           │                       │
     │ 1. initiateTransfer()     │                       │
     ├───────────────────────────┤                       │
     │    Burn + 이벤트 발생     │                       │
     │                           │                       │
     │ 2. Validator 투표         │                       │
     │    ───────────────────────┤                       │
     │                           │ 3. 2/3 합의           │
     │                           │                       │
     │                           │ 4. Mint 명령          │
     │                           ├──────────────────────▶│
     │                           │                       │
     │                           │    5. 검증 + Mint     │
```

### 코드 참조

**Gateway.sol**
```solidity
function initiateTransfer(
    address recipient,
    uint256 amount,
    string calldata destChain
) external {
    token.burn(msg.sender, amount);
    emit TransferInitiated(transferId, msg.sender, recipient, amount, destChain);
}
```

**Executor.sol**
```solidity
function executeMint(
    bytes32 commandId,
    address recipient,
    uint256 amount,
    bytes[] calldata signatures
) external nonReentrant {
    require(signatures.length >= threshold);
    // 서명 검증 후 토큰 발행
}
```

### 보안

| 위험 | 대응 |
|------|------|
| 이중 지불 | processedCommands 맵 |
| 재전송 공격 | commandId에 timestamp 포함 |
| 서명 위조 | 2/3 threshold + ecrecover |

---

## 2. Oracle 합의

### 개요
Validator 투표를 통한 외부 체인 이벤트 검증.

### 투표 프로세스
```
전송 이벤트: 0xabc...

Validator A: ✓ 투표 완료 (블록 100)
Validator B: ✓ 투표 완료 (블록 101)
Validator C: ✓ 투표 완료 (블록 101)
Validator D: ○ 대기 중
Validator E: ○ 대기 중

현황: 3/5 (60%)
임계값: 4/5 (67%)
결과: 대기 중
```

### 임계값 계산
```
threshold = (validatorCount * 2 + 2) / 3

3명: 3명 필요 (100%)
5명: 4명 필요 (80%)
7명: 5명 필요 (71%)
```

### 동적 임계값
오프라인 Validator 제외:
```go
func GetDynamicThreshold(ctx) (threshold, activeCount) {
    activeCount := countActiveValidators(ctx)
    threshold = (activeCount * 2 + 2) / 3
    return threshold, activeCount
}
```

---

## 3. 양방향 Netting

### 개요
두 은행 간 상호 채무 상계.

### 예시
```
NETTING 전
──────────
Bank A → Bank B: 100 (cred-A를 B가 보유)
Bank B → Bank A:  30 (cred-B를 A가 보유)
총 채무: 130

NETTING
───────
상계액 = min(100, 30) = 30
cred-A 30 소각, cred-B 30 소각

NETTING 후
──────────
순 채무: Bank A → Bank B: 70
감소율: 130 → 70 (46%)
```

### 코드 참조
```go
func ExecuteBilateralNetting(ctx, bankA, bankB string) (netAmount, netDebtor, error) {
    aOwesB := GetCreditBalance(ctx, bankB, "cred-"+bankA)
    bOwesA := GetCreditBalance(ctx, bankA, "cred-"+bankB)

    netAmount = math.MinInt(aOwesB, bOwesA)

    BurnCredit(ctx, bankB, "cred-"+bankA, netAmount)
    BurnCredit(ctx, bankA, "cred-"+bankB, netAmount)

    if aOwesB.GT(bOwesA) {
        return aOwesB.Sub(bOwesA), bankA, nil
    }
    return bOwesA.Sub(aOwesB), bankB, nil
}
```

### Netting 사이클

| 파라미터 | 기본값 | 설명 |
|----------|--------|------|
| 트리거 | 블록 높이 | 매 N 블록마다 |
| 주기 | 720 블록 | 약 1시간 (5초 블록) |
| 대상 | 모든 활성 쌍 | 잔액 있는 은행 쌍 |

---

## 4. 신용 토큰 (IOU)

### 토큰 모델
```
토큰: cred-BANK_A
발행자: Bank A
의미: "Bank A가 보유자에게 갚아야 할 금액"
가치: 1 cred = 1 스테이블코인 단위

보유자:
├── Bank B: 100,000
├── Bank C:  50,000
└── Bank D:  25,000

총 발행량: 175,000 (Bank A의 총 채무)
```

### 속성

| 속성 | 설명 |
|------|------|
| 발행자별 분리 | 은행별 독립 토큰 |
| 대체 가능 | 동일 발행자 토큰 교환 가능 |
| 소각 가능 | Netting 시 소각 |
| 양도 불가 | 거래 불가, Netting만 가능 |

---

## 5. 다중 서명

### 서명 집계
```
메시지: MintCommand(0xabc, 0xdef, 1000, "besu-b")

1단계: 해시 계산
  hash = keccak256(commandId || recipient || amount || chain)
  ethHash = "\x19Ethereum Signed Message:\n32" + hash

2단계: 서명 수집
  Validator A: sig_a = sign(ethHash, privKey_a)
  Validator B: sig_b = sign(ethHash, privKey_b)
  Validator C: sig_c = sign(ethHash, privKey_c)

3단계: 집계
  signatures = [sig_a, sig_b, sig_c]

4단계: Executor 검증
  for sig in signatures:
      signer = ecrecover(ethHash, sig)
      require(validators[signer] == true)
```

---

## 6. 오류 처리

### 복구 메커니즘

| 오류 | 핸들러 | 복구 방법 |
|------|--------|----------|
| 네트워크 타임아웃 | Exponential backoff | 최대 5회 재시도 |
| RPC 실패 | Circuit breaker | 60초 open 상태 |
| 합의 타임아웃 | Dynamic threshold | 오프라인 제외 |
| Netting 실패 | 스냅샷 | 롤백 |
| 서명 불일치 | Validator 동기화 | 버전 체크 |

### Netting 롤백
```go
func ExecuteNettingWithRollback(ctx, pairs) error {
    snapshot := CreateSnapshot(ctx, pairs)

    err := ExecuteNetting(ctx, pairs)
    if err != nil {
        RollbackNetting(ctx, snapshot)
        return err
    }

    return nil
}
```

---

## 7. 가스 추정

### 추정 로직
```solidity
function estimateMintGas(signatures) returns (uint256) {
    uint256 baseGas = 50000;           // 상태 변경
    uint256 sigGas = signatures.length * 5000;
    uint256 mintGas = 30000;

    uint256 total = baseGas + sigGas + mintGas;
    return (total * 120) / 100;        // 20% 버퍼
}
```

### 비용 구성

| 작업 | 가스 | 설명 |
|------|------|------|
| 기본 | 50,000 | 상태 변경 기본 비용 |
| 서명 검증 | 5,000 | 서명당 ecrecover |
| 토큰 발행 | 30,000 | ERC20 mint |
| 버퍼 | +20% | 안전 마진 |

</details>
