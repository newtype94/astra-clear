package oracle

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/interbank-netting/cosmos/types"
	"github.com/interbank-netting/cosmos/x/oracle/keeper"
)

// GenesisState defines the oracle module's genesis state.
type GenesisState struct {
	VoteStatuses       []types.VoteStatus       `json:"vote_statuses"`
	ConfirmedTransfers []types.TransferEvent    `json:"confirmed_transfers"`
	Params             Params                   `json:"params"`
}

// Params defines the parameters for the oracle module.
type Params struct {
	VotingPeriod      int64 `json:"voting_period"`       // Voting period in seconds
	ConsensusTimeout  int64 `json:"consensus_timeout"`   // Consensus timeout in seconds
	MinValidatorCount int   `json:"min_validator_count"` // Minimum validator count for consensus
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		VotingPeriod:      300,  // 5 minutes
		ConsensusTimeout:  1800, // 30 minutes
		MinValidatorCount: 1,    // Minimum 1 validator
	}
}

// DefaultGenesisState returns the default genesis state
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		VoteStatuses:       []types.VoteStatus{},
		ConfirmedTransfers: []types.TransferEvent{},
		Params:             DefaultParams(),
	}
}

// ValidateGenesis validates the oracle genesis parameters
func ValidateGenesis(data *GenesisState) error {
	if data.Params.VotingPeriod <= 0 {
		return fmt.Errorf("voting period must be positive: %d", data.Params.VotingPeriod)
	}
	
	if data.Params.ConsensusTimeout <= 0 {
		return fmt.Errorf("consensus timeout must be positive: %d", data.Params.ConsensusTimeout)
	}
	
	if data.Params.MinValidatorCount <= 0 {
		return fmt.Errorf("minimum validator count must be positive: %d", data.Params.MinValidatorCount)
	}
	
	return nil
}

// InitGenesis initializes the oracle module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState *GenesisState) {
	// Initialize vote statuses
	// TODO: Implement when keeper methods are available
	_ = genState.VoteStatuses

	// Initialize confirmed transfers
	// TODO: Implement when keeper methods are available
	_ = genState.ConfirmedTransfers

	// Set parameters (would need parameter store implementation)
	// k.SetParams(ctx, genState.Params)
	_ = ctx
	_ = k
}

// ExportGenesis returns the oracle module's exported genesis.
func ExportGenesis(ctx sdk.Context, keeper keeper.Keeper) *GenesisState {
	genesis := DefaultGenesisState()
	
	// Export vote statuses (would need keeper methods)
	// genesis.VoteStatuses = keeper.GetAllVoteStatuses(ctx)
	
	// Export confirmed transfers (would need keeper methods)
	// genesis.ConfirmedTransfers = keeper.GetAllConfirmedTransfers(ctx)
	
	// Export parameters (would need parameter store)
	// genesis.Params = keeper.GetParams(ctx)
	
	return genesis
}