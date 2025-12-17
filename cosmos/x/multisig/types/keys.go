package types

const (
	// ModuleName defines the module name
	ModuleName = "multisig"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_multisig"
)

// Store key prefixes
var (
	// ValidatorSetKeyPrefix is the prefix for validator set storage
	ValidatorSetKeyPrefix = []byte{0x01}
	
	// MintCommandKeyPrefix is the prefix for mint command storage
	MintCommandKeyPrefix = []byte{0x02}
	
	// SignatureKeyPrefix is the prefix for signature storage
	SignatureKeyPrefix = []byte{0x03}
	
	// ValidatorKeyPrefix is the prefix for individual validator storage
	ValidatorKeyPrefix = []byte{0x04}
	
	// CommandStatusKeyPrefix is the prefix for command status storage
	CommandStatusKeyPrefix = []byte{0x05}
)

// GetValidatorSetKey returns the store key for the current validator set
func GetValidatorSetKey() []byte {
	return ValidatorSetKeyPrefix
}

// GetMintCommandKey returns the store key for a mint command
func GetMintCommandKey(commandID string) []byte {
	return append(MintCommandKeyPrefix, []byte(commandID)...)
}

// GetSignatureKey returns the store key for a signature
func GetSignatureKey(commandID, validator string) []byte {
	key := append(SignatureKeyPrefix, []byte(commandID)...)
	key = append(key, []byte("/")...)
	return append(key, []byte(validator)...)
}

// GetValidatorKey returns the store key for a validator
func GetValidatorKey(address string) []byte {
	return append(ValidatorKeyPrefix, []byte(address)...)
}

// GetCommandStatusKey returns the store key for command status
func GetCommandStatusKey(commandID string) []byte {
	return append(CommandStatusKeyPrefix, []byte(commandID)...)
}