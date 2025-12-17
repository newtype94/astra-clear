package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/netting module sentinel errors
var (
	ErrInvalidCreditToken     = sdkerrors.Register(ModuleName, 1, "invalid credit token")
	ErrInsufficientBalance    = sdkerrors.Register(ModuleName, 2, "insufficient credit balance")
	ErrCreditTokenNotFound    = sdkerrors.Register(ModuleName, 3, "credit token not found")
	ErrInvalidBankID          = sdkerrors.Register(ModuleName, 4, "invalid bank ID")
	ErrNettingInProgress      = sdkerrors.Register(ModuleName, 5, "netting already in progress")
	ErrNettingFailed          = sdkerrors.Register(ModuleName, 6, "netting process failed")
	ErrInvalidNettingCycle    = sdkerrors.Register(ModuleName, 7, "invalid netting cycle")
	ErrDuplicateCreditToken   = sdkerrors.Register(ModuleName, 8, "duplicate credit token")
	ErrInvalidAmount          = sdkerrors.Register(ModuleName, 9, "invalid amount")
	ErrUnauthorized           = sdkerrors.Register(ModuleName, 10, "unauthorized operation")
	ErrNettingNotRequired     = sdkerrors.Register(ModuleName, 11, "netting not required")
	ErrInvalidDebtPosition    = sdkerrors.Register(ModuleName, 12, "invalid debt position")
)