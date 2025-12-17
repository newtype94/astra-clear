package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/multisig module sentinel errors
var (
	ErrInvalidValidator       = sdkerrors.Register(ModuleName, 1, "invalid validator")
	ErrValidatorNotFound      = sdkerrors.Register(ModuleName, 2, "validator not found")
	ErrValidatorAlreadyExists = sdkerrors.Register(ModuleName, 3, "validator already exists")
	ErrInvalidSignature       = sdkerrors.Register(ModuleName, 4, "invalid signature")
	ErrDuplicateSignature     = sdkerrors.Register(ModuleName, 5, "duplicate signature")
	ErrCommandNotFound        = sdkerrors.Register(ModuleName, 6, "command not found")
	ErrCommandAlreadySigned   = sdkerrors.Register(ModuleName, 7, "command already signed by validator")
	ErrInsufficientSignatures = sdkerrors.Register(ModuleName, 8, "insufficient signatures")
	ErrInvalidThreshold       = sdkerrors.Register(ModuleName, 9, "invalid threshold")
	ErrValidatorSetEmpty      = sdkerrors.Register(ModuleName, 10, "validator set cannot be empty")
	ErrInvalidCommandID       = sdkerrors.Register(ModuleName, 11, "invalid command ID")
	ErrCommandExpired         = sdkerrors.Register(ModuleName, 12, "command expired")
	ErrUnauthorized           = sdkerrors.Register(ModuleName, 13, "unauthorized operation")
	ErrInvalidECDSASignature  = sdkerrors.Register(ModuleName, 14, "invalid ECDSA signature")
	ErrSignatureVerification  = sdkerrors.Register(ModuleName, 15, "signature verification failed")
)