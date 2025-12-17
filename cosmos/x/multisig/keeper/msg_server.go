package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	multisigtypes "github.com/interbank-netting/cosmos/x/multisig/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) multisigtypes.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ multisigtypes.MsgServer = msgServer{}

// GenerateMintCommand handles MsgGenerateMintCommand messages
func (k msgServer) GenerateMintCommand(goCtx context.Context, msg *multisigtypes.MsgGenerateMintCommand) (*multisigtypes.MsgGenerateMintCommandResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Generate mint command
	command, err := k.Keeper.GenerateMintCommand(ctx, msg.TargetChain, msg.Recipient, msg.Amount)
	if err != nil {
		return nil, err
	}

	return &multisigtypes.MsgGenerateMintCommandResponse{
		Success:   true,
		CommandID: command.CommandID,
	}, nil
}

// SignCommand handles MsgSignCommand messages
func (k msgServer) SignCommand(goCtx context.Context, msg *multisigtypes.MsgSignCommand) (*multisigtypes.MsgSignCommandResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Add signature to command
	err := k.Keeper.AddSignatureToCommand(ctx, msg.CommandID, msg.Signature)
	if err != nil {
		return nil, err
	}

	// Get updated command to check signature count and threshold status
	command, found := k.Keeper.GetCommand(ctx, msg.CommandID)
	if !found {
		return nil, multisigtypes.ErrCommandNotFound
	}

	validatorSet := k.Keeper.GetValidatorSet(ctx)
	thresholdMet := len(command.Signatures) >= validatorSet.Threshold

	return &multisigtypes.MsgSignCommandResponse{
		Success:        true,
		SignatureCount: len(command.Signatures),
		ThresholdMet:   thresholdMet,
	}, nil
}

// UpdateValidatorSet handles MsgUpdateValidatorSet messages
func (k msgServer) UpdateValidatorSet(goCtx context.Context, msg *multisigtypes.MsgUpdateValidatorSet) (*multisigtypes.MsgUpdateValidatorSetResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Update validator set
	err := k.Keeper.UpdateValidatorSet(ctx, msg.Validators)
	if err != nil {
		return nil, err
	}

	// Get updated validator set info
	validatorSet := k.Keeper.GetValidatorSet(ctx)

	return &multisigtypes.MsgUpdateValidatorSetResponse{
		Success:   true,
		Version:   validatorSet.Version,
		Threshold: validatorSet.Threshold,
	}, nil
}

// AddValidator handles MsgAddValidator messages
func (k msgServer) AddValidator(goCtx context.Context, msg *multisigtypes.MsgAddValidator) (*multisigtypes.MsgAddValidatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Add validator
	err := k.Keeper.AddValidator(ctx, msg.Validator)
	if err != nil {
		return nil, err
	}

	return &multisigtypes.MsgAddValidatorResponse{
		Success: true,
	}, nil
}

// RemoveValidator handles MsgRemoveValidator messages
func (k msgServer) RemoveValidator(goCtx context.Context, msg *multisigtypes.MsgRemoveValidator) (*multisigtypes.MsgRemoveValidatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Remove validator
	err := k.Keeper.RemoveValidator(ctx, msg.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	return &multisigtypes.MsgRemoveValidatorResponse{
		Success: true,
	}, nil
}