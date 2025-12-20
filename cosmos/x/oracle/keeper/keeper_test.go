package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
			threshold := (validatorCount * 2) / 3
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
			insufficientVotes := threshold - 1
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
			submitVotes(ctx, oracleKeeper, transferEvent, validators[:threshold])

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

func setupTestEnvironment(t *testing.T, validatorCount int) (sdk.Context, keeper.Keeper, *MockStakingKeeper) {
	// Create store key
	storeKey := storetypes.NewKVStoreKey("oracle")

	// Create test context with store
	testCtx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx

	// Create mock staking keeper
	stakingKeeper := NewMockStakingKeeper()

	// Create oracle keeper with mocks
	oracleKeeper := keeper.Keeper{} // Simplified for property testing

	return ctx, oracleKeeper, stakingKeeper
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

func submitVotes(ctx sdk.Context, oracleKeeper keeper.Keeper, transferEvent types.TransferEvent, validators []types.Validator) {
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
	validators map[string]types.Validator
}

func NewMockStakingKeeper() *MockStakingKeeper {
	return &MockStakingKeeper{
		validators: make(map[string]types.Validator),
	}
}

func (m *MockStakingKeeper) SetValidator(validator types.Validator) {
	m.validators[validator.Address] = validator
}

func (m *MockStakingKeeper) GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error) {
	// Mock implementation for testing
	if v, ok := m.validators[addr.String()]; ok {
		return stakingtypes.Validator{
			OperatorAddress: v.Address,
			Status:          stakingtypes.Bonded,
		}, nil
	}
	return stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound
}

func (m *MockStakingKeeper) GetBondedValidatorsByPower(ctx context.Context) ([]stakingtypes.Validator, error) {
	validators := make([]stakingtypes.Validator, 0, len(m.validators))
	for _, v := range m.validators {
		validators = append(validators, stakingtypes.Validator{
			OperatorAddress: v.Address,
			Status:          stakingtypes.Bonded,
		})
	}
	return validators, nil
}

// Silence unused imports
var _ = gopter.Gen(nil)
var _ = require.New(nil)
var _ = log.NewNopLogger()
var _ = math.ZeroInt()
var _ = runtime.NewKVStoreService(nil)
