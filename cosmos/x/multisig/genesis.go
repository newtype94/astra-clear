package multisig

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/interbank-netting/cosmos/types"
	"github.com/interbank-netting/cosmos/x/multisig/keeper"
)

// GenesisState defines the multisig module's genesis state.
type GenesisState struct {
	ValidatorSet types.ValidatorSet     `json:"validator_set"`
	MintCommands []types.MintCommand    `json:"mint_commands"`
	Params       Params                 `json:"params"`
}

// ProtoMessage implements proto.Message
func (gs *GenesisState) ProtoMessage() {}

// Reset implements proto.Message
func (gs *GenesisState) Reset() { *gs = GenesisState{} }

// String implements proto.Message
func (gs *GenesisState) String() string {
	return fmt.Sprintf("GenesisState{Validators: %d, MintCommands: %d}", len(gs.ValidatorSet.Validators), len(gs.MintCommands))
}

// Params defines the parameters for the multisig module.
type Params struct {
	SigningTimeout    int64 `json:"signing_timeout"`     // Signing timeout in seconds
	MaxCommandAge     int64 `json:"max_command_age"`     // Maximum command age in seconds
	MinValidatorCount int   `json:"min_validator_count"` // Minimum validator count
	MaxValidatorCount int   `json:"max_validator_count"` // Maximum validator count
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		SigningTimeout:    3600, // 1 hour
		MaxCommandAge:     7200, // 2 hours
		MinValidatorCount: 1,    // Minimum 1 validator
		MaxValidatorCount: 100,  // Maximum 100 validators
	}
}

// DefaultGenesisState returns the default genesis state
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		ValidatorSet: types.ValidatorSet{
			Validators:   []types.Validator{},
			Threshold:    1,
			UpdateHeight: 0,
			Version:      1,
		},
		MintCommands: []types.MintCommand{},
		Params:       DefaultParams(),
	}
}

// ValidateGenesis validates the multisig genesis parameters
func ValidateGenesis(data *GenesisState) error {
	if data.Params.SigningTimeout <= 0 {
		return fmt.Errorf("signing timeout must be positive: %d", data.Params.SigningTimeout)
	}
	
	if data.Params.MaxCommandAge <= 0 {
		return fmt.Errorf("max command age must be positive: %d", data.Params.MaxCommandAge)
	}
	
	if data.Params.MinValidatorCount <= 0 {
		return fmt.Errorf("minimum validator count must be positive: %d", data.Params.MinValidatorCount)
	}
	
	if data.Params.MaxValidatorCount <= 0 {
		return fmt.Errorf("maximum validator count must be positive: %d", data.Params.MaxValidatorCount)
	}
	
	if data.Params.MinValidatorCount > data.Params.MaxValidatorCount {
		return fmt.Errorf("minimum validator count cannot be greater than maximum: %d > %d", 
			data.Params.MinValidatorCount, data.Params.MaxValidatorCount)
	}
	
	// Validate validator set
	if data.ValidatorSet.Threshold <= 0 {
		return fmt.Errorf("validator set threshold must be positive: %d", data.ValidatorSet.Threshold)
	}
	
	if len(data.ValidatorSet.Validators) > 0 {
		// Check threshold is reasonable for validator count
		validatorCount := len(data.ValidatorSet.Validators)
		expectedThreshold := (validatorCount * 2) / 3
		if (validatorCount*2)%3 != 0 {
			expectedThreshold++
		}
		
		if data.ValidatorSet.Threshold > validatorCount {
			return fmt.Errorf("threshold cannot be greater than validator count: %d > %d", 
				data.ValidatorSet.Threshold, validatorCount)
		}
	}
	
	// Validate validators
	for i, validator := range data.ValidatorSet.Validators {
		if validator.Address == "" {
			return fmt.Errorf("validator %d: address cannot be empty", i)
		}
		if len(validator.PubKey) == 0 {
			return fmt.Errorf("validator %d: public key cannot be empty", i)
		}
		if validator.Power <= 0 {
			return fmt.Errorf("validator %d: power must be positive", i)
		}
	}
	
	// Validate mint commands
	for i, command := range data.MintCommands {
		if command.CommandID == "" {
			return fmt.Errorf("mint command %d: command ID cannot be empty", i)
		}
		if command.TargetChain == "" {
			return fmt.Errorf("mint command %d: target chain cannot be empty", i)
		}
		if command.Recipient == "" {
			return fmt.Errorf("mint command %d: recipient cannot be empty", i)
		}
		if command.Amount.IsNil() || command.Amount.LTE(math.ZeroInt()) {
			return fmt.Errorf("mint command %d: amount must be positive", i)
		}
	}
	
	return nil
}

// InitGenesis initializes the multisig module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, keeper keeper.Keeper, genState *GenesisState) {
	// Initialize validator set
	if len(genState.ValidatorSet.Validators) > 0 {
		err := keeper.UpdateValidatorSet(ctx, genState.ValidatorSet.Validators)
		if err != nil {
			panic(fmt.Sprintf("failed to initialize validator set: %v", err))
		}
	}
	
	// Initialize mint commands
	// TODO: Implement when keeper methods are available
	_ = genState.MintCommands
	
	// Set parameters (would need parameter store implementation)
	// keeper.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the multisig module's exported genesis.
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) *GenesisState {
	genesis := DefaultGenesisState()
	
	// Export validator set
	genesis.ValidatorSet = keeper.GetValidatorSet(ctx)
	
	// Export mint commands (would need keeper methods)
	// genesis.MintCommands = keeper.GetAllMintCommands(ctx)
	
	// Export parameters (would need parameter store)
	// genesis.Params = keeper.GetParams(ctx)
	
	return genesis
}