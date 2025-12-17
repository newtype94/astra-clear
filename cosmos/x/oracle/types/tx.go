package types

// MsgVoteResponse defines the response for MsgVote
type MsgVoteResponse struct {
	Success   bool `json:"success"`
	Consensus bool `json:"consensus"`
}

// MsgServer defines the msg service for the oracle module
type MsgServer interface {
	Vote(ctx context.Context, msg *MsgVote) (*MsgVoteResponse, error)
}

// Placeholder for protobuf service descriptor
// In a real implementation, this would be generated from .proto files
var _Msg_serviceDesc = struct{}{}

// Context import for interface
import "context"