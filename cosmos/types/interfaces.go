package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// OracleKeeper defines the expected interface for the oracle module
type OracleKeeper interface {
	// Vote collection and consensus
	SubmitVote(ctx sdk.Context, vote Vote) error
	GetVoteStatus(ctx sdk.Context, txHash string) (VoteStatus, bool)
	CheckConsensus(ctx sdk.Context, txHash string) (bool, error)
	ConfirmTransfer(ctx sdk.Context, txHash string) error
	
	// Validator management
	IsActiveValidator(ctx sdk.Context, validator string) bool
	GetValidatorPubKey(ctx sdk.Context, validator string) ([]byte, bool)
	VerifySignature(ctx sdk.Context, validator string, data []byte, signature []byte) bool
}

// NettingKeeper defines the expected interface for the netting module
type NettingKeeper interface {
	// Credit token management
	IssueCreditToken(ctx sdk.Context, token CreditToken) error
	BurnCreditToken(ctx sdk.Context, denom string, amount math.Int) error
	TransferCreditToken(ctx sdk.Context, from, to, denom string, amount math.Int) error

	// Balance queries
	GetCreditBalance(ctx sdk.Context, bank, denom string) math.Int
	GetAllCreditBalances(ctx sdk.Context, bank string) map[string]math.Int
	GetDebtPosition(ctx sdk.Context, bankA, bankB string) (math.Int, math.Int)

	// Netting operations
	TriggerNetting(ctx sdk.Context) error
	CalculateNetting(ctx sdk.Context) ([]BankPair, error)
	ExecuteNetting(ctx sdk.Context, pairs []BankPair) error
	GetNettingCycle(ctx sdk.Context, cycleID uint64) (NettingCycle, bool)
}

// MultisigKeeper defines the expected interface for the multisig module
type MultisigKeeper interface {
	// Validator set management
	GetValidatorSet(ctx sdk.Context) ValidatorSet
	UpdateValidatorSet(ctx sdk.Context, validators []Validator) error
	AddValidator(ctx sdk.Context, validator Validator) error
	RemoveValidator(ctx sdk.Context, address string) error

	// Command generation and signing
	GenerateMintCommand(ctx sdk.Context, targetChain, recipient string, amount math.Int) (MintCommand, error)
	CollectSignatures(ctx sdk.Context, commandID string) error
	VerifyCommand(ctx sdk.Context, command MintCommand) bool
	GetCommand(ctx sdk.Context, commandID string) (MintCommand, bool)

	// ECDSA operations
	SignData(ctx sdk.Context, validator string, data []byte) (ECDSASignature, error)
	VerifyECDSASignature(ctx sdk.Context, data []byte, signature ECDSASignature) bool
}

// BankKeeper defines the expected interface for the bank module
type BankKeeper interface {
	SendCoins(ctx sdk.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
}

// AccountKeeper defines the expected interface for the account module
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx sdk.Context, acc sdk.AccountI)
	NewAccountWithAddress(ctx sdk.Context, addr sdk.AccAddress) sdk.AccountI
}

// StakingKeeper defines the expected interface for the staking module
type StakingKeeper interface {
	GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator stakingtypes.ValidatorI, found bool)
	GetAllValidators(ctx sdk.Context) (validators []stakingtypes.ValidatorI)
	GetBondedValidatorsByPower(ctx sdk.Context) []stakingtypes.ValidatorI
}

// EventEmitter defines the interface for emitting blockchain events
type EventEmitter interface {
	EmitTransferConfirmed(ctx sdk.Context, event TransferEvent)
	EmitCreditIssued(ctx sdk.Context, token CreditToken)
	EmitCreditBurned(ctx sdk.Context, denom string, amount math.Int)
	EmitNettingCompleted(ctx sdk.Context, cycle NettingCycle)
	EmitMintCommandGenerated(ctx sdk.Context, command MintCommand)
	EmitValidatorSetUpdated(ctx sdk.Context, validatorSet ValidatorSet)
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogTransfer(ctx sdk.Context, event TransferEvent) error
	LogNetting(ctx sdk.Context, cycle NettingCycle) error
	LogCreditOperation(ctx sdk.Context, operation string, token CreditToken) error
	LogValidatorOperation(ctx sdk.Context, operation string, validator Validator) error
	QueryLogs(ctx sdk.Context, startTime, endTime int64, eventType string) ([]AuditLog, error)
}

// RelayerInterface defines the interface for external relayer communication
type RelayerInterface interface {
	// Event monitoring
	MonitorBesuEvents(chainID string, callback func(TransferEvent)) error
	MonitorCosmosEvents(callback func(MintCommand)) error
	
	// Command execution
	ExecuteBesuCommand(chainID string, command MintCommand) error
	SendCosmosTransaction(msg sdk.Msg) error
	
	// Status checking
	GetChainStatus(chainID string) (bool, error)
	GetTransactionStatus(chainID, txHash string) (string, error)
}