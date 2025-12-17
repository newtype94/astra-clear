package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/interbank-netting/cosmos/types"
)

const (
	TypeMsgGenerateMintCommand = "generate_mint_command"
	TypeMsgSignCommand         = "sign_command"
	TypeMsgUpdateValidatorSet  = "update_validator_set"
	TypeMsgAddValidator        = "add_validator"
	TypeMsgRemoveValidator     = "remove_validator"
)

var (
	_ sdk.Msg = &MsgGenerateMintCommand{}
	_ sdk.Msg = &MsgSignCommand{}
	_ sdk.Msg = &MsgUpdateValidatorSet{}
	_ sdk.Msg = &MsgAddValidator{}
	_ sdk.Msg = &MsgRemoveValidator{}
)

// MsgGenerateMintCommand defines a message for generating mint commands
type MsgGenerateMintCommand struct {
	Generator   string  `json:"generator"`
	TargetChain string  `json:"target_chain"`
	Recipient   string  `json:"recipient"`
	Amount      sdk.Int `json:"amount"`
}

// NewMsgGenerateMintCommand creates a new MsgGenerateMintCommand instance
func NewMsgGenerateMintCommand(generator, targetChain, recipient string, amount sdk.Int) *MsgGenerateMintCommand {
	return &MsgGenerateMintCommand{
		Generator:   generator,
		TargetChain: targetChain,
		Recipient:   recipient,
		Amount:      amount,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgGenerateMintCommand) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgGenerateMintCommand) Type() string {
	return TypeMsgGenerateMintCommand
}

// GetSigners implements the sdk.Msg interface
func (msg MsgGenerateMintCommand) GetSigners() []sdk.AccAddress {
	generator, err := sdk.AccAddressFromBech32(msg.Generator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{generator}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgGenerateMintCommand) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgGenerateMintCommand) ValidateBasic() error {
	if msg.Generator == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "generator cannot be empty")
	}
	
	_, err := sdk.AccAddressFromBech32(msg.Generator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid generator address: %s", err)
	}
	
	if msg.TargetChain == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "target chain cannot be empty")
	}
	
	if msg.Recipient == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "recipient cannot be empty")
	}
	
	if msg.Amount.IsNil() || msg.Amount.LTE(sdk.ZeroInt()) {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "amount must be positive")
	}
	
	return nil
}

// MsgSignCommand defines a message for signing commands
type MsgSignCommand struct {
	Signer    string                `json:"signer"`
	CommandID string                `json:"command_id"`
	Signature types.ECDSASignature  `json:"signature"`
}

// NewMsgSignCommand creates a new MsgSignCommand instance
func NewMsgSignCommand(signer, commandID string, signature types.ECDSASignature) *MsgSignCommand {
	return &MsgSignCommand{
		Signer:    signer,
		CommandID: commandID,
		Signature: signature,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgSignCommand) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgSignCommand) Type() string {
	return TypeMsgSignCommand
}

// GetSigners implements the sdk.Msg interface
func (msg MsgSignCommand) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgSignCommand) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgSignCommand) ValidateBasic() error {
	if msg.Signer == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "signer cannot be empty")
	}
	
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address: %s", err)
	}
	
	if msg.CommandID == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "command ID cannot be empty")
	}
	
	if msg.Signature.Validator == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "signature validator cannot be empty")
	}
	
	if len(msg.Signature.R) == 0 || len(msg.Signature.S) == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "signature R and S cannot be empty")
	}
	
	return nil
}

// MsgUpdateValidatorSet defines a message for updating the validator set
type MsgUpdateValidatorSet struct {
	Updater    string              `json:"updater"`
	Validators []types.Validator   `json:"validators"`
}

// NewMsgUpdateValidatorSet creates a new MsgUpdateValidatorSet instance
func NewMsgUpdateValidatorSet(updater string, validators []types.Validator) *MsgUpdateValidatorSet {
	return &MsgUpdateValidatorSet{
		Updater:    updater,
		Validators: validators,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgUpdateValidatorSet) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgUpdateValidatorSet) Type() string {
	return TypeMsgUpdateValidatorSet
}

// GetSigners implements the sdk.Msg interface
func (msg MsgUpdateValidatorSet) GetSigners() []sdk.AccAddress {
	updater, err := sdk.AccAddressFromBech32(msg.Updater)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{updater}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgUpdateValidatorSet) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgUpdateValidatorSet) ValidateBasic() error {
	if msg.Updater == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "updater cannot be empty")
	}
	
	_, err := sdk.AccAddressFromBech32(msg.Updater)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid updater address: %s", err)
	}
	
	if len(msg.Validators) == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "validators cannot be empty")
	}
	
	// Validate each validator
	for i, validator := range msg.Validators {
		if validator.Address == "" {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "validator %d: address cannot be empty", i)
		}
		if len(validator.PubKey) == 0 {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "validator %d: public key cannot be empty", i)
		}
		if validator.Power <= 0 {
			return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "validator %d: power must be positive", i)
		}
	}
	
	return nil
}

// MsgAddValidator defines a message for adding a validator
type MsgAddValidator struct {
	Adder     string           `json:"adder"`
	Validator types.Validator  `json:"validator"`
}

// NewMsgAddValidator creates a new MsgAddValidator instance
func NewMsgAddValidator(adder string, validator types.Validator) *MsgAddValidator {
	return &MsgAddValidator{
		Adder:     adder,
		Validator: validator,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgAddValidator) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgAddValidator) Type() string {
	return TypeMsgAddValidator
}

// GetSigners implements the sdk.Msg interface
func (msg MsgAddValidator) GetSigners() []sdk.AccAddress {
	adder, err := sdk.AccAddressFromBech32(msg.Adder)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{adder}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgAddValidator) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgAddValidator) ValidateBasic() error {
	if msg.Adder == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "adder cannot be empty")
	}
	
	_, err := sdk.AccAddressFromBech32(msg.Adder)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid adder address: %s", err)
	}
	
	if msg.Validator.Address == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "validator address cannot be empty")
	}
	
	if len(msg.Validator.PubKey) == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "validator public key cannot be empty")
	}
	
	if msg.Validator.Power <= 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "validator power must be positive")
	}
	
	return nil
}

// MsgRemoveValidator defines a message for removing a validator
type MsgRemoveValidator struct {
	Remover          string `json:"remover"`
	ValidatorAddress string `json:"validator_address"`
}

// NewMsgRemoveValidator creates a new MsgRemoveValidator instance
func NewMsgRemoveValidator(remover, validatorAddress string) *MsgRemoveValidator {
	return &MsgRemoveValidator{
		Remover:          remover,
		ValidatorAddress: validatorAddress,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgRemoveValidator) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgRemoveValidator) Type() string {
	return TypeMsgRemoveValidator
}

// GetSigners implements the sdk.Msg interface
func (msg MsgRemoveValidator) GetSigners() []sdk.AccAddress {
	remover, err := sdk.AccAddressFromBech32(msg.Remover)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{remover}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgRemoveValidator) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgRemoveValidator) ValidateBasic() error {
	if msg.Remover == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "remover cannot be empty")
	}
	
	_, err := sdk.AccAddressFromBech32(msg.Remover)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid remover address: %s", err)
	}
	
	if msg.ValidatorAddress == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "validator address cannot be empty")
	}
	
	return nil
}