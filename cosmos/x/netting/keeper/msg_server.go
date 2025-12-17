package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	nettingtypes "github.com/interbank-netting/cosmos/x/netting/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) nettingtypes.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ nettingtypes.MsgServer = msgServer{}

// IssueCreditToken handles MsgIssueCreditToken messages
func (k msgServer) IssueCreditToken(goCtx context.Context, msg *nettingtypes.MsgIssueCreditToken) (*nettingtypes.MsgIssueCreditTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Issue the credit token
	if err := k.Keeper.IssueCreditToken(ctx, msg.CreditToken); err != nil {
		return nil, err
	}

	return &nettingtypes.MsgIssueCreditTokenResponse{
		Success: true,
		Denom:   msg.CreditToken.Denom,
	}, nil
}

// BurnCreditToken handles MsgBurnCreditToken messages
func (k msgServer) BurnCreditToken(goCtx context.Context, msg *nettingtypes.MsgBurnCreditToken) (*nettingtypes.MsgBurnCreditTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Burn the credit token
	if err := k.Keeper.BurnCreditToken(ctx, msg.Denom, msg.Amount); err != nil {
		return nil, err
	}

	return &nettingtypes.MsgBurnCreditTokenResponse{
		Success: true,
	}, nil
}

// TriggerNetting handles MsgTriggerNetting messages
func (k msgServer) TriggerNetting(goCtx context.Context, msg *nettingtypes.MsgTriggerNetting) (*nettingtypes.MsgTriggerNettingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Trigger netting process
	if err := k.Keeper.TriggerNetting(ctx); err != nil {
		return nil, err
	}

	// Get the latest netting cycle ID (current block height)
	cycleID := uint64(ctx.BlockHeight())
	
	// Calculate number of pairs that were netted
	pairs, err := k.Keeper.CalculateNetting(ctx)
	if err != nil {
		return nil, err
	}

	return &nettingtypes.MsgTriggerNettingResponse{
		Success:  true,
		CycleID:  cycleID,
		NetCount: len(pairs),
	}, nil
}