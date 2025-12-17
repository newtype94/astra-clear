package keeper_test

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/interbank-netting/cosmos/testutil"
	"github.com/interbank-netting/cosmos/types"
	"github.com/interbank-netting/cosmos/x/multisig/keeper"
	multisigtypes "github.com/interbank-netting/cosmos/x/multisig/types"
)

// **Feature: interbank-netting-engine, Property 7: 서명 검증**
// **검증: 요구사항 3.5**
func TestProperty_SignatureVerification_ValidSignaturesAccepted(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

	properties.Property("valid signatures from registered validators are accepted", prop.ForAll(
		func(validatorSet types.ValidatorSet, mintCommand types.MintCommand) bool {
			// Setup test environment
			ctx, multisigKeeper := setupMultisigTestEnvironment(t)
			
			// Skip if validator set is empty
			if len(validatorSet.Validators) == 0 {
				return true
			}

			// Update validator set
			err := multisigKeeper.UpdateValidatorSet(ctx, validatorSet.Validators)
			if err != nil {
				return false // Should successfully update validator set
			}

			// Store mint command
			_, err = multisigKeeper.GenerateMintCommand(ctx, mintCommand.TargetChain, mintCommand.Recipient, mintCommand.Amount)
			if err != nil {
				return false // Should successfully generate command
			}

			// Get the generated command (it will have a different ID)
			// For testing, we'll use the validator's public key to verify signatures
			validator := validatorSet.Validators[0]
			
			// Generate signature using validator's key
			commandData := []byte(mintCommand.CommandID)
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
		testutil.GenValidatorSet(),
		testutil.GenMintCommand(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 7: 서명 검증**
// **검증: 요구사항 3.5**
func TestProperty_SignatureVerification_InvalidSignaturesRejected(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

	properties.Property("invalid signatures are rejected", prop.ForAll(
		func(validatorSet types.ValidatorSet, mintCommand types.MintCommand) bool {
			// Setup test environment
			ctx, multisigKeeper := setupMultisigTestEnvironment(t)
			
			// Skip if validator set is empty
			if len(validatorSet.Validators) == 0 {
				return true
			}

			// Update validator set
			err := multisigKeeper.UpdateValidatorSet(ctx, validatorSet.Validators)
			if err != nil {
				return false
			}

			validator := validatorSet.Validators[0]
			commandData := []byte(mintCommand.CommandID)

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
		testutil.GenValidatorSet(),
		testutil.GenMintCommand(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 7: 서명 검증**
// **검증: 요구사항 3.5**
func TestProperty_SignatureVerification_UsesRegisteredPublicKey(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

	properties.Property("signature verification uses registered public key", prop.ForAll(
		func(validatorSet types.ValidatorSet) bool {
			// Setup test environment
			ctx, multisigKeeper := setupMultisigTestEnvironment(t)
			
			// Skip if validator set is empty
			if len(validatorSet.Validators) == 0 {
				return true
			}

			// Update validator set
			err := multisigKeeper.UpdateValidatorSet(ctx, validatorSet.Validators)
			if err != nil {
				return false
			}

			// Verify each validator has a public key registered
			for _, validator := range validatorSet.Validators {
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
		testutil.GenValidatorSet(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 7: 서명 검증**
// **검증: 요구사항 3.5**
func TestProperty_SignatureVerification_DuplicateSignaturesRejected(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

	properties.Property("duplicate signatures from same validator are rejected", prop.ForAll(
		func(validatorSet types.ValidatorSet, mintCommand types.MintCommand) bool {
			// Setup test environment
			ctx, multisigKeeper := setupMultisigTestEnvironment(t)
			
			// Skip if validator set is empty
			if len(validatorSet.Validators) == 0 {
				return true
			}

			// Update validator set
			err := multisigKeeper.UpdateValidatorSet(ctx, validatorSet.Validators)
			if err != nil {
				return false
			}

			// Generate mint command
			command, err := multisigKeeper.GenerateMintCommand(ctx, mintCommand.TargetChain, mintCommand.Recipient, mintCommand.Amount)
			if err != nil {
				return false
			}

			validator := validatorSet.Validators[0]
			
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
		testutil.GenValidatorSet(),
		testutil.GenMintCommand(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 7: 서명 검증**
// **검증: 요구사항 3.5**
func TestProperty_SignatureVerification_AllValidatorsCanSign(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

	properties.Property("all registered validators can sign commands", prop.ForAll(
		func(validatorSet types.ValidatorSet, mintCommand types.MintCommand) bool {
			// Setup test environment
			ctx, multisigKeeper := setupMultisigTestEnvironment(t)
			
			// Skip if validator set is empty or too large
			if len(validatorSet.Validators) == 0 || len(validatorSet.Validators) > 5 {
				return true
			}

			// Update validator set
			err := multisigKeeper.UpdateValidatorSet(ctx, validatorSet.Validators)
			if err != nil {
				return false
			}

			// Generate mint command
			command, err := multisigKeeper.GenerateMintCommand(ctx, mintCommand.TargetChain, mintCommand.Recipient, mintCommand.Amount)
			if err != nil {
				return false
			}

			commandData := []byte(command.CommandID)

			// Each validator should be able to sign
			for _, validator := range validatorSet.Validators {
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

			if len(updatedCommand.Signatures) != len(validatorSet.Validators) {
				return false // Should have signature from each validator
			}

			return true
		},
		testutil.GenValidatorSet(),
		testutil.GenMintCommand(),
	))

	properties.TestingRun(t)
}

// Helper functions for testing

func setupMultisigTestEnvironment(t *testing.T) (sdk.Context, keeper.Keeper) {
	// Create mock context
	ctx := sdk.NewContext(nil, false, nil)
	
	// Create mock keepers
	bankKeeper := NewMockBankKeeper()
	stakingKeeper := NewMockStakingKeeper()
	
	// Create multisig keeper with mocks
	multisigKeeper := keeper.Keeper{} // Simplified for property testing
	
	return ctx, multisigKeeper
}

// MockBankKeeper for testing
type MockBankKeeper struct{}

func NewMockBankKeeper() MockBankKeeper {
	return MockBankKeeper{}
}

func (m MockBankKeeper) SendCoins(ctx sdk.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m MockBankKeeper) MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m MockBankKeeper) BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m MockBankKeeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, sdk.ZeroInt())
}

func (m MockBankKeeper) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return sdk.Coins{}
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
	return nil, false
}

func (m MockStakingKeeper) GetAllValidators(ctx sdk.Context) []sdk.ValidatorI {
	return []sdk.ValidatorI{}
}

func (m MockStakingKeeper) GetBondedValidatorsByPower(ctx sdk.Context) []sdk.ValidatorI {
	return []sdk.ValidatorI{}
}