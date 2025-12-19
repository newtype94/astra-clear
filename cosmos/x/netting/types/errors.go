package types

import (
	"cosmossdk.io/errors"
)

// x/netting module sentinel errors
var (
	ErrInvalidCreditToken     = errors.Register(ModuleName, 1, "invalid credit token")
	ErrInsufficientBalance    = errors.Register(ModuleName, 2, "insufficient credit balance")
	ErrCreditTokenNotFound    = errors.Register(ModuleName, 3, "credit token not found")
	ErrInvalidBankID          = errors.Register(ModuleName, 4, "invalid bank ID")
	ErrNettingInProgress      = errors.Register(ModuleName, 5, "netting already in progress")
	ErrNettingFailed          = errors.Register(ModuleName, 6, "netting process failed")
	ErrInvalidNettingCycle    = errors.Register(ModuleName, 7, "invalid netting cycle")
	ErrDuplicateCreditToken   = errors.Register(ModuleName, 8, "duplicate credit token")
	ErrInvalidAmount          = errors.Register(ModuleName, 9, "invalid amount")
	ErrUnauthorized           = errors.Register(ModuleName, 10, "unauthorized operation")
	ErrNettingNotRequired     = errors.Register(ModuleName, 11, "netting not required")
	ErrInvalidDebtPosition    = errors.Register(ModuleName, 12, "invalid debt position")
)