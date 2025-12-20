package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/require"

	testhelpers "github.com/interbank-netting/cosmos/testutil"
	"github.com/interbank-netting/cosmos/types"
	"github.com/interbank-netting/cosmos/x/oracle/keeper"
)

// **Feature: interbank-netting-engine, Property 6: 합의 메커니즘**
// **검증: 요구사항 3.1, 3.2, 3.3, 3.4**
func TestProperty_ConsensusReached_OnlyWith2ThirdsMajority(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("consensus mechanism requires 2/3+ majority", prop.ForAll(
		func(transferEvent types.TransferEvent, validatorCount int) bool {
			// Setup test environment
			ctx, oracleKeeper, stakingKeeper := setupTestEnvironment(t, validatorCount)

			// Calculate 2/3 threshold
			threshold := int32((validatorCount * 2) / 3)
			if (validatorCount*2)%3 != 0 {
				threshold++ // Round up for 2/3+ majority
			}
			if threshold < 1 {
				threshold = 1
			}

			// Generate validators and votes
			validators := generateValidators(validatorCount)
			setupValidators(ctx, stakingKeeper, validators)

			// Test case 1: Insufficient votes (less than 2/3)
			insufficientVotes := int(threshold - 1)
			if insufficientVotes > 0 {
				submitVotes(ctx, oracleKeeper, transferEvent, validators[:insufficientVotes])

				// Check that consensus is NOT reached
				consensus, err := oracleKeeper.CheckConsensus(ctx, transferEvent.TxHash)
				if err != nil || consensus {
					return false // Should not reach consensus with insufficient votes
				}

				// Verify transfer is not confirmed
				voteStatus, found := oracleKeeper.GetVoteStatus(ctx, transferEvent.TxHash)
				if !found || voteStatus.Confirmed {
					return false // Should not be confirmed
				}
			}

			// Test case 2: Sufficient votes (exactly 2/3 or more)
			submitVotes(ctx, oracleKeeper, transferEvent, validators[:int(threshold)])

			// Check that consensus IS reached
			consensus, err := oracleKeeper.CheckConsensus(ctx, transferEvent.TxHash)
			if err != nil || !consensus {
				return false // Should reach consensus with sufficient votes
			}

			// Verify transfer is confirmed
			voteStatus, found := oracleKeeper.GetVoteStatus(ctx, transferEvent.TxHash)
			if !found || !voteStatus.Confirmed {
				return false // Should be confirmed
			}

			// Verify vote count matches threshold
			if voteStatus.VoteCount < threshold {
				return false // Vote count should be at least threshold
			}

			return true
		},
		testhelpers.GenTransferEvent(),
		gen.IntRange(1, 10), // Test with 1-10 validators
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 6: 합의 메커니즘**
// **검증: 요구사항 3.5**
func TestProperty_SignatureVerification_RejectsInvalidSignatures(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("invalid signatures are rejected", prop.ForAll(
		func(transferEvent types.TransferEvent, validatorCount int) bool {
			// Setup test environment
			ctx, oracleKeeper, stakingKeeper := setupTestEnvironment(t, validatorCount)

			// Generate validators
			validators := generateValidators(validatorCount)
			setupValidators(ctx, stakingKeeper, validators)

			if len(validators) == 0 {
				return true // Skip empty validator set
			}

			// Test with invalid signature (empty signature)
			invalidVote := types.Vote{
				TxHash:    transferEvent.TxHash,
				Validator: validators[0].Address,
				EventData: transferEvent,
				Signature: []byte{}, // Invalid empty signature
				VoteTime:  ctx.BlockTime().Unix(),
			}

			// Submit vote with invalid signature
			err := oracleKeeper.SubmitVote(ctx, invalidVote)

			// Should reject invalid signature
			if err == nil {
				return false // Should have returned an error for invalid signature
			}

			// Verify no vote was recorded
			voteStatus, found := oracleKeeper.GetVoteStatus(ctx, transferEvent.TxHash)
			if found && voteStatus.VoteCount > 0 {
				return false // Should not have recorded any votes
			}

			return true
		},
		testhelpers.GenTransferEvent(),
		gen.IntRange(1, 5), // Test with 1-5 validators
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 6: 합의 메커니즘**
// **검증: 요구사항 3.1, 3.4**
func TestProperty_DuplicateVotes_AreRejected(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("duplicate votes from same validator are rejected", prop.ForAll(
		func(transferEvent types.TransferEvent) bool {
			// Setup test environment with single validator
			ctx, oracleKeeper, stakingKeeper := setupTestEnvironment(t, 1)

			// Generate single validator
			validators := generateValidators(1)
			setupValidators(ctx, stakingKeeper, validators)

			if len(validators) == 0 {
				return true // Skip empty validator set
			}

			// Create valid vote
			validVote := types.Vote{
				TxHash:    transferEvent.TxHash,
				Validator: validators[0].Address,
				EventData: transferEvent,
				Signature: []byte("valid_signature"), // Mock valid signature
				VoteTime:  ctx.BlockTime().Unix(),
			}

			// Submit first vote (should succeed)
			err1 := oracleKeeper.SubmitVote(ctx, validVote)
			if err1 != nil {
				return false // First vote should succeed
			}

			// Submit duplicate vote (should fail)
			err2 := oracleKeeper.SubmitVote(ctx, validVote)
			if err2 == nil {
				return false // Duplicate vote should be rejected
			}

			// Verify only one vote was recorded
			voteStatus, found := oracleKeeper.GetVoteStatus(ctx, transferEvent.TxHash)
			if !found || voteStatus.VoteCount != 1 {
				return false // Should have exactly one vote
			}

			return true
		},
		testhelpers.GenTransferEvent(),
	))

	properties.TestingRun(t)
}

// Helper functions for testing

func setupTestEnvironment(t *testing.T, validatorCount int) (sdk.Context, *keeper.Keeper, *MockStakingKeeper) {
	// Create store key
	storeKey := storetypes.NewKVStoreKey("oracle")

	// Create test context with store
	testCtx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx

	// Create proto codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create mock keepers
	mockBankKeeper := NewMockBankKeeper()
	stakingKeeper := NewMockStakingKeeper()

	// Create oracle keeper with proper initialization
	oracleKeeper := keeper.NewKeeper(
		cdc,
		storeKey,
		nil, // memKey
		paramtypes.Subspace{},
		mockBankKeeper,
		stakingKeeper,
	)

	return ctx, oracleKeeper, stakingKeeper
}

// MockBankKeeper for testing
type MockBankKeeper struct{}

func NewMockBankKeeper() *MockBankKeeper {
	return &MockBankKeeper{}
}

func (m *MockBankKeeper) SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, math.ZeroInt())
}

func generateValidators(count int) []types.Validator {
	validators := make([]types.Validator, count)
	for i := 0; i < count; i++ {
		validators[i] = types.Validator{
			Address:  sdk.ValAddress([]byte{byte(i)}).String(),
			PubKey:   []byte{byte(i), byte(i + 1), byte(i + 2)},
			Power:    1,
			Active:   true,
			JoinedAt: 0,
		}
	}
	return validators
}

func setupValidators(ctx sdk.Context, stakingKeeper *MockStakingKeeper, validators []types.Validator) {
	for _, validator := range validators {
		stakingKeeper.SetValidator(validator)
	}
}

func submitVotes(ctx sdk.Context, oracleKeeper *keeper.Keeper, transferEvent types.TransferEvent, validators []types.Validator) {
	for _, validator := range validators {
		vote := types.Vote{
			TxHash:    transferEvent.TxHash,
			Validator: validator.Address,
			EventData: transferEvent,
			Signature: []byte("mock_signature"), // Mock signature for testing
			VoteTime:  ctx.BlockTime().Unix(),
		}

		// Ignore errors in property tests - we're testing the overall behavior
		_ = oracleKeeper.SubmitVote(ctx, vote)
	}
}

// MockStakingKeeper for testing
type MockStakingKeeper struct {
	validators       map[string]types.Validator
	stakingValidator map[string]stakingtypes.Validator
}

func NewMockStakingKeeper() *MockStakingKeeper {
	return &MockStakingKeeper{
		validators:       make(map[string]types.Validator),
		stakingValidator: make(map[string]stakingtypes.Validator),
	}
}

func (m *MockStakingKeeper) SetValidator(validator types.Validator) {
	m.validators[validator.Address] = validator

	// Create a mock staking validator with a proper consensus pubkey
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	pkAny, _ := codectypes.NewAnyWithValue(pubKey)

	stakingVal := stakingtypes.Validator{
		OperatorAddress:   validator.Address,
		ConsensusPubkey:   pkAny,
		Status:            stakingtypes.Bonded,
		Jailed:            false,
		DelegatorShares:   math.LegacyNewDec(1),
		MinSelfDelegation: math.NewInt(1),
	}
	m.stakingValidator[validator.Address] = stakingVal
}

func (m *MockStakingKeeper) GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error) {
	// Mock implementation for testing
	if v, ok := m.stakingValidator[addr.String()]; ok {
		return v, nil
	}
	return stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound
}

func (m *MockStakingKeeper) GetBondedValidatorsByPower(ctx context.Context) ([]stakingtypes.Validator, error) {
	validators := make([]stakingtypes.Validator, 0, len(m.stakingValidator))
	for _, v := range m.stakingValidator {
		validators = append(validators, v)
	}
	return validators, nil
}

// Silence unused imports
var _ = gopter.Gen(nil)
var _ = require.New(nil)
var _ = log.NewNopLogger()
var _ = math.ZeroInt()
var _ = runtime.NewKVStoreService(nil)

// =============================================================================
// **Feature: interbank-netting-engine, Property 14: 감사 로깅**
// **검증: 요구사항 7.1, 7.2, 7.3, 7.5**
// =============================================================================

// TestProperty_AuditLog_SaveAndRetrieve tests that audit logs are correctly saved and retrieved
func TestProperty_AuditLog_SaveAndRetrieve(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("audit logs can be saved and retrieved by ID", prop.ForAll(
		func(eventType string, txHash string) bool {
			ctx, oracleKeeper, _ := setupTestEnvironment(t, 3)

			// Create audit log
			auditLog := types.AuditLog{
				EventType: eventType,
				TxHash:    txHash,
				Timestamp: ctx.BlockTime().Unix(),
				Details: map[string]string{
					"test_key": "test_value",
				},
			}

			// Save audit log
			id, err := oracleKeeper.SaveAuditLog(ctx, auditLog)
			if err != nil {
				return false
			}

			// Retrieve audit log
			retrieved, found := oracleKeeper.GetAuditLog(ctx, id)
			if !found {
				return false
			}

			// Verify fields match
			if retrieved.ID != id {
				return false
			}
			if retrieved.EventType != eventType {
				return false
			}
			if retrieved.TxHash != txHash {
				return false
			}
			if retrieved.Details["test_key"] != "test_value" {
				return false
			}

			return true
		},
		gen.AnyString(),
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// TestProperty_AuditLog_TimeRangeQuery tests time range queries for audit logs
func TestProperty_AuditLog_TimeRangeQuery(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("audit logs can be queried by time range", prop.ForAll(
		func(logCount int) bool {
			ctx, oracleKeeper, _ := setupTestEnvironment(t, 3)

			if logCount <= 0 {
				logCount = 1
			}
			if logCount > 10 {
				logCount = 10
			}

			// Create multiple audit logs with different timestamps
			baseTime := ctx.BlockTime().Unix()
			var savedIDs []uint64

			for i := 0; i < logCount; i++ {
				auditLog := types.AuditLog{
					EventType: types.EventTypeTransferConfirmed,
					TxHash:    "tx-" + string(rune(i+65)), // tx-A, tx-B, etc.
					Timestamp: baseTime + int64(i*10),    // 10 second intervals
					Details:   map[string]string{"index": string(rune(i + 48))},
				}

				id, err := oracleKeeper.SaveAuditLog(ctx, auditLog)
				if err != nil {
					return false
				}
				savedIDs = append(savedIDs, id)
			}

			// Query all logs
			allLogs := oracleKeeper.GetAllAuditLogs(ctx)
			if len(allLogs) != logCount {
				return false
			}

			// Query by time range - should get all logs
			startTime := baseTime - 1
			endTime := baseTime + int64(logCount*10) + 1
			timeRangeLogs := oracleKeeper.GetAuditLogsByTimeRange(ctx, startTime, endTime)
			if len(timeRangeLogs) != logCount {
				return false
			}

			return true
		},
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t)
}

// TestProperty_AuditLog_TypeFilter tests event type filtering for audit logs
func TestProperty_AuditLog_TypeFilter(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("audit logs can be filtered by event type", prop.ForAll(
		func(transferCount, creditCount int) bool {
			ctx, oracleKeeper, _ := setupTestEnvironment(t, 3)

			// Normalize counts
			if transferCount < 0 {
				transferCount = 0
			}
			if transferCount > 5 {
				transferCount = 5
			}
			if creditCount < 0 {
				creditCount = 0
			}
			if creditCount > 5 {
				creditCount = 5
			}

			// Create transfer confirmed logs
			for i := 0; i < transferCount; i++ {
				auditLog := types.AuditLog{
					EventType: types.EventTypeTransferConfirmed,
					TxHash:    "transfer-tx-" + string(rune(i+65)),
					Timestamp: ctx.BlockTime().Unix(),
					Details:   map[string]string{},
				}
				_, err := oracleKeeper.SaveAuditLog(ctx, auditLog)
				if err != nil {
					return false
				}
			}

			// Create credit issued logs
			for i := 0; i < creditCount; i++ {
				auditLog := types.AuditLog{
					EventType: types.EventTypeCreditIssued,
					TxHash:    "credit-tx-" + string(rune(i+65)),
					Timestamp: ctx.BlockTime().Unix(),
					Details:   map[string]string{},
				}
				_, err := oracleKeeper.SaveAuditLog(ctx, auditLog)
				if err != nil {
					return false
				}
			}

			// Filter by transfer confirmed type
			transferLogs := oracleKeeper.GetAuditLogsByEventType(ctx, types.EventTypeTransferConfirmed)
			if len(transferLogs) != transferCount {
				return false
			}

			// Filter by credit issued type
			creditLogs := oracleKeeper.GetAuditLogsByEventType(ctx, types.EventTypeCreditIssued)
			if len(creditLogs) != creditCount {
				return false
			}

			// Verify total count
			allLogs := oracleKeeper.GetAllAuditLogs(ctx)
			if len(allLogs) != transferCount+creditCount {
				return false
			}

			return true
		},
		gen.IntRange(0, 5),
		gen.IntRange(0, 5),
	))

	properties.TestingRun(t)
}

// TestProperty_AuditLog_TxHashTraceability tests tx hash traceability (Requirement 7.3)
func TestProperty_AuditLog_TxHashTraceability(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("audit logs can be traced by transaction hash", prop.ForAll(
		func(txHash string) bool {
			ctx, oracleKeeper, _ := setupTestEnvironment(t, 3)

			if txHash == "" {
				txHash = "test-tx-hash"
			}

			// Create multiple logs with same txHash (e.g., transfer confirmed + credit issued)
			log1 := types.AuditLog{
				EventType: types.EventTypeTransferConfirmed,
				TxHash:    txHash,
				Timestamp: ctx.BlockTime().Unix(),
				Details:   map[string]string{"phase": "1"},
			}
			_, err := oracleKeeper.SaveAuditLog(ctx, log1)
			if err != nil {
				return false
			}

			log2 := types.AuditLog{
				EventType: types.EventTypeCreditIssued,
				TxHash:    txHash,
				Timestamp: ctx.BlockTime().Unix() + 1,
				Details:   map[string]string{"phase": "2"},
			}
			_, err = oracleKeeper.SaveAuditLog(ctx, log2)
			if err != nil {
				return false
			}

			// Create a log with different txHash
			log3 := types.AuditLog{
				EventType: types.EventTypeNettingCompleted,
				TxHash:    "other-tx-hash",
				Timestamp: ctx.BlockTime().Unix() + 2,
				Details:   map[string]string{"phase": "3"},
			}
			_, err = oracleKeeper.SaveAuditLog(ctx, log3)
			if err != nil {
				return false
			}

			// Query by txHash
			logs := oracleKeeper.GetAuditLogsByTxHash(ctx, txHash)

			// Should find exactly 2 logs with the same txHash
			if len(logs) != 2 {
				return false
			}

			// Verify all logs have the correct txHash
			for _, log := range logs {
				if log.TxHash != txHash {
					return false
				}
			}

			return true
		},
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

// TestProperty_AuditLog_AutoIncrementID tests that audit log IDs auto-increment
func TestProperty_AuditLog_AutoIncrementID(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("audit log IDs auto-increment correctly", prop.ForAll(
		func(logCount int) bool {
			ctx, oracleKeeper, _ := setupTestEnvironment(t, 3)

			if logCount <= 0 {
				logCount = 1
			}
			if logCount > 20 {
				logCount = 20
			}

			var previousID uint64 = 0

			for i := 0; i < logCount; i++ {
				auditLog := types.AuditLog{
					EventType: types.EventTypeTransferConfirmed,
					TxHash:    "tx-" + string(rune(i+65)),
					Timestamp: ctx.BlockTime().Unix(),
					Details:   map[string]string{},
				}

				id, err := oracleKeeper.SaveAuditLog(ctx, auditLog)
				if err != nil {
					return false
				}

				// ID should be greater than previous
				if id <= previousID {
					return false
				}
				previousID = id
			}

			// Count should match
			count := oracleKeeper.GetAuditLogCount(ctx)
			if count != uint64(logCount) {
				return false
			}

			return true
		},
		gen.IntRange(1, 20),
	))

	properties.TestingRun(t)
}
