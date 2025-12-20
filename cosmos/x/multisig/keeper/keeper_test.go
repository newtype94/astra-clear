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

// **Feature: interbank-netting-engine, Property 8: 다중 서명 임계값**
// **검증: 요구사항 5.3 - 검증자의 최소 3분의 2로부터 서명을 요구**
func TestProperty_MultisigThreshold_TwoThirdsMajority(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	properties.Property("command requires 2/3 validator signatures to be marked as signed", prop.ForAll(
		func(validatorCount int) bool {
			// Need at least 1 validator
			if validatorCount <= 0 || validatorCount > 10 {
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

			// Verify threshold calculation (2/3 rounded up)
			validatorSet := multisigKeeper.GetValidatorSet(ctx)
			expectedThreshold := (validatorCount * 2) / 3
			if (validatorCount*2)%3 != 0 {
				expectedThreshold++
			}
			if expectedThreshold < 1 {
				expectedThreshold = 1
			}

			if int(validatorSet.Threshold) != expectedThreshold {
				return false // Threshold should be 2/3 majority
			}

			// Generate mint command
			command, err := multisigKeeper.GenerateMintCommand(ctx, "bank-a", "recipient1", math.NewInt(1000))
			if err != nil {
				return false
			}

			// Add signatures one by one until threshold
			for i := 0; i < len(validators); i++ {
				validator := validators[i]
				commandData := []byte(command.CommandID)
				signature, err := multisigKeeper.SignData(ctx, validator.Address, commandData)
				if err != nil {
					return false
				}

				err = multisigKeeper.AddSignatureToCommand(ctx, command.CommandID, signature)
				if err != nil {
					return false
				}

				// Check command status after adding signature
				updatedCommand, found := multisigKeeper.GetCommand(ctx, command.CommandID)
				if !found {
					return false
				}

				// Before threshold: should be pending
				// At or after threshold: should be signed
				signatureCount := i + 1
				if signatureCount >= expectedThreshold {
					// Should be marked as signed
					if updatedCommand.Status != int32(types.CommandStatusSigned) {
						return false // Command should be signed when threshold reached
					}
				} else {
					// Should still be pending
					if updatedCommand.Status != int32(types.CommandStatusPending) {
						return false // Command should be pending before threshold
					}
				}
			}

			return true
		},
		gen.IntRange(1, 10), // Test with 1-10 validators
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 8: 다중 서명 임계값**
// **검증: 요구사항 5.3 - 서명이 부족하면 명령 거부**
func TestProperty_MultisigThreshold_InsufficientSignaturesNotConfirmed(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	properties.Property("command with insufficient signatures remains pending", prop.ForAll(
		func(validatorCount int) bool {
			// Need at least 3 validators to have meaningful test
			if validatorCount < 3 || validatorCount > 10 {
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

			validatorSet := multisigKeeper.GetValidatorSet(ctx)
			threshold := int(validatorSet.Threshold)

			// Generate mint command
			command, err := multisigKeeper.GenerateMintCommand(ctx, "bank-a", "recipient1", math.NewInt(1000))
			if err != nil {
				return false
			}

			// Add signatures but stay below threshold
			signaturesNeeded := threshold - 1
			if signaturesNeeded <= 0 {
				signaturesNeeded = 0
			}

			for i := 0; i < signaturesNeeded; i++ {
				validator := validators[i]
				commandData := []byte(command.CommandID)
				signature, err := multisigKeeper.SignData(ctx, validator.Address, commandData)
				if err != nil {
					return false
				}

				err = multisigKeeper.AddSignatureToCommand(ctx, command.CommandID, signature)
				if err != nil {
					return false
				}
			}

			// Command should still be pending (not enough signatures)
			updatedCommand, found := multisigKeeper.GetCommand(ctx, command.CommandID)
			if !found {
				return false
			}

			if updatedCommand.Status != int32(types.CommandStatusPending) {
				return false // Command should remain pending with insufficient signatures
			}

			return true
		},
		gen.IntRange(3, 10), // Test with 3-10 validators
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 9: 발행 명령 수집**
// **검증: 요구사항 5.2 - 활성 검증자들로부터 ECDSA 서명을 수집**
func TestProperty_ProcessPendingCommands_CollectsSignatures(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	properties.Property("ProcessPendingCommands collects signatures from active validators", prop.ForAll(
		func(validatorCount int) bool {
			// Need at least 1 validator
			if validatorCount <= 0 || validatorCount > 5 {
				return true
			}

			// Setup test environment
			ctx, multisigKeeper := setupMultisigTestEnvironment(t)

			// Generate validators (all active)
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

			// Verify command is pending
			pendingCommands := multisigKeeper.GetAllPendingCommands(ctx)
			if len(pendingCommands) != 1 {
				return false // Should have one pending command
			}

			// Process pending commands (simulates EndBlock)
			err = multisigKeeper.ProcessPendingCommands(ctx)
			if err != nil {
				return false
			}

			// Check that signatures were collected
			updatedCommand, found := multisigKeeper.GetCommand(ctx, command.CommandID)
			if !found {
				return false
			}

			// All active validators should have signed
			if len(updatedCommand.Signatures) != validatorCount {
				return false // All validators should have signed
			}

			// Command should be signed if threshold reached
			validatorSet := multisigKeeper.GetValidatorSet(ctx)
			if validatorCount >= int(validatorSet.Threshold) {
				if updatedCommand.Status != int32(types.CommandStatusSigned) {
					return false // Command should be signed
				}
			}

			return true
		},
		gen.IntRange(1, 5), // Test with 1-5 validators
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 9: 발행 명령 수집**
// **검증: 비활성 검증자는 서명에서 제외**
func TestProperty_ProcessPendingCommands_SkipsInactiveValidators(t *testing.T) {
	ctx, multisigKeeper := setupMultisigTestEnvironment(t)

	// Generate 5 validators, 2 inactive
	validators := generateValidators(5)
	validators[2].Active = false
	validators[4].Active = false

	// Update validator set
	err := multisigKeeper.UpdateValidatorSet(ctx, validators)
	require.NoError(t, err)

	// Generate mint command
	command, err := multisigKeeper.GenerateMintCommand(ctx, "bank-a", "recipient1", math.NewInt(1000))
	require.NoError(t, err)

	// Process pending commands
	err = multisigKeeper.ProcessPendingCommands(ctx)
	require.NoError(t, err)

	// Check signatures - only active validators should sign
	updatedCommand, found := multisigKeeper.GetCommand(ctx, command.CommandID)
	require.True(t, found)

	// Should have 3 signatures (from active validators only)
	require.Equal(t, 3, len(updatedCommand.Signatures))

	// Verify inactive validators didn't sign
	for _, sig := range updatedCommand.Signatures {
		require.NotEqual(t, validators[2].Address, sig.Validator)
		require.NotEqual(t, validators[4].Address, sig.Validator)
	}
}

// **Feature: interbank-netting-engine, Property 10: 명령 상태 전환**
// **검증: 요구사항 5.1 - 발행 명령 생성 및 상태 전환**
func TestProperty_CommandStatusTransitions(t *testing.T) {
	ctx, multisigKeeper := setupMultisigTestEnvironment(t)

	// Setup validators
	validators := generateValidators(3)
	err := multisigKeeper.UpdateValidatorSet(ctx, validators)
	require.NoError(t, err)

	// 1. Create command - should be Pending
	command, err := multisigKeeper.GenerateMintCommand(ctx, "bank-a", "recipient1", math.NewInt(1000))
	require.NoError(t, err)
	require.Equal(t, int32(types.CommandStatusPending), command.Status)

	// 2. Process to collect signatures - should become Signed (threshold = 2)
	err = multisigKeeper.ProcessPendingCommands(ctx)
	require.NoError(t, err)

	updatedCommand, found := multisigKeeper.GetCommand(ctx, command.CommandID)
	require.True(t, found)
	require.Equal(t, int32(types.CommandStatusSigned), updatedCommand.Status)

	// 3. Mark as executed - should become Executed
	err = multisigKeeper.MarkCommandExecuted(ctx, command.CommandID)
	require.NoError(t, err)

	finalCommand, found := multisigKeeper.GetCommand(ctx, command.CommandID)
	require.True(t, found)
	require.Equal(t, int32(types.CommandStatusExecuted), finalCommand.Status)
}

// **Unit Test: 명령 쿼리 메서드 검증**
func TestGetCommandsByStatus(t *testing.T) {
	ctx, multisigKeeper := setupMultisigTestEnvironment(t)

	// Setup validators
	validators := generateValidators(3)
	err := multisigKeeper.UpdateValidatorSet(ctx, validators)
	require.NoError(t, err)

	// Create multiple commands
	cmd1, _ := multisigKeeper.GenerateMintCommand(ctx, "bank-a", "recipient1", math.NewInt(1000))
	cmd2, _ := multisigKeeper.GenerateMintCommand(ctx, "bank-b", "recipient2", math.NewInt(2000))
	cmd3, _ := multisigKeeper.GenerateMintCommand(ctx, "bank-c", "recipient3", math.NewInt(3000))

	// Initially all should be pending
	pendingCommands := multisigKeeper.GetAllPendingCommands(ctx)
	require.Equal(t, 3, len(pendingCommands))

	signedCommands := multisigKeeper.GetSignedCommands(ctx)
	require.Equal(t, 0, len(signedCommands))

	// Process first command only - add enough signatures
	for _, v := range validators {
		commandData := []byte(cmd1.CommandID)
		signature, _ := multisigKeeper.SignData(ctx, v.Address, commandData)
		_ = multisigKeeper.AddSignatureToCommand(ctx, cmd1.CommandID, signature)
	}

	// Now should have 2 pending and 1 signed
	pendingCommands = multisigKeeper.GetAllPendingCommands(ctx)
	require.Equal(t, 2, len(pendingCommands))

	signedCommands = multisigKeeper.GetSignedCommands(ctx)
	require.Equal(t, 1, len(signedCommands))
	require.Equal(t, cmd1.CommandID, signedCommands[0].CommandID)

	// Verify pending commands are cmd2 and cmd3
	pendingIDs := make(map[string]bool)
	for _, cmd := range pendingCommands {
		pendingIDs[cmd.CommandID] = true
	}
	require.True(t, pendingIDs[cmd2.CommandID])
	require.True(t, pendingIDs[cmd3.CommandID])
}
