package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/interbank-netting/cosmos/types"
	"github.com/interbank-netting/cosmos/x/oracle/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// Vote handles MsgVote messages
func (k msgServer) Vote(goCtx context.Context, msg *types.MsgVote) (*types.MsgVoteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Create vote from message
	vote := commontypes.Vote{
		TxHash:    msg.TxHash,
		Validator: msg.Validator,
		EventData: msg.EventData,
		Signature: msg.Signature,
		VoteTime:  ctx.BlockTime().Unix(),
	}

	// Submit the vote
	if err := k.Keeper.SubmitVote(ctx, vote); err != nil {
		return nil, err
	}

	// Check if consensus was reached
	consensus, err := k.Keeper.CheckConsensus(ctx, msg.TxHash)
	if err != nil {
		return nil, err
	}

	return &types.MsgVoteResponse{
		Success:   true,
		Consensus: consensus,
	}, nil
}