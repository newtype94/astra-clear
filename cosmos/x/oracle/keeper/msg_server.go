package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/interbank-netting/cosmos/types"
	oracletypes "github.com/interbank-netting/cosmos/x/oracle/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) oracletypes.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ oracletypes.MsgServer = msgServer{}

// Vote handles MsgVote messages
func (k msgServer) Vote(goCtx context.Context, msg *oracletypes.MsgVote) (*oracletypes.MsgVoteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Create vote from message
	vote := types.Vote{
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

	return &oracletypes.MsgVoteResponse{
		Success:   true,
		Consensus: consensus,
	}, nil
}