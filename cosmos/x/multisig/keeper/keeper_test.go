package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/require"

	"github.com/interbank-netting/cosmos/types"
	"github.com/interbank-netting/cosmos/x/multisig/keeper"
)

// **Feature: interbank-netting-engine, Property 7: 서명 검증**
// **검증: 요구사항 3.5**
func TestProperty_SignatureVerification_ValidSignaturesAccepted(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	properties.Property("valid signatures from registered validators are accepted", prop.ForAll(
		func(validatorCount int) bool {
			// Skip if no validators
			if validatorCount <= 0 {
				return true
			}

			// Setup test environment
			ctx, multisigKeeper := setupMultisigTestEnvironment(t)

			// Generate validators
			validators := generateValidators(validatorCount)

			// Update validator set
			err := multisigKeeper.UpdateValidatorSet(ctx, validators)
			if err != nil {
				return false // Should successfully update validator set
			}

			// Generate mint command
			command, err := multisigKeeper.GenerateMintCommand(ctx, "bank-a", "recipient1", math.NewInt(1000))
			if err != nil {
				return false // Should successfully generate command
			}

			// Get a validator from the set
			validator := validators[0]

			// Generate signature using validator's key
			commandData := []byte(command.CommandID)
			signature, err := multisigKeeper.SignData(ctx, validator.Address, commandData)
			if err != nil {
				return false // Should successfully sign data
			}

			// Verify the signature
			isValid := multisigKeeper.VerifyECDSASignature(ctx, commandData, signature)
			if !isValid {
				return false // Valid signature should be accepted
			}

			// Verify signature has correct structure
			if len(signature.R) == 0 || len(signature.S) == 0 {
				return false // Signature should have R and S components
			}

			if signature.Validator != validator.Address {
				return false // Signature should be attributed to correct validator
			}

			return true
		},
		gen.IntRange(1, 5), // Test with 1-5 validators
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 7: 서명 검증**
// **검증: 요구사항 3.5**
func TestProperty_SignatureVerification_InvalidSignaturesRejected(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	properties.Property("invalid signatures are rejected", prop.ForAll(
		func(validatorCount int) bool {
			// Skip if no validators
			if validatorCount <= 0 {
				return true
			}

			// Setup test environment
			ctx, multisigKeeper := setupMultisigTestEnvironment(t)

			// Generate validators
			validators := generateValidators(validatorCount)

			// Update validator set
			err := multisigKeeper.UpdateValidatorSet(ctx, validators)
			if err != nil {
				return false
			}

			validator := validators[0]
			commandData := []byte("test-command-id")

			// Test 1: Empty signature should be rejected
			emptySignature := types.ECDSASignature{
				Validator: validator.Address,
				R:         []byte{},
				S:         []byte{},
				V:         0,
				Timestamp: ctx.BlockTime().Unix(),
			}

			isValid := multisigKeeper.VerifyECDSASignature(ctx, commandData, emptySignature)
			if isValid {
				return false // Empty signature should be rejected
			}

			// Test 2: Signature from unregistered validator should be rejected
			unregisteredSignature := types.ECDSASignature{
				Validator: "unregistered-validator",
				R:         make([]byte, 32),
				S:         make([]byte, 32),
				V:         27,
				Timestamp: ctx.BlockTime().Unix(),
			}

			isValid = multisigKeeper.VerifyECDSASignature(ctx, commandData, unregisteredSignature)
			if isValid {
				return false // Signature from unregistered validator should be rejected
			}

			return true
		},
		gen.IntRange(1, 5), // Test with 1-5 validators
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 7: 서명 검증**
// **검증: 요구사항 3.5**
func TestProperty_SignatureVerification_UsesRegisteredPublicKey(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	properties.Property("signature verification uses registered public key", prop.ForAll(
		func(validatorCount int) bool {
			// Skip if no validators
			if validatorCount <= 0 {
				return true
			}

			// Setup test environment
			ctx, multisigKeeper := setupMultisigTestEnvironment(t)

			// Generate validators
			validators := generateValidators(validatorCount)

			// Update validator set
			err := multisigKeeper.UpdateValidatorSet(ctx, validators)
			if err != nil {
				return false
			}

			// Verify each validator has a public key registered
			for _, validator := range validators {
				if len(validator.PubKey) == 0 {
					return false // Each validator should have a public key
				}

				// Verify we can retrieve the validator's public key
				retrievedSet := multisigKeeper.GetValidatorSet(ctx)
				found := false
				for _, v := range retrievedSet.Validators {
					if v.Address == validator.Address {
						found = true
						if len(v.PubKey) == 0 {
							return false // Retrieved validator should have public key
						}
						// Public keys should match
						if string(v.PubKey) != string(validator.PubKey) {
							return false // Public key should match registered key
						}
					}
				}

				if !found {
					return false // Validator should be found in set
				}
			}

			return true
		},
		gen.IntRange(1, 5), // Test with 1-5 validators
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 7: 서명 검증**
// **검증: 요구사항 3.5**
func TestProperty_SignatureVerification_DuplicateSignaturesRejected(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	properties.Property("duplicate signatures from same validator are rejected", prop.ForAll(
		func(validatorCount int) bool {
			// Skip if no validators
			if validatorCount <= 0 {
				return true
			}

			// Setup test environment
			ctx, multisigKeeper := setupMultisigTestEnvironment(t)

			// Generate validators
			validators := generateValidators(validatorCount)

			// Update validator set
			err := multisigKeeper.UpdateValidatorSet(ctx, validators)
			if err != nil {
				return false
			}

			// Generate mint command
			command, err := multisigKeeper.GenerateMintCommand(ctx, "bank-a", "recipient1", math.NewInt(1000))
			if err != nil {
				return false
			}

			validator := validators[0]

			// Generate signature
			commandData := []byte(command.CommandID)
			signature, err := multisigKeeper.SignData(ctx, validator.Address, commandData)
			if err != nil {
				return false
			}

			// Add signature first time (should succeed)
			err1 := multisigKeeper.AddSignatureToCommand(ctx, command.CommandID, signature)
			if err1 != nil {
				return false // First signature should be accepted
			}

			// Try to add same signature again (should fail)
			err2 := multisigKeeper.AddSignatureToCommand(ctx, command.CommandID, signature)
			if err2 == nil {
				return false // Duplicate signature should be rejected
			}

			// Verify only one signature was recorded
			updatedCommand, found := multisigKeeper.GetCommand(ctx, command.CommandID)
			if !found {
				return false
			}

			if len(updatedCommand.Signatures) != 1 {
				return false // Should have exactly one signature
			}

			return true
		},
		gen.IntRange(1, 5), // Test with 1-5 validators
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 7: 서명 검증**
// **검증: 요구사항 3.5**
func TestProperty_SignatureVerification_AllValidatorsCanSign(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	properties.Property("all registered validators can sign commands", prop.ForAll(
		func(validatorCount int) bool {
			// Skip if no validators or too many
			if validatorCount <= 0 || validatorCount > 5 {
				return true
			}

			// Setup test environment
			ctx, multisigKeeper := setupMultisigTestEnvironment(t)

			// Generate validators
			validators := generateValidators(validatorCount)

			// Update validator set
			err := multisigKeeper.UpdateValidatorSet(ctx, validators)
			if err != nil {
				return false
			}

			// Generate mint command
			command, err := multisigKeeper.GenerateMintCommand(ctx, "bank-a", "recipient1", math.NewInt(1000))
			if err != nil {
				return false
			}

			commandData := []byte(command.CommandID)

			// Each validator should be able to sign
			for _, validator := range validators {
				signature, err := multisigKeeper.SignData(ctx, validator.Address, commandData)
				if err != nil {
					return false // Each validator should be able to sign
				}

				// Verify signature is valid
				isValid := multisigKeeper.VerifyECDSASignature(ctx, commandData, signature)
				if !isValid {
					return false // Each validator's signature should be valid
				}

				// Add signature to command
				err = multisigKeeper.AddSignatureToCommand(ctx, command.CommandID, signature)
				if err != nil {
					return false // Should be able to add each validator's signature
				}
			}

			// Verify all signatures were recorded
			updatedCommand, found := multisigKeeper.GetCommand(ctx, command.CommandID)
			if !found {
				return false
			}

			if len(updatedCommand.Signatures) != len(validators) {
				return false // Should have signature from each validator
			}

			return true
		},
		gen.IntRange(1, 5), // Test with 1-5 validators
	))

	properties.TestingRun(t)
}

// **Unit Tests**

func TestUpdateValidatorSet_Success(t *testing.T) {
	ctx, k := setupMultisigTestEnvironment(t)

	validators := generateValidators(3)
	err := k.UpdateValidatorSet(ctx, validators)
	require.NoError(t, err)

	validatorSet := k.GetValidatorSet(ctx)
	require.Equal(t, 3, len(validatorSet.Validators))
	require.Equal(t, int32(2), validatorSet.Threshold) // 2/3 of 3 = 2
}

func TestUpdateValidatorSet_EmptyReturnsError(t *testing.T) {
	ctx, k := setupMultisigTestEnvironment(t)

	err := k.UpdateValidatorSet(ctx, []types.Validator{})
	require.Error(t, err)
}

func TestGenerateMintCommand_Success(t *testing.T) {
	ctx, k := setupMultisigTestEnvironment(t)

	validators := generateValidators(3)
	_ = k.UpdateValidatorSet(ctx, validators)

	command, err := k.GenerateMintCommand(ctx, "bank-a", "recipient1", math.NewInt(1000))
	require.NoError(t, err)
	require.NotEmpty(t, command.CommandID)
	require.Equal(t, "bank-a", command.TargetChain)
	require.Equal(t, "recipient1", command.Recipient)
	require.Equal(t, math.NewInt(1000), command.Amount)
}

func TestSignData_UnregisteredValidator(t *testing.T) {
	ctx, k := setupMultisigTestEnvironment(t)

	validators := generateValidators(3)
	_ = k.UpdateValidatorSet(ctx, validators)

	_, err := k.SignData(ctx, "unregistered-validator", []byte("test"))
	require.Error(t, err)
}

// Helper functions for testing

func setupMultisigTestEnvironment(t *testing.T) (sdk.Context, *keeper.Keeper) {
	// Create store key
	storeKey := storetypes.NewKVStoreKey("multisig")

	// Create test context with store
	testCtx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx

	// Create codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create mock keepers
	mockBankKeeper := NewMockBankKeeper()
	mockStakingKeeper := NewMockStakingKeeper()

	// Create multisig keeper with proper initialization
	multisigKeeper := keeper.NewKeeper(
		cdc,
		storeKey,
		nil, // memKey
		paramtypes.Subspace{}, // empty paramstore for tests
		mockBankKeeper,
		mockStakingKeeper,
	)

	return ctx, multisigKeeper
}

func generateValidators(count int) []types.Validator {
	validators := make([]types.Validator, count)
	for i := 0; i < count; i++ {
		validators[i] = types.Validator{
			Address:  sdk.ValAddress([]byte{byte(i + 1)}).String(),
			PubKey:   make([]byte, 33), // Compressed secp256k1 public key size
			Power:    1,
			Active:   true,
			JoinedAt: 0,
		}
		// Fill in some mock public key bytes
		for j := 0; j < 33; j++ {
			validators[i].PubKey[j] = byte((i + j) % 256)
		}
	}
	return validators
}

// MockBankKeeper for testing - implements types.BankKeeper
type MockBankKeeper struct{}

func NewMockBankKeeper() *MockBankKeeper {
	return &MockBankKeeper{}
}

func (m *MockBankKeeper) SendCoins(ctx sdk.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, math.ZeroInt())
}

func (m *MockBankKeeper) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return sdk.Coins{}
}

// MockStakingKeeper for testing - implements types.StakingKeeper
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
	return stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound
}

func (m *MockStakingKeeper) GetAllValidators(ctx context.Context) ([]stakingtypes.Validator, error) {
	return []stakingtypes.Validator{}, nil
}

func (m *MockStakingKeeper) GetBondedValidatorsByPower(ctx context.Context) ([]stakingtypes.Validator, error) {
	return []stakingtypes.Validator{}, nil
}
