package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
		return fmt.Errorf("issuer cannot be empty")
	}

	_, err := sdk.AccAddressFromBech32(msg.Issuer)
	if err != nil {
		return fmt.Errorf("invalid issuer address: %w", err)
	}

	if msg.CreditToken.Denom == "" {
		return fmt.Errorf("credit token denom cannot be empty")
	}

	if msg.CreditToken.IssuerBank == "" {
		return fmt.Errorf("issuer bank cannot be empty")
	}

	if msg.CreditToken.HolderBank == "" {
		return fmt.Errorf("holder bank cannot be empty")
	}

	if msg.CreditToken.Amount.IsNil() || msg.CreditToken.Amount.LTE(math.ZeroInt()) {
		return fmt.Errorf("credit token amount must be positive")
	}

	if msg.CreditToken.OriginTx == "" {
		return fmt.Errorf("origin transaction cannot be empty")
	}

	return nil
}

// MsgBurnCreditToken defines a message for burning credit tokens
type MsgBurnCreditToken struct {
	Burner string  `json:"burner"`
	Denom  string  `json:"denom"`
	Amount math.Int `json:"amount"`
}

// NewMsgBurnCreditToken creates a new MsgBurnCreditToken instance
func NewMsgBurnCreditToken(burner, denom string, amount math.Int) *MsgBurnCreditToken {
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
		return fmt.Errorf("burner cannot be empty")
	}

	_, err := sdk.AccAddressFromBech32(msg.Burner)
	if err != nil {
		return fmt.Errorf("invalid burner address: %w", err)
	}

	if msg.Denom == "" {
		return fmt.Errorf("denom cannot be empty")
	}

	if msg.Amount.IsNil() || msg.Amount.LTE(math.ZeroInt()) {
		return fmt.Errorf("amount must be positive")
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
		return fmt.Errorf("triggerer cannot be empty")
	}

	_, err := sdk.AccAddressFromBech32(msg.Triggerer)
	if err != nil {
		return fmt.Errorf("invalid triggerer address: %w", err)
	}

	return nil
}