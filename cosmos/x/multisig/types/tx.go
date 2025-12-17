package types

import "context"

// MsgGenerateMintCommandResponse defines the response for MsgGenerateMintCommand
type MsgGenerateMintCommandResponse struct {
	Success   bool   `json:"success"`
	CommandID string `json:"command_id"`
}

// MsgSignCommandResponse defines the response for MsgSignCommand
type MsgSignCommandResponse struct {
	Success        bool `json:"success"`
	SignatureCount int  `json:"signature_count"`
	ThresholdMet   bool `json:"threshold_met"`
}

// MsgUpdateValidatorSetResponse defines the response for MsgUpdateValidatorSet
type MsgUpdateValidatorSetResponse struct {
	Success   bool   `json:"success"`
	Version   uint64 `json:"version"`
	Threshold int    `json:"threshold"`
}

// MsgAddValidatorResponse defines the response for MsgAddValidator
type MsgAddValidatorResponse struct {
	Success bool `json:"success"`
}

// MsgRemoveValidatorResponse defines the response for MsgRemoveValidator
type MsgRemoveValidatorResponse struct {
	Success bool `json:"success"`
}

// MsgServer defines the msg service for the multisig module
type MsgServer interface {
	GenerateMintCommand(ctx context.Context, msg *MsgGenerateMintCommand) (*MsgGenerateMintCommandResponse, error)
	SignCommand(ctx context.Context, msg *MsgSignCommand) (*MsgSignCommandResponse, error)
	UpdateValidatorSet(ctx context.Context, msg *MsgUpdateValidatorSet) (*MsgUpdateValidatorSetResponse, error)
	AddValidator(ctx context.Context, msg *MsgAddValidator) (*MsgAddValidatorResponse, error)
	RemoveValidator(ctx context.Context, msg *MsgRemoveValidator) (*MsgRemoveValidatorResponse, error)
}

// Placeholder for protobuf service descriptor
// In a real implementation, this would be generated from .proto files
var _Msg_serviceDesc = struct{}{}