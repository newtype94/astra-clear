package types

// Multisig module event types
const (
	EventTypeMintCommandGenerated = "mint_command_generated"
	EventTypeCommandSigned        = "command_signed"
	EventTypeThresholdReached     = "threshold_reached"
	EventTypeValidatorSetUpdated  = "validator_set_updated"
	EventTypeValidatorAdded       = "validator_added"
	EventTypeValidatorRemoved     = "validator_removed"
	EventTypeSignatureVerified    = "signature_verified"
	EventTypeSignatureRejected    = "signature_rejected"
)

// Multisig module event attribute keys
const (
	AttributeKeyCommandID        = "command_id"
	AttributeKeyTargetChain      = "target_chain"
	AttributeKeyRecipient        = "recipient"
	AttributeKeyAmount           = "amount"
	AttributeKeyValidator        = "validator"
	AttributeKeySignatureCount   = "signature_count"
	AttributeKeyThreshold        = "threshold"
	AttributeKeyValidatorCount   = "validator_count"
	AttributeKeyValidatorAddress = "validator_address"
	AttributeKeyValidatorPubKey  = "validator_pub_key"
	AttributeKeyValidatorPower   = "validator_power"
	AttributeKeyVersion          = "version"
	AttributeKeyUpdateHeight     = "update_height"
	AttributeKeyReason           = "reason"
)