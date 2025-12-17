package keeper_test

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"

	"github.com/interbank-netting/cosmos/testutil"
	"github.com/interbank-netting/cosmos/types"
	"github.com/interbank-netting/cosmos/x/oracle/keeper"
	oracletypes "github.com/interbank-netting/cosmos/x/oracle/types"
)

// **Feature: interbank-netting-engine, Property 6: 합의 메커니즘**
// **검증: 요구사항 3.1, 3.2, 3.3, 3.4**
func TestProperty_ConsensusReached_OnlyWith2ThirdsMajority(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

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
		testutil.GenTransferEvent(),
		gen.IntRange(1, 10), // Test with 1-10 validators
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 6: 합의 메커니즘**  
// **검증: 요구사항 3.5**
func TestProperty_SignatureVerification_RejectsInvalidSignatures(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

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
		testutil.GenTransferEvent(),
		gen.IntRange(1, 5), // Test with 1-5 validators
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 6: 합의 메커니즘**
// **검증: 요구사항 3.1, 3.4**
func TestProperty_DuplicateVotes_AreRejected(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

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
		testutil.GenTransferEvent(),
	))

	properties.TestingRun(t)
}

// Helper functions for testing

func setupTestEnvironment(t *testing.T, validatorCount int) (sdk.Context, keeper.Keeper, MockStakingKeeper) {
	// Create mock context
	ctx := sdk.NewContext(nil, false, nil)
	
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
			Address: testdata.CreateTestPubKeys(1)[0].Address().String(),
			PubKey:  testdata.CreateTestPubKeys(1)[0].Bytes(),
			Power:   1,
			Active:  true,
			JoinedAt: 0,
		}
	}
	return validators
}

func setupValidators(ctx sdk.Context, stakingKeeper MockStakingKeeper, validators []types.Validator) {
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

func NewMockStakingKeeper() MockStakingKeeper {
	return MockStakingKeeper{
		validators: make(map[string]types.Validator),
	}
}

func (m MockStakingKeeper) SetValidator(validator types.Validator) {
	m.validators[validator.Address] = validator
}

func (m MockStakingKeeper) GetValidator(ctx sdk.Context, addr sdk.ValAddress) (sdk.ValidatorI, bool) {
	// Mock implementation for testing
	return nil, false
}

func (m MockStakingKeeper) GetAllValidators(ctx sdk.Context) []sdk.ValidatorI {
	// Mock implementation for testing
	return []sdk.ValidatorI{}
}

func (m MockStakingKeeper) GetBondedValidatorsByPower(ctx sdk.Context) []sdk.ValidatorI {
	// Mock implementation for testing
	validators := make([]sdk.ValidatorI, 0, len(m.validators))
	// Convert to ValidatorI interface (simplified for testing)
	return validators
}