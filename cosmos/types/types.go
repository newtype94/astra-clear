package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// TransferEvent represents a cross-chain transfer event from Besu networks
type TransferEvent struct {
	TxHash      string   `protobuf:"bytes,1,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash"`
	Sender      string   `protobuf:"bytes,2,opt,name=sender,proto3" json:"sender"`
	Recipient   string   `protobuf:"bytes,3,opt,name=recipient,proto3" json:"recipient"`
	Amount      math.Int `protobuf:"bytes,4,opt,name=amount,proto3,customtype=cosmossdk.io/math.Int" json:"amount"`
	Nonce       uint64   `protobuf:"varint,5,opt,name=nonce,proto3" json:"nonce"`
	SourceChain string   `protobuf:"bytes,6,opt,name=source_chain,json=sourceChain,proto3" json:"source_chain"`
	DestChain   string   `protobuf:"bytes,7,opt,name=dest_chain,json=destChain,proto3" json:"dest_chain"`
	BlockHeight uint64   `protobuf:"varint,8,opt,name=block_height,json=blockHeight,proto3" json:"block_height"`
	Timestamp   int64    `protobuf:"varint,9,opt,name=timestamp,proto3" json:"timestamp"`
}

func (t *TransferEvent) ProtoMessage()  {}
func (t *TransferEvent) Reset()         { *t = TransferEvent{} }
func (t *TransferEvent) String() string { return fmt.Sprintf("TransferEvent{TxHash: %s}", t.TxHash) }

// Vote represents a validator's vote on a transfer event
type Vote struct {
	TxHash    string        `protobuf:"bytes,1,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash"`
	Validator string        `protobuf:"bytes,2,opt,name=validator,proto3" json:"validator"`
	EventData TransferEvent `protobuf:"bytes,3,opt,name=event_data,json=eventData,proto3" json:"event_data"`
	Signature []byte        `protobuf:"bytes,4,opt,name=signature,proto3" json:"signature"`
	VoteTime  int64         `protobuf:"varint,5,opt,name=vote_time,json=voteTime,proto3" json:"vote_time"`
}

func (v *Vote) ProtoMessage()  {}
func (v *Vote) Reset()         { *v = Vote{} }
func (v *Vote) String() string { return fmt.Sprintf("Vote{TxHash: %s, Validator: %s}", v.TxHash, v.Validator) }

// VoteStatus tracks the voting status for a transfer event
type VoteStatus struct {
	TxHash      string `protobuf:"bytes,1,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash"`
	Votes       []Vote `protobuf:"bytes,2,rep,name=votes,proto3" json:"votes"`
	Confirmed   bool   `protobuf:"varint,3,opt,name=confirmed,proto3" json:"confirmed"`
	Threshold   int32  `protobuf:"varint,4,opt,name=threshold,proto3" json:"threshold"`
	VoteCount   int32  `protobuf:"varint,5,opt,name=vote_count,json=voteCount,proto3" json:"vote_count"`
	CreatedAt   int64  `protobuf:"varint,6,opt,name=created_at,json=createdAt,proto3" json:"created_at"`
	ConfirmedAt int64  `protobuf:"varint,7,opt,name=confirmed_at,json=confirmedAt,proto3" json:"confirmed_at"`
}

func (vs *VoteStatus) ProtoMessage()  {}
func (vs *VoteStatus) Reset()         { *vs = VoteStatus{} }
func (vs *VoteStatus) String() string {
	return fmt.Sprintf("VoteStatus{TxHash: %s, Confirmed: %v}", vs.TxHash, vs.Confirmed)
}

// CreditToken represents a bank-issued credit token (IOU)
type CreditToken struct {
	Denom      string   `protobuf:"bytes,1,opt,name=denom,proto3" json:"denom"`
	IssuerBank string   `protobuf:"bytes,2,opt,name=issuer_bank,json=issuerBank,proto3" json:"issuer_bank"`
	HolderBank string   `protobuf:"bytes,3,opt,name=holder_bank,json=holderBank,proto3" json:"holder_bank"`
	Amount     math.Int `protobuf:"bytes,4,opt,name=amount,proto3,customtype=cosmossdk.io/math.Int" json:"amount"`
	OriginTx   string   `protobuf:"bytes,5,opt,name=origin_tx,json=originTx,proto3" json:"origin_tx"`
	IssuedAt   int64    `protobuf:"varint,6,opt,name=issued_at,json=issuedAt,proto3" json:"issued_at"`
}

func (ct *CreditToken) ProtoMessage()  {}
func (ct *CreditToken) Reset()         { *ct = CreditToken{} }
func (ct *CreditToken) String() string {
	return fmt.Sprintf("CreditToken{Denom: %s, Amount: %s}", ct.Denom, ct.Amount.String())
}

// NettingCycle represents a netting operation cycle
type NettingCycle struct {
	CycleID     uint64              `protobuf:"varint,1,opt,name=cycle_id,json=cycleId,proto3" json:"cycle_id"`
	BlockHeight int64               `protobuf:"varint,2,opt,name=block_height,json=blockHeight,proto3" json:"block_height"`
	Pairs       []BankPair          `protobuf:"bytes,3,rep,name=pairs,proto3" json:"pairs"`
	NetAmounts  map[string]math.Int `protobuf:"bytes,4,rep,name=net_amounts,json=netAmounts,proto3" json:"net_amounts" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3,customtype=cosmossdk.io/math.Int"`
	StartTime   int64               `protobuf:"varint,5,opt,name=start_time,json=startTime,proto3" json:"start_time"`
	EndTime     int64               `protobuf:"varint,6,opt,name=end_time,json=endTime,proto3" json:"end_time"`
	Status      int32               `protobuf:"varint,7,opt,name=status,proto3" json:"status"`
}

func (nc *NettingCycle) ProtoMessage()  {}
func (nc *NettingCycle) Reset()         { *nc = NettingCycle{} }
func (nc *NettingCycle) String() string {
	return fmt.Sprintf("NettingCycle{CycleID: %d, Status: %d}", nc.CycleID, nc.Status)
}

// BankPair represents a pair of banks involved in netting
type BankPair struct {
	BankA     string   `protobuf:"bytes,1,opt,name=bank_a,json=bankA,proto3" json:"bank_a"`
	BankB     string   `protobuf:"bytes,2,opt,name=bank_b,json=bankB,proto3" json:"bank_b"`
	AmountA   math.Int `protobuf:"bytes,3,opt,name=amount_a,json=amountA,proto3,customtype=cosmossdk.io/math.Int" json:"amount_a"`
	AmountB   math.Int `protobuf:"bytes,4,opt,name=amount_b,json=amountB,proto3,customtype=cosmossdk.io/math.Int" json:"amount_b"`
	NetAmount math.Int `protobuf:"bytes,5,opt,name=net_amount,json=netAmount,proto3,customtype=cosmossdk.io/math.Int" json:"net_amount"`
	NetDebtor string   `protobuf:"bytes,6,opt,name=net_debtor,json=netDebtor,proto3" json:"net_debtor"`
}

func (bp *BankPair) ProtoMessage()  {}
func (bp *BankPair) Reset()         { *bp = BankPair{} }
func (bp *BankPair) String() string { return fmt.Sprintf("BankPair{BankA: %s, BankB: %s}", bp.BankA, bp.BankB) }

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
	Validators   []Validator `protobuf:"bytes,1,rep,name=validators,proto3" json:"validators"`
	Threshold    int32       `protobuf:"varint,2,opt,name=threshold,proto3" json:"threshold"`
	UpdateHeight int64       `protobuf:"varint,3,opt,name=update_height,json=updateHeight,proto3" json:"update_height"`
	Version      uint64      `protobuf:"varint,4,opt,name=version,proto3" json:"version"`
}

func (vs *ValidatorSet) ProtoMessage()  {}
func (vs *ValidatorSet) Reset()         { *vs = ValidatorSet{} }
func (vs *ValidatorSet) String() string {
	return fmt.Sprintf("ValidatorSet{ValidatorCount: %d, Threshold: %d}", len(vs.Validators), vs.Threshold)
}

// Validator represents a validator in the set
type Validator struct {
	Address  string `protobuf:"bytes,1,opt,name=address,proto3" json:"address"`
	PubKey   []byte `protobuf:"bytes,2,opt,name=pub_key,json=pubKey,proto3" json:"pub_key"`
	Power    int64  `protobuf:"varint,3,opt,name=power,proto3" json:"power"`
	Active   bool   `protobuf:"varint,4,opt,name=active,proto3" json:"active"`
	JoinedAt int64  `protobuf:"varint,5,opt,name=joined_at,json=joinedAt,proto3" json:"joined_at"`
}

func (v *Validator) ProtoMessage()  {}
func (v *Validator) Reset()         { *v = Validator{} }
func (v *Validator) String() string { return fmt.Sprintf("Validator{Address: %s, Power: %d}", v.Address, v.Power) }

// MintCommand represents a command to mint tokens on a destination chain
type MintCommand struct {
	CommandID   string           `protobuf:"bytes,1,opt,name=command_id,json=commandId,proto3" json:"command_id"`
	TargetChain string           `protobuf:"bytes,2,opt,name=target_chain,json=targetChain,proto3" json:"target_chain"`
	Recipient   string           `protobuf:"bytes,3,opt,name=recipient,proto3" json:"recipient"`
	Amount      math.Int         `protobuf:"bytes,4,opt,name=amount,proto3,customtype=cosmossdk.io/math.Int" json:"amount"`
	Signatures  []ECDSASignature `protobuf:"bytes,5,rep,name=signatures,proto3" json:"signatures"`
	CreatedAt   int64            `protobuf:"varint,6,opt,name=created_at,json=createdAt,proto3" json:"created_at"`
	Status      int32            `protobuf:"varint,7,opt,name=status,proto3" json:"status"`
}

func (mc *MintCommand) ProtoMessage()  {}
func (mc *MintCommand) Reset()         { *mc = MintCommand{} }
func (mc *MintCommand) String() string {
	return fmt.Sprintf("MintCommand{CommandID: %s, Recipient: %s}", mc.CommandID, mc.Recipient)
}

// ECDSASignature represents an ECDSA signature
type ECDSASignature struct {
	Validator string `protobuf:"bytes,1,opt,name=validator,proto3" json:"validator"`
	R         []byte `protobuf:"bytes,2,opt,name=r,proto3" json:"r"`
	S         []byte `protobuf:"bytes,3,opt,name=s,proto3" json:"s"`
	V         uint32 `protobuf:"varint,4,opt,name=v,proto3" json:"v"`
	Timestamp int64  `protobuf:"varint,5,opt,name=timestamp,proto3" json:"timestamp"`
}

func (es *ECDSASignature) ProtoMessage()  {}
func (es *ECDSASignature) Reset()         { *es = ECDSASignature{} }
func (es *ECDSASignature) String() string { return fmt.Sprintf("ECDSASignature{Validator: %s}", es.Validator) }

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
	ID          uint64            `protobuf:"varint,1,opt,name=id,proto3" json:"id"`
	EventType   string            `protobuf:"bytes,2,opt,name=event_type,json=eventType,proto3" json:"event_type"`
	TxHash      string            `protobuf:"bytes,3,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash"`
	Details     map[string]string `protobuf:"bytes,4,rep,name=details,proto3" json:"details" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Timestamp   int64             `protobuf:"varint,5,opt,name=timestamp,proto3" json:"timestamp"`
	BlockHeight int64             `protobuf:"varint,6,opt,name=block_height,json=blockHeight,proto3" json:"block_height"`
}

func (al *AuditLog) ProtoMessage()  {}
func (al *AuditLog) Reset()         { *al = AuditLog{} }
func (al *AuditLog) String() string { return fmt.Sprintf("AuditLog{ID: %d, EventType: %s}", al.ID, al.EventType) }

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
