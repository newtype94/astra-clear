package types

import (
	"cosmossdk.io/errors"
)

// x/oracle module sentinel errors
var (
	ErrInvalidValidator     = errors.Register(ModuleName, 1, "invalid validator")
	ErrInvalidSignature     = errors.Register(ModuleName, 2, "invalid signature")
	ErrDuplicateVote        = errors.Register(ModuleName, 3, "duplicate vote")
	ErrTransferNotFound     = errors.Register(ModuleName, 4, "transfer not found")
	ErrTransferAlreadyConfirmed = errors.Register(ModuleName, 5, "transfer already confirmed")
	ErrInsufficientVotes    = errors.Register(ModuleName, 6, "insufficient votes for consensus")
	ErrInvalidEventData     = errors.Register(ModuleName, 7, "invalid event data")
	ErrValidatorNotActive   = errors.Register(ModuleName, 8, "validator not active")
	ErrConsensusTimeout     = errors.Register(ModuleName, 9, "consensus timeout")
	ErrInvalidTxHash        = errors.Register(ModuleName, 10, "invalid transaction hash")
)