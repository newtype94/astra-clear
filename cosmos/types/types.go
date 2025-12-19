package types

import (
	"cosmossdk.io/math"
)

// TransferEvent represents a cross-chain transfer event from Besu networks
type TransferEvent struct {
	TxHash      string    `json:"tx_hash"`
	Sender      string    `json:"sender"`
	Recipient   string    `json:"recipient"`
	Amount      math.Int  `json:"amount"`
	Nonce       uint64    `json:"nonce"`
	SourceChain string    `json:"source_chain"`
	DestChain   string    `json:"dest_chain"`
	BlockHeight uint64    `json:"block_height"`
	Timestamp   int64     `json:"timestamp"`
}

// Vote represents a validator's vote on a transfer event
type Vote struct {
	TxHash      string        `json:"tx_hash"`
	Validator   string        `json:"validator"`
	EventData   TransferEvent `json:"event_data"`
	Signature   []byte        `json:"signature"`
	VoteTime    int64         `json:"vote_time"`
}

// VoteStatus tracks the voting status for a transfer event
type VoteStatus struct {
	TxHash      string `json:"tx_hash"`
	Votes       []Vote `json:"votes"`
	Confirmed   bool   `json:"confirmed"`
	Threshold   int    `json:"threshold"`
	VoteCount   int    `json:"vote_count"`
	CreatedAt   int64  `json:"created_at"`
	ConfirmedAt int64  `json:"confirmed_at"`
}

// CreditToken represents a bank-issued credit token (IOU)
type CreditToken struct {
	Denom       string    `json:"denom"`        // Format: "cred-{BankID}"
	IssuerBank  string    `json:"issuer_bank"`  // Bank that issued this credit
	HolderBank  string    `json:"holder_bank"`  // Bank that holds this credit
	Amount      math.Int  `json:"amount"`       // Amount of credit
	OriginTx    string    `json:"origin_tx"`    // Original transfer transaction hash
	IssuedAt    int64     `json:"issued_at"`    // Timestamp when issued
}

// NettingCycle represents a netting operation cycle
type NettingCycle struct {
	CycleID     uint64                `json:"cycle_id"`
	BlockHeight int64                 `json:"block_height"`
	Pairs       []BankPair            `json:"pairs"`
	NetAmounts  map[string]math.Int   `json:"net_amounts"`
	StartTime   int64                 `json:"start_time"`
	EndTime     int64                 `json:"end_time"`
	Status      NettingStatus         `json:"status"`
}

// BankPair represents a pair of banks involved in netting
type BankPair struct {
	BankA     string    `json:"bank_a"`
	BankB     string    `json:"bank_b"`
	AmountA   math.Int  `json:"amount_a"`     // Amount A owes to B
	AmountB   math.Int  `json:"amount_b"`     // Amount B owes to A
	NetAmount math.Int  `json:"net_amount"`   // Net amount after netting
	NetDebtor string    `json:"net_debtor"`   // Which bank owes the net amount
}

// NettingStatus represents the status of a netting cycle
type NettingStatus int

const (
	NettingStatusPending NettingStatus = iota
	NettingStatusInProgress
	NettingStatusCompleted
	NettingStatusFailed
)

// ValidatorSet represents the set of validators for multi-signature operations
type ValidatorSet struct {
	Validators   []Validator `json:"validators"`
	Threshold    int         `json:"threshold"`    // Minimum signatures required (2/3)
	UpdateHeight int64       `json:"update_height"`
	Version      uint64      `json:"version"`
}

// Validator represents a validator in the set
type Validator struct {
	Address   string `json:"address"`
	PubKey    []byte `json:"pub_key"`    // ECDSA public key
	Power     int64  `json:"power"`      // Voting power
	Active    bool   `json:"active"`     // Whether validator is active
	JoinedAt  int64  `json:"joined_at"`  // When validator joined
}

// MintCommand represents a command to mint tokens on a destination chain
type MintCommand struct {
	CommandID   string            `json:"command_id"`
	TargetChain string            `json:"target_chain"`
	Recipient   string            `json:"recipient"`
	Amount      math.Int          `json:"amount"`
	Signatures  []ECDSASignature  `json:"signatures"`
	CreatedAt   int64             `json:"created_at"`
	Status      CommandStatus     `json:"status"`
}

// ECDSASignature represents an ECDSA signature
type ECDSASignature struct {
	Validator string `json:"validator"`
	R         []byte `json:"r"`
	S         []byte `json:"s"`
	V         uint8  `json:"v"`
	Timestamp int64  `json:"timestamp"`
}

// CommandStatus represents the status of a mint command
type CommandStatus int

const (
	CommandStatusPending CommandStatus = iota
	CommandStatusSigned
	CommandStatusExecuted
	CommandStatusFailed
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID          uint64            `json:"id"`
	EventType   string            `json:"event_type"`
	TxHash      string            `json:"tx_hash"`
	Details     map[string]string `json:"details"`
	Timestamp   int64             `json:"timestamp"`
	BlockHeight int64             `json:"block_height"`
}

// EventType constants for audit logging
const (
	EventTypeTransferInitiated = "transfer_initiated"
	EventTypeTransferConfirmed = "transfer_confirmed"
	EventTypeCreditIssued      = "credit_issued"
	EventTypeCreditBurned      = "credit_burned"
	EventTypeNettingStarted    = "netting_started"
	EventTypeNettingCompleted  = "netting_completed"
	EventTypeValidatorAdded    = "validator_added"
	EventTypeValidatorRemoved  = "validator_removed"
)