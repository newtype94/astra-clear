package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/oracle module sentinel errors
var (
	ErrInvalidValidator     = sdkerrors.Register(ModuleName, 1, "invalid validator")
	ErrInvalidSignature     = sdkerrors.Register(ModuleName, 2, "invalid signature")
	ErrDuplicateVote        = sdkerrors.Register(ModuleName, 3, "duplicate vote")
	ErrTransferNotFound     = sdkerrors.Register(ModuleName, 4, "transfer not found")
	ErrTransferAlreadyConfirmed = sdkerrors.Register(ModuleName, 5, "transfer already confirmed")
	ErrInsufficientVotes    = sdkerrors.Register(ModuleName, 6, "insufficient votes for consensus")
	ErrInvalidEventData     = sdkerrors.Register(ModuleName, 7, "invalid event data")
	ErrValidatorNotActive   = sdkerrors.Register(ModuleName, 8, "validator not active")
	ErrConsensusTimeout     = sdkerrors.Register(ModuleName, 9, "consensus timeout")
	ErrInvalidTxHash        = sdkerrors.Register(ModuleName, 10, "invalid transaction hash")
)