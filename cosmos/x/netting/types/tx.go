package types

import "context"

// MsgIssueCreditTokenResponse defines the response for MsgIssueCreditToken
type MsgIssueCreditTokenResponse struct {
	Success bool   `json:"success"`
	Denom   string `json:"denom"`
}

// MsgBurnCreditTokenResponse defines the response for MsgBurnCreditToken
type MsgBurnCreditTokenResponse struct {
	Success bool `json:"success"`
}

// MsgTriggerNettingResponse defines the response for MsgTriggerNetting
type MsgTriggerNettingResponse struct {
	Success  bool   `json:"success"`
	CycleID  uint64 `json:"cycle_id"`
	NetCount int    `json:"net_count"`
}

// MsgServer defines the msg service for the netting module
type MsgServer interface {
	IssueCreditToken(ctx context.Context, msg *MsgIssueCreditToken) (*MsgIssueCreditTokenResponse, error)
	BurnCreditToken(ctx context.Context, msg *MsgBurnCreditToken) (*MsgBurnCreditTokenResponse, error)
	TriggerNetting(ctx context.Context, msg *MsgTriggerNetting) (*MsgTriggerNettingResponse, error)
}

// Placeholder for protobuf service descriptor
// In a real implementation, this would be generated from .proto files
var _Msg_serviceDesc = struct{}{}