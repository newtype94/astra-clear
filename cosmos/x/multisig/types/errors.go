package types

import (
	"cosmossdk.io/errors"
)

// x/multisig module sentinel errors
var (
	ErrInvalidValidator       = errors.Register(ModuleName, 1, "invalid validator")
	ErrValidatorNotFound      = errors.Register(ModuleName, 2, "validator not found")
	ErrValidatorAlreadyExists = errors.Register(ModuleName, 3, "validator already exists")
	ErrInvalidSignature       = errors.Register(ModuleName, 4, "invalid signature")
	ErrDuplicateSignature     = errors.Register(ModuleName, 5, "duplicate signature")
	ErrCommandNotFound        = errors.Register(ModuleName, 6, "command not found")
	ErrCommandAlreadySigned   = errors.Register(ModuleName, 7, "command already signed by validator")
	ErrInsufficientSignatures = errors.Register(ModuleName, 8, "insufficient signatures")
	ErrInvalidThreshold       = errors.Register(ModuleName, 9, "invalid threshold")
	ErrValidatorSetEmpty      = errors.Register(ModuleName, 10, "validator set cannot be empty")
	ErrInvalidCommandID       = errors.Register(ModuleName, 11, "invalid command ID")
	ErrCommandExpired         = errors.Register(ModuleName, 12, "command expired")
	ErrUnauthorized           = errors.Register(ModuleName, 13, "unauthorized operation")
	ErrInvalidECDSASignature  = errors.Register(ModuleName, 14, "invalid ECDSA signature")
	ErrSignatureVerification  = errors.Register(ModuleName, 15, "signature verification failed")
)