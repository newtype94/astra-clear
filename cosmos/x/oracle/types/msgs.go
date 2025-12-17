package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/interbank-netting/cosmos/types"
)

const (
	TypeMsgVote = "vote"
)

var _ sdk.Msg = &MsgVote{}

// MsgVote defines a message for submitting a vote on a transfer event
type MsgVote struct {
	TxHash      string             `json:"tx_hash"`
	Validator   string             `json:"validator"`
	EventData   types.TransferEvent `json:"event_data"`
	Signature   []byte             `json:"signature"`
}

// NewMsgVote creates a new MsgVote instance
func NewMsgVote(txHash, validator string, eventData types.TransferEvent, signature []byte) *MsgVote {
	return &MsgVote{
		TxHash:    txHash,
		Validator: validator,
		EventData: eventData,
		Signature: signature,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgVote) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgVote) Type() string {
	return TypeMsgVote
}

// GetSigners implements the sdk.Msg interface
func (msg MsgVote) GetSigners() []sdk.AccAddress {
	validator, err := sdk.AccAddressFromBech32(msg.Validator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{validator}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgVote) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgVote) ValidateBasic() error {
	if msg.TxHash == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "tx hash cannot be empty")
	}
	
	if msg.Validator == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "validator cannot be empty")
	}
	
	_, err := sdk.AccAddressFromBech32(msg.Validator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid validator address: %s", err)
	}
	
	if msg.EventData.TxHash != msg.TxHash {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "event data tx hash must match message tx hash")
	}
	
	if msg.EventData.Sender == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "event data sender cannot be empty")
	}
	
	if msg.EventData.Recipient == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "event data recipient cannot be empty")
	}
	
	if msg.EventData.Amount.IsNil() || msg.EventData.Amount.LTE(sdk.ZeroInt()) {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "event data amount must be positive")
	}
	
	if msg.EventData.SourceChain == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "event data source chain cannot be empty")
	}
	
	if msg.EventData.DestChain == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "event data dest chain cannot be empty")
	}
	
	if len(msg.Signature) == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "signature cannot be empty")
	}
	
	return nil
}