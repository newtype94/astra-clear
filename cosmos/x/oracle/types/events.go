package types

// Oracle module event types
const (
	EventTypeVoteSubmitted     = "vote_submitted"
	EventTypeTransferConfirmed = "transfer_confirmed"
	EventTypeConsensusReached  = "consensus_reached"
	EventTypeVoteRejected      = "vote_rejected"
	EventTypeTransferRejected  = "transfer_rejected"
	EventTypeConsensusTimeout  = "consensus_timeout"
)

// Oracle module event attribute keys
const (
	AttributeKeyTxHash      = "tx_hash"
	AttributeKeyValidator   = "validator"
	AttributeKeySender      = "sender"
	AttributeKeyRecipient   = "recipient"
	AttributeKeyAmount      = "amount"
	AttributeKeySourceChain = "source_chain"
	AttributeKeyDestChain   = "dest_chain"
	AttributeKeyVoteCount   = "vote_count"
	AttributeKeyThreshold   = "threshold"
	AttributeKeyReason      = "reason"
)