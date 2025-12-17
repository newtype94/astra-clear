package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/interbank-netting/cosmos/types"
)

const (
	TypeMsgIssueCreditToken = "issue_credit_token"
	TypeMsgBurnCreditToken  = "burn_credit_token"
	TypeMsgTriggerNetting   = "trigger_netting"
)

var (
	_ sdk.Msg = &MsgIssueCreditToken{}
	_ sdk.Msg = &MsgBurnCreditToken{}
	_ sdk.Msg = &MsgTriggerNetting{}
)

// MsgIssueCreditToken defines a message for issuing credit tokens
type MsgIssueCreditToken struct {
	Issuer      string             `json:"issuer"`
	CreditToken types.CreditToken  `json:"credit_token"`
}

// NewMsgIssueCreditToken creates a new MsgIssueCreditToken instance
func NewMsgIssueCreditToken(issuer string, creditToken types.CreditToken) *MsgIssueCreditToken {
	return &MsgIssueCreditToken{
		Issuer:      issuer,
		CreditToken: creditToken,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgIssueCreditToken) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgIssueCreditToken) Type() string {
	return TypeMsgIssueCreditToken
}

// GetSigners implements the sdk.Msg interface
func (msg MsgIssueCreditToken) GetSigners() []sdk.AccAddress {
	issuer, err := sdk.AccAddressFromBech32(msg.Issuer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{issuer}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgIssueCreditToken) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgIssueCreditToken) ValidateBasic() error {
	if msg.Issuer == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "issuer cannot be empty")
	}
	
	_, err := sdk.AccAddressFromBech32(msg.Issuer)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid issuer address: %s", err)
	}
	
	if msg.CreditToken.Denom == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "credit token denom cannot be empty")
	}
	
	if msg.CreditToken.IssuerBank == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "issuer bank cannot be empty")
	}
	
	if msg.CreditToken.HolderBank == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "holder bank cannot be empty")
	}
	
	if msg.CreditToken.Amount.IsNil() || msg.CreditToken.Amount.LTE(sdk.ZeroInt()) {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "credit token amount must be positive")
	}
	
	if msg.CreditToken.OriginTx == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "origin transaction cannot be empty")
	}
	
	return nil
}

// MsgBurnCreditToken defines a message for burning credit tokens
type MsgBurnCreditToken struct {
	Burner string  `json:"burner"`
	Denom  string  `json:"denom"`
	Amount sdk.Int `json:"amount"`
}

// NewMsgBurnCreditToken creates a new MsgBurnCreditToken instance
func NewMsgBurnCreditToken(burner, denom string, amount sdk.Int) *MsgBurnCreditToken {
	return &MsgBurnCreditToken{
		Burner: burner,
		Denom:  denom,
		Amount: amount,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgBurnCreditToken) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgBurnCreditToken) Type() string {
	return TypeMsgBurnCreditToken
}

// GetSigners implements the sdk.Msg interface
func (msg MsgBurnCreditToken) GetSigners() []sdk.AccAddress {
	burner, err := sdk.AccAddressFromBech32(msg.Burner)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{burner}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgBurnCreditToken) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgBurnCreditToken) ValidateBasic() error {
	if msg.Burner == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "burner cannot be empty")
	}
	
	_, err := sdk.AccAddressFromBech32(msg.Burner)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid burner address: %s", err)
	}
	
	if msg.Denom == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "denom cannot be empty")
	}
	
	if msg.Amount.IsNil() || msg.Amount.LTE(sdk.ZeroInt()) {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "amount must be positive")
	}
	
	return nil
}

// MsgTriggerNetting defines a message for triggering netting process
type MsgTriggerNetting struct {
	Triggerer string `json:"triggerer"`
}

// NewMsgTriggerNetting creates a new MsgTriggerNetting instance
func NewMsgTriggerNetting(triggerer string) *MsgTriggerNetting {
	return &MsgTriggerNetting{
		Triggerer: triggerer,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgTriggerNetting) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgTriggerNetting) Type() string {
	return TypeMsgTriggerNetting
}

// GetSigners implements the sdk.Msg interface
func (msg MsgTriggerNetting) GetSigners() []sdk.AccAddress {
	triggerer, err := sdk.AccAddressFromBech32(msg.Triggerer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{triggerer}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgTriggerNetting) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgTriggerNetting) ValidateBasic() error {
	if msg.Triggerer == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "triggerer cannot be empty")
	}
	
	_, err := sdk.AccAddressFromBech32(msg.Triggerer)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid triggerer address: %s", err)
	}
	
	return nil
}