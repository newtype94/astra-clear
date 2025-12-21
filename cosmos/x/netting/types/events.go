package types

// Netting module event types
const (
	EventTypeCreditIssued      = "credit_issued"
	EventTypeCreditBurned      = "credit_burned"
	EventTypeCreditTransferred = "credit_transferred"
	EventTypeNettingTriggered  = "netting_triggered"
	EventTypeNettingCompleted  = "netting_completed"
	EventTypeNettingFailed     = "netting_failed"
	EventTypeNettingRollback   = "netting_rollback"
)

// Netting module event attribute keys
const (
	AttributeKeyDenom         = "denom"
	AttributeKeyAmount        = "amount"
	AttributeKeyIssuerBank    = "issuer_bank"
	AttributeKeyHolderBank    = "holder_bank"
	AttributeKeyFromBank      = "from_bank"
	AttributeKeyToBank        = "to_bank"
	AttributeKeyOriginTx      = "origin_tx"
	AttributeKeyCycleID       = "cycle_id"
	AttributeKeyBlockHeight   = "block_height"
	AttributeKeyPairCount     = "pair_count"
	AttributeKeyNetAmount     = "net_amount"
	AttributeKeyBankA         = "bank_a"
	AttributeKeyBankB         = "bank_b"
	AttributeKeyAmountA       = "amount_a"
	AttributeKeyAmountB       = "amount_b"
	AttributeKeyNetDebtor     = "net_debtor"
	AttributeKeyReason        = "reason"
)