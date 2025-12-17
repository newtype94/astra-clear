# 설계 문서

## 개요

은행간 상계 엔진은 실시간 사용자 지급과 지연된 은행간 정산을 분리하는 혁신적인 하이브리드 블록체인 시스템입니다. 시스템은 Cosmos Hub를 중앙 청산 엔진으로 사용하고, 각 은행이 운영하는 Hyperledger Besu 네트워크를 연결하여 발행자 기반 신용 토큰(IOU) 모델을 통해 자본 효율성을 극대화합니다.

핵심 철학은 "100건의 거래를 1건의 정산으로 압축"하여 유동성 절약(Liquidity Saving)을 달성하는 것입니다.

## 아키텍처

### 전체 시스템 아키텍처

```
┌─────────────────┐    Events     ┌──────────────────┐    Commands    ┌─────────────────┐
│   Bank A        │ ──────────────▶│   Cosmos Hub     │◀────────────── │   Bank B        │
│ (Hyperledger    │               │  (Central        │               │ (Hyperledger    │
│  Besu Network)  │               │   Clearing)      │               │  Besu Network)  │
│                 │               │                  │               │                 │
│ ┌─────────────┐ │               │ ┌──────────────┐ │               │ ┌─────────────┐ │
│ │ Gateway.sol │ │               │ │ x/oracle     │ │               │ │Executor.sol │ │
│ │ (Burn)      │ │               │ │ x/netting    │ │               │ │ (Mint)      │ │
│ │             │ │               │ │ x/multisig   │ │               │ │             │ │
│ └─────────────┘ │               │ └──────────────┘ │               │ └─────────────┘ │
└─────────────────┘               └──────────────────┘               └─────────────────┘
        │                                   │                                   │
        │                                   │                                   │
        └─────────────── Relayer ───────────┴─────────────── Relayer ──────────┘
```

### Hub-and-Spoke 모델

- **Hub (Cosmos)**: 중앙 청산 엔진, 모든 부채 관리 및 상계 처리
- **Spoke (Besu Networks)**: 각 은행의 프라이빗 블록체인, 실제 토큰 발행/소각
- **Relayers**: 상태 없는(stateless) 메시지 전달자

## 컴포넌트 및 인터페이스

### Cosmos Hub 모듈

#### x/oracle 모듈
```go
type Vote struct {
    TxHash      string
    Validator   string
    EventData   TransferEvent
    Signature   []byte
}

type VoteStatus struct {
    Votes       []Vote
    Confirmed   bool
    Threshold   int
}
```

**주요 기능:**
- 외부 체인 이벤트 수집 및 검증
- 2/3 과반수 합의 메커니즘
- 이벤트 최종 확정 처리

#### x/netting 모듈
```go
type CreditToken struct {
    Denom       string  // "cred-{BankID}"
    IssuerBank  string
    Amount      sdk.Int
}

type NettingCycle struct {
    BlockHeight int64
    Pairs       []BankPair
    NetAmounts  map[string]sdk.Int
}
```

**주요 기능:**
- 신용 토큰 발행/소각/전송
- 주기적 상계 로직 (EndBlocker)
- 은행간 부채 포지션 관리

#### x/multisig 모듈
```go
type ValidatorSet struct {
    Validators  []Validator
    Threshold   int
    UpdateHeight int64
}

type MintCommand struct {
    TargetChain string
    Recipient   string
    Amount      sdk.Int
    Signatures  []ECDSASignature
}
```

**주요 기능:**
- ECDSA 키 관리 (secp256k1)
- 다중 서명 생성 및 검증
- 검증자 세트 관리

### Hyperledger Besu 스마트 컨트랙트

#### Gateway.sol (소스 체인)
```solidity
contract Gateway {
    IERC20 public token;
    
    event TransferInitiated(
        address indexed sender,
        string recipient,
        uint256 amount,
        uint256 nonce
    );
    
    function sendToChain(
        string memory destChain,
        string memory recipient,
        uint256 amount
    ) external {
        token.burn(msg.sender, amount);
        emit TransferInitiated(msg.sender, recipient, amount, nonce++);
    }
}
```

#### Executor.sol (목적지 체인)
```solidity
contract Executor {
    mapping(address => bool) public validators;
    uint256 public threshold;
    
    function executeMint(
        bytes32 commandId,
        address recipient,
        uint256 amount,
        bytes[] memory signatures
    ) external {
        require(verifySignatures(commandId, signatures), "Invalid signatures");
        token.mint(recipient, amount);
        emit TransferCompleted(recipient, amount);
    }
    
    function verifySignatures(
        bytes32 hash,
        bytes[] memory signatures
    ) internal view returns (bool) {
        uint validCount = 0;
        for (uint i = 0; i < signatures.length; i++) {
            address signer = ecrecover(hash, v, r, s);
            if (validators[signer]) validCount++;
        }
        return validCount >= threshold;
    }
}
```

## 데이터 모델

### Cosmos Hub 상태 저장소

```
// 은행별 신용 토큰 잔액
Store/bank/{bankAddress}/balance/{denom} -> Amount

// 상계 이력
Store/netting/cycle/{blockHeight} -> NettingResult

// 검증자 세트
Store/validators/current -> ValidatorSet

// 투표 상태
Store/oracle/votes/{txHash} -> VoteStatus

// 발행 명령 대기열
Store/multisig/commands/{commandId} -> MintCommand
```

### 이벤트 스키마

```go
// Cosmos 이벤트
type CreditMinted struct {
    Issuer      string
    Recipient   string
    Amount      sdk.Int
    OriginTx    string
}

type NettingCompleted struct {
    BankPairs   []string
    NetAmounts  map[string]sdk.Int
    BlockHeight int64
}

// Besu 이벤트
type TransferInitiated struct {
    Sender      address
    Recipient   string
    Amount      uint256
    Nonce       uint256
}

type TransferCompleted struct {
    Recipient   address
    Amount      uint256
    CommandId   bytes32
}
```

## 정확성 속성

*속성은 시스템의 모든 유효한 실행에서 참이어야 하는 특성 또는 동작입니다. 본질적으로 시스템이 수행해야 하는 작업에 대한 공식적인 명세입니다. 속성은 인간이 읽을 수 있는 명세와 기계 검증 가능한 정확성 보장 사이의 다리 역할을 합니다.*

### 속성 반영

사전 작업 분석을 검토한 결과, 다음과 같은 중복성을 식별했습니다:

**중복 제거:**
- 속성 1.1과 1.2는 토큰 소각과 이벤트 발생을 별도로 테스트하지만, 실제로는 하나의 원자적 작업입니다. 이를 하나의 포괄적인 속성으로 결합합니다.
- 속성 3.1, 3.2, 3.3은 모두 합의 메커니즘의 다른 측면을 다루지만, 전체 합의 프로세스를 테스트하는 하나의 속성으로 통합할 수 있습니다.
- 속성 4.2와 4.3은 상계 계산과 실행을 별도로 테스트하지만, 이는 하나의 상계 프로세스로 결합 가능합니다.
- 속성 5.1, 5.2, 5.3은 모두 서명 프로세스의 다른 단계를 다루므로 하나의 포괄적인 서명 속성으로 통합합니다.

**최종 속성:**
중복 제거 후 12개의 고유한 속성이 남았으며, 각각은 시스템의 서로 다른 측면을 검증합니다.

### 속성 1: 토큰 소각 및 이벤트 발생
*임의의* 유효한 사용자와 금액에 대해, 은행간 이체를 시작하면 정확한 금액이 소각되고 올바른 TransferInitiated 이벤트가 발생해야 합니다
**검증: 요구사항 1.1, 1.2**

### 속성 2: 이체 완료 시 1:1 대응
*임의의* 이체 시퀀스에 대해, 모든 체인에서 총 소각된 금액과 총 발행된 금액이 일치해야 합니다
**검증: 요구사항 1.5**

### 속성 3: 신용 토큰 발행 및 전송
*임의의* 검증된 은행간 이체에 대해, 올바른 형식의 신용 토큰이 발행되고 목적지 은행 계정으로 전송되어야 합니다
**검증: 요구사항 2.1, 2.2**

### 속성 4: 신용 잔액 조회 정확성
*임의의* 부채 상태에 대해, 신용 잔액 조회는 모든 은행 쌍 간의 정확한 부채 포지션을 반환해야 합니다
**검증: 요구사항 2.4**

### 속성 5: 신용 토큰 추적성
*임의의* 신용 토큰에 대해, 원래 이체의 불변 기록을 조회할 수 있어야 합니다
**검증: 요구사항 2.5**

### 속성 6: 합의 메커니즘
*임의의* TransferInitiated 이벤트에 대해, 3분의 2 이상의 유효한 검증자 투표가 있을 때만 이체가 확정되고 신용 토큰 발행이 트리거되어야 합니다
**검증: 요구사항 3.1, 3.2, 3.3, 3.4**

### 속성 7: 서명 검증
*임의의* 서명에 대해, 등록된 공개 키를 사용한 검증이 올바르게 수행되어야 합니다
**검증: 요구사항 3.5**

### 속성 8: 주기적 상계 트리거
*임의의* 블록 높이에 대해, 10블록마다 상계 프로세스가 트리거되고 모든 은행의 신용 토큰 잔액이 스캔되어야 합니다
**검증: 요구사항 4.1**

### 속성 9: 상계 계산 및 실행
*임의의* 상호 신용 토큰 보유 상황에 대해, 최소 중복 금액이 올바르게 계산되고 해당 금액이 양 은행에서 소각되어야 합니다
**검증: 요구사항 4.2, 4.3**

### 속성 10: 상계 완료 후 상태 업데이트
*임의의* 상계 작업에 대해, 완료 후 영향받은 모든 은행의 순 부채 포지션이 정확하게 업데이트되고 NettingCompleted 이벤트가 발생해야 합니다
**검증: 요구사항 4.4, 4.5**

### 속성 11: 다중 서명 프로세스
*임의의* 검증된 이체에 대해, 올바른 발행 명령이 생성되고 3분의 2 이상의 검증자로부터 ECDSA 서명이 수집되어야 합니다
**검증: 요구사항 5.1, 5.2, 5.3**

### 속성 12: 스마트 컨트랙트 서명 검증
*임의의* 발행 명령에 대해, Executor_Contract는 ecrecover를 사용하여 서명을 검증하고, 유효하지 않은 서명은 거부해야 합니다
**검증: 요구사항 5.4, 5.5**

### 속성 13: 검증자 세트 동기화
*임의의* 검증자 세트 변경에 대해, 모든 연결된 Besu 네트워크에 변경사항이 전파되고 3분의 2 임계값이 유지되어야 합니다
**검증: 요구사항 6.1, 6.2, 6.3, 6.4, 6.5**

### 속성 14: 감사 로깅
*임의의* 시스템 작업(이체, 상계, 토큰 발행/소각)에 대해, 완전한 세부사항과 추적성을 가진 로그가 기록되어야 합니다
**검증: 요구사항 7.1, 7.2, 7.3**

### 속성 15: 감사 쿼리
*임의의* 기간에 대해, 감사 쿼리는 해당 기간의 완전한 거래 이력을 제공해야 합니다
**검증: 요구사항 7.4**

## 오류 처리

### 네트워크 오류
- **연결 실패**: Relayer는 지수 백오프로 재시도
- **부분 실패**: 각 체인별로 독립적인 상태 관리
- **타임아웃**: 설정 가능한 타임아웃으로 데드락 방지

### 합의 실패
- **불충분한 투표**: 이벤트 거부 및 상태 유지
- **검증자 오프라인**: 동적 임계값 조정 (최소 3분의 2 유지)
- **서명 오류**: 개별 서명 검증 실패 시 해당 서명만 제외

### 상계 오류
- **계산 오류**: 상계 중단 및 이전 상태 복원
- **부분 소각 실패**: 원자적 트랜잭션으로 전체 롤백
- **동시성 문제**: 블록 높이 기반 순차 처리

### 스마트 컨트랙트 오류
- **가스 부족**: 자동 가스 추정 및 여유분 추가
- **서명 검증 실패**: 명령 거부 및 이벤트 로깅
- **권한 오류**: 검증자 세트 불일치 감지 및 동기화

## 테스트 전략

### 이중 테스트 접근법

시스템은 단위 테스트와 속성 기반 테스트의 상호 보완적 접근법을 사용합니다:

**단위 테스트**:
- 구체적인 예제와 엣지 케이스 검증
- 컴포넌트 간 통합 지점 테스트
- 오류 조건 및 예외 상황 처리

**속성 기반 테스트**:
- 모든 입력에 대해 성립해야 하는 보편적 속성 검증
- 최소 100회 반복 실행으로 랜덤 입력 테스트
- 각 테스트는 설계 문서의 정확성 속성과 명시적으로 연결

### 속성 기반 테스트 요구사항

**테스트 라이브러리**: Go 언어용 `gopter` 라이브러리 사용
**반복 횟수**: 각 속성 테스트는 최소 100회 반복 실행
**태깅 형식**: 각 속성 기반 테스트는 다음 형식으로 태깅:
`**Feature: interbank-netting-engine, Property {번호}: {속성 텍스트}**`

**예시**:
```go
// **Feature: interbank-netting-engine, Property 1: 토큰 소각 및 이벤트 발생**
func TestTokenBurnAndEventEmission(t *testing.T) {
    properties := gopter.NewProperties(gopter.DefaultTestParameters())
    properties.Property("burn and emit for any valid transfer", prop.ForAll(
        func(user common.Address, amount *big.Int) bool {
            // 속성 테스트 로직
        },
        genValidUser(), genValidAmount(),
    ))
    properties.TestingRun(t)
}
```

### 테스트 커버리지

**단위 테스트 영역**:
- 개별 함수 및 메서드 동작
- 오류 처리 및 예외 상황
- 모듈 간 인터페이스

**속성 테스트 영역**:
- 시스템 불변성 (속성 2, 10)
- 합의 메커니즘 (속성 6, 7)
- 상계 로직 (속성 8, 9)
- 서명 및 검증 (속성 11, 12)
- 감사 및 추적성 (속성 14, 15)

각 정확성 속성은 단일 속성 기반 테스트로 구현되며, 설계 문서의 해당 속성을 명시적으로 참조합니다.