# 📜 Project Spec: Cosmos-Besu Hybrid Debt Netting Engine

## 1. 시스템 개요 (System Overview)

- **목표:** 프라이빗 블록체인(Hyperledger Besu)을 사용하는 은행 간의 자금 이체를 즉시 처리하지 않고, Cosmos Hub에서 **부채(Credit/IOU)** 형태로 기록 및 관리하다가, 주기적으로 **상계(Netting)** 처리하여 실제 자금 이동을 최소화함.
- **핵심 철학:** "자본 효율성 극대화" (Liquidity Saving). 100건의 거래를 1건의 정산으로 압축.
- **아키텍처 유형:** Hub-and-Spoke (Hub: Cosmos SDK / Spoke: Hyperledger Besu).

---

## 2. 토큰 모델: 발행자 기반 IOU (Issuer-based Credit Token)

### 2.1. 개념

- 중앙 담보금 풀(Pool) 방식을 사용하지 않음.
- 각 참여 은행(Bank)은 자신의 신용을 담보로 하는 **고유한 Credit Token**을 Cosmos Hub 상에서 발행함.

### 2.2. 토큰 구조

- **`cred-{BankID}`**: 해당 은행이 지급을 보증하는 부채 토큰.
- **가치 페깅:** 1 `cred-Token` = 1 Stablecoin Unit (e.g., 1 USDC).

### 2.3. 발행 및 소유 로직

- **상황:** Bank A(Src) → Bank B(Dst) 100원 송금 발생.
- **해석:** A가 B에게 100원의 빚이 생김.
- **Cosmos 동작:**
    1. `cred-A` 100개를 발행(Mint).
    2. `Bank B`의 Cosmos 지갑으로 전송.
    - *결과:* B는 A에 대한 100원짜리 청구권을 보유함.

---

## 3. 핵심 프로세스 (Process Flow)

### Phase 1: 이벤트 관측 및 투표 (Observation & Voting)

1. **Src Chain (Besu):**
    - 사용자가 `Gateway` 컨트랙트에 자산 예치(Lock).
    - Event Log 발생: `TransferInitiated(sender, recipient, amount, nonce)`.
2. **Relayer:**
    - 이벤트를 감지하여 Cosmos에 `MsgVote` 트랜잭션 전송.
3. **Hub (Cosmos - `x/oracle`):**
    - Validator들이 투표. 2/3 이상 합의 시 해당 거래를 **확정(Finalized)**.
    - 확정 즉시 `cred-{Sender}`를 발행하여 `Recipient` 계정으로 전송.

### Phase 2: 주기적 상계 (Netting Cycle)

1. **Trigger:** 매 10블록마다 `EndBlocker`에서 실행.
2. **Logic (Swap & Burn):**
    - 시스템이 모든 은행 지갑을 스캔.
    - **Case:** Bank A가 `cred-B`(30개) 보유 / Bank B가 `cred-A`(100개) 보유.
    - **Action:** 교차하는 부채 중 **최소값(30)**만큼 상쇄.
        - A가 가진 `cred-B` 30개 소각 (Burn).
        - B가 가진 `cred-A` 100개 중 30개 소각 (Burn) → 70개 남음.
3. **Result:** 최종적으로 B는 A에 대한 청구권(`cred-A`) 70개만 보유.

## 4. 컴포넌트 상세 명세 (Component Specs)

### 4.1. Cosmos SDK (Golang) - The Engine

- **Module `x/oracle`:**
    - 기능: 외부 체인 이벤트 투표 및 합의.
    - Store: `VoteMap[TxHash] -> VoteStatus`
- **Module `x/netting` (핵심):**
    - 기능: `cred` 토큰 발행/소각/전송 및 주기적 상계 로직.
    - Hook: `EndBlocker`에서 10블록마다 트리거.
- **Module `x/multisig`:**
    - 기능: Besu용 ECDSA 키 관리 및 서명 생성.
    - Crypto: `libsecp256k1` 사용 (Ethereum 호환).

### 4.2. Hyperledger Besu (Solidity) - The Ledger

- **`Gateway.sol`:**
    - `deposit(token, amount, destination)`: ERC20 `transferFrom` 후 이벤트 방출.
- **`Executor.sol`:**
    - `executeBatch(commandId, targets, amounts, signatures[])`:
    - **검증 로직:**Solidity
        
        `for (uint i = 0; i < signatures.length; i++) {
            address signer = ecrecover(hash, v, r, s);
            require(isValidator[signer], "Invalid Signer");
        }
        require(validCount >= threshold, "Not enough sigs");`
        
    - `updateValidatorSet(newValidators[])`: 검증자 목록 관리.

### 4.3. Relayer (TypeScript/Go)

- 단순 배달부 역할.
- Logic: `Besu Event` → `Cosmos MsgVote` / `Cosmos Batch Event` → `Besu Execute`.

---

## 5. 보안 및 데이터 구조 (Security & Data)

### 5.1. 서명 방식

- **알고리즘:** ECDSA (secp256k1) - 이더리움 표준.
- **목적:** Besu 스마트 컨트랙트에서 가스비 효율적으로 검증하기 위함 (`ecrecover` precompile 사용).

### 5.2. KVStore 구조 (Cosmos)

- **Balances:** Standard `x/bank` store (`Address` -> `Denom` -> `Amount`).
- **Checkpoint:** `LastNettingBlockHeight` (마지막 상계 시점 기록).

---

## 

---

# 📜 Project Spec v2: Alice-to-Bob Netting & Settlement Engine

## 1. 페르소나 및 엔티티 정의 (Actors)

- **Alice (Src User):** A은행(Src Chain)의 고객. 돈을 보내는 사람.
- **Bob (Dst User):** B은행(Dst Chain)의 고객. 돈을 받는 사람.
- **Bank A (Src Node):** Hyperledger Besu 노드. 여기서 Alice의 토큰이 **소각(Burn)**됨.
- **Bank B (Dst Node):** Hyperledger Besu 노드. 여기서 Bob에게 토큰이 **발행(Mint)**됨.
- **Hub (Cosmos):** 부채(Credit)를 기록하고 상계하는 중앙 엔진.
- **Executor (Off-chain Relayer):** 최종 확정된 정산 명령을 배달하는 배달부.

---

## 2. 시나리오 흐름도 (Step-by-Step Flow)

### Step 1: Source Chain (Burn)

> 상황: Alice가 Bob에게 100 USDC를 보내고 싶어 함.
> 
1. **Action:** Alice가 A은행 체인의 `Gateway` 컨트랙트에 트랜잭션 전송.
2. **Contract Logic:**
    - Alice의 지갑에서 100 USDC를 **즉시 소각(Burn)**. (총 발행량 감소)
    - Event Log 방출: `TransferInitiated(sender: Alice, receiver: Bob, amount: 100, nonce: 101)`
3. **의미:** A은행은 이제 Alice의 자산을 없앴으므로, B은행에게 "Bob한테 100원 대신 줘"라고 부탁하는 상태(부채 발생)가 됨.

### Step 2: Cosmos Hub (Observation & Credit Minting)

1. **Observe:** Relayer들이 Step 1의 이벤트를 감지하고 `MsgVote` 제출.
2. **Consensus:** Validator 2/3 이상이 투표하면 확정.
3. **Credit Issuance (On-Chain Logic):**
    - 시스템은 "A은행이 B은행에게 줄 돈 100원이 생겼음"을 인지.
    - `cred-BankA` (A은행의 부채 토큰) 100개를 발행(Mint).
    - *`BankB_Cosmos_Account`*에게 이 토큰을 전송.
    - *현 상태: B은행은 A은행에 대한 100원짜리 청구권을 보유함.*

### Step 3: Netting (The Magic Moment)

> 가정: 이전에 반대 방향 거래(Bob → Alice 30원)가 있어서, A은행도 cred-BankB 30개를 가지고 있다고 가정.
> 
1. **Trigger:** 10블록 주기 도래 (`EndBlocker`).
2. **Netting Logic:**
    - A은행 보유: `cred-BankB` (30개)
    - B은행 보유: `cred-BankA` (100개)
    - **상계 실행:** 교차되는 30개만큼 서로 소각(Burn).
        - A은행의 `cred-BankB` 30개 → 0개 (전부 소각)
        - B은행의 `cred-BankA` 100개 → 70개 (30개 차감)
3. **Result:** 최종적으로 B은행은 A은행에게 **70원**만 받으면 됨.

### Step 4: Settlement & Signing (Multisig)

1. **Batch Creation:**
    - 상계 후 남은 잔액(70원)과 수신자 정보(Bob)를 묶어서 출금 명령 생성.
    - `Command: { TargetChain: BankB, MintTo: Bob, Amount: 70 }`
    - *(참고: 원래 100원이었지만, 상계 로직은 은행 간 정산이고, Bob은 100원을 다 받아야 하므로, 여기서는 Bob에게 줄 100원에 대한 Mint 명령이 나가야 함. **중요: 아키텍처 결정 필요**)*
    
    > ⚠️ 잠깐! 여기서 아주 중요한 비즈니스 로직 분기점입니다.
    > 
    
    > 옵션 A (Real-time Mint): Bob은 상계고 나발이고 즉시 받아야 함. (사용자 경험 최우선)
    > 
    > 
    > 옵션 B (Deferred Mint): 상계가 끝날 때까지 Bob도 기다려야 함. (안정성 최우선)
    > 
    
    > 님의 요청 사항("Bob의 계정에 바로 토큰 mint")을 만족시키려면 옵션 A로 가되, 은행 간 정산(Settlement)과 사용자 지급(User Payment)을 분리해야 합니다.
    > 
    
    > [수정된 로직 적용]:
    > 
    
    > Cosmos는 Alice의 요청(100원)이 확정되자마자 즉시 B은행 체인에 "Bob에게 100원 Mint 해줘"라는 명령을 내보냅니다. (상계 여부와 상관없이)상계(Netting)는 은행끼리 나중에 정산할 때(실제 돈 옮길 때) 쓰는 장부상의 로직으로 남겨둡니다.
    > 
    
    **(다시 정리한 Step 4 - Real-time User Experience 버전)**
    
    1. Step 2(확정) 직후, Cosmos는 즉시 `MsgSignMintRequest`를 생성.
    2. 내용: "Bank B야, **Bob에게 100원 Mint 해줘.** (이 돈은 나중에 A랑 상계해서 처리할게)"
    3. Validator들이 이 명령에 **ECDSA 서명**.

### Step 5: Destination Chain (Execution & Mint)

1. **Relay:** Executor가 서명(`[Sig1, Sig2, Sig3]`)과 명령(`Mint 100 to Bob`)을 B은행 체인으로 배달.
2. **Verify:** `Executor` 컨트랙트가 서명 검증.
3. **Action:**
    - B은행 체인의 토큰 컨트랙트 호출.
    - **Bob의 주소로 100 USDC 발행(Mint).**
    - Event: `TransferCompleted(Bob, 100)`

---

## 3. 핵심 데이터 구조 (Data Structures)

### 3.1. Cosmos Store (State)

Go

# 

`// 1. 은행 간 부채 장부 (Netting용)
// Key: BankAddress, Value: { Denom: Amount }
Store/BankB_Account: {
    "cred-BankA": 100  // B는 A에게 100원 받을 권리가 있음 (상계 전)
}

// 2. 사용자 지급 대기열 (Outgoing Batch)
// Key: BatchID, Value: PaymentInstruction
Store/OutgoingBatches: {
    ID: 501,
    Target: "BankB_Chain",
    Recipient: "Bob_Address",
    Amount: 100,
    Status: "Signed_ReadyToRelay"
}`

### 3.2. Besu Smart Contract (Solidity)

Solidity

# 

`// Gateway.sol (Src)
function sendToChain(string memory _destChain, string memory _recipient, uint256 _amount) public {
    // 1. Alice의 토큰을 태워버림 (Burn)
    token.burn(msg.sender, _amount); 
    
    // 2. 이벤트 방출 -> Cosmos가 감지함
    emit TransferInitiated(msg.sender, _recipient, _amount); 
}

// Executor.sol (Dst)
function executeMint(
    bytes32 _commandId, 
    address _recipient, // Bob
    uint256 _amount,    // 100
    bytes[] memory _sigs
) public {
    // 1. 서명 검증 (Validator Set과 대조)
    verifySignatures(_commandId, _sigs);
    
    // 2. Bob에게 토큰 발행 (Mint)
    token.mint(_recipient, _amount);
}`

---

