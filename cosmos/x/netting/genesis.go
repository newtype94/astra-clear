package netting

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/interbank-netting/cosmos/types"
	"github.com/interbank-netting/cosmos/x/netting/keeper"
)

// GenesisState defines the netting module's genesis state.
type GenesisState struct {
	CreditTokens    []types.CreditToken    `json:"credit_tokens"`
	NettingCycles   []types.NettingCycle   `json:"netting_cycles"`
	LastNettingBlock int64                 `json:"last_netting_block"`
	Params          Params                 `json:"params"`
}

// Params defines the parameters for the netting module.
type Params struct {
	NettingInterval   int64 `json:"netting_interval"`    // Netting interval in blocks
	MinNettingAmount  int64 `json:"min_netting_amount"`  // Minimum amount for netting
	MaxNettingPairs   int   `json:"max_netting_pairs"`   // Maximum pairs per netting cycle
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		NettingInterval:  10,     // Every 10 blocks
		MinNettingAmount: 1,      // Minimum 1 unit
		MaxNettingPairs:  100,    // Maximum 100 pairs per cycle
	}
}

// DefaultGenesisState returns the default genesis state
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		CreditTokens:     []types.CreditToken{},
		NettingCycles:    []types.NettingCycle{},
		LastNettingBlock: 0,
		Params:           DefaultParams(),
	}
}

// ValidateGenesis validates the netting genesis parameters
func ValidateGenesis(data *GenesisState) error {
	if data.Params.NettingInterval <= 0 {
		return fmt.Errorf("netting interval must be positive: %d", data.Params.NettingInterval)
	}
	
	if data.Params.MinNettingAmount <= 0 {
		return fmt.Errorf("minimum netting amount must be positive: %d", data.Params.MinNettingAmount)
	}
	
	if data.Params.MaxNettingPairs <= 0 {
		return fmt.Errorf("maximum netting pairs must be positive: %d", data.Params.MaxNettingPairs)
	}
	
	if data.LastNettingBlock < 0 {
		return fmt.Errorf("last netting block cannot be negative: %d", data.LastNettingBlock)
	}
	
	// Validate credit tokens
	for i, token := range data.CreditTokens {
		if token.Denom == "" {
			return fmt.Errorf("credit token %d: denom cannot be empty", i)
		}
		if token.IssuerBank == "" {
			return fmt.Errorf("credit token %d: issuer bank cannot be empty", i)
		}
		if token.HolderBank == "" {
			return fmt.Errorf("credit token %d: holder bank cannot be empty", i)
		}
		if token.Amount.IsNil() || token.Amount.LTE(sdk.ZeroInt()) {
			return fmt.Errorf("credit token %d: amount must be positive", i)
		}
	}
	
	// Validate netting cycles
	for i, cycle := range data.NettingCycles {
		if cycle.CycleID == 0 {
			return fmt.Errorf("netting cycle %d: cycle ID cannot be zero", i)
		}
		if cycle.BlockHeight < 0 {
			return fmt.Errorf("netting cycle %d: block height cannot be negative", i)
		}
	}
	
	return nil
}

// InitGenesis initializes the netting module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, keeper keeper.Keeper, genState *GenesisState) {
	// Initialize credit tokens
	for _, token := range genState.CreditTokens {
		// Issue credit token (keeper method would need to be implemented)
		_ = keeper.IssueCreditToken(ctx, token)
	}
	
	// Initialize netting cycles
	for _, cycle := range genState.NettingCycles {
		// Store netting cycle (keeper method would need to be implemented)
		// keeper.SetNettingCycle(ctx, cycle)
	}
	
	// Set last netting block
	// keeper.SetLastNettingBlock(ctx, genState.LastNettingBlock)
	
	// Set parameters (would need parameter store implementation)
	// keeper.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the netting module's exported genesis.
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) *GenesisState {
	genesis := DefaultGenesisState()
	
	// Export credit tokens (would need keeper methods)
	// genesis.CreditTokens = keeper.GetAllCreditTokens(ctx)
	
	// Export netting cycles (would need keeper methods)
	// genesis.NettingCycles = keeper.GetAllNettingCycles(ctx)
	
	// Export last netting block (would need keeper methods)
	// genesis.LastNettingBlock = keeper.GetLastNettingBlock(ctx)
	
	// Export parameters (would need parameter store)
	// genesis.Params = keeper.GetParams(ctx)
	
	return genesis
}

// Import fmt for error formatting
import "fmt"