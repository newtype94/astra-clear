package types

const (
	// ModuleName defines the module name
	ModuleName = "oracle"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_oracle"
)

// Store key prefixes
var (
	// VoteStatusKeyPrefix is the prefix for vote status storage
	VoteStatusKeyPrefix = []byte{0x01}
	
	// VoteKeyPrefix is the prefix for individual votes storage
	VoteKeyPrefix = []byte{0x02}
	
	// ValidatorKeyPrefix is the prefix for validator storage
	ValidatorKeyPrefix = []byte{0x03}
	
	// ConfirmedTransferKeyPrefix is the prefix for confirmed transfers
	ConfirmedTransferKeyPrefix = []byte{0x04}
)

// GetVoteStatusKey returns the store key for a vote status
func GetVoteStatusKey(txHash string) []byte {
	return append(VoteStatusKeyPrefix, []byte(txHash)...)
}

// GetVoteKey returns the store key for a specific vote
func GetVoteKey(txHash, validator string) []byte {
	key := append(VoteKeyPrefix, []byte(txHash)...)
	key = append(key, []byte("/")...)
	return append(key, []byte(validator)...)
}

// GetValidatorKey returns the store key for a validator
func GetValidatorKey(validator string) []byte {
	return append(ValidatorKeyPrefix, []byte(validator)...)
}

// GetConfirmedTransferKey returns the store key for a confirmed transfer
func GetConfirmedTransferKey(txHash string) []byte {
	return append(ConfirmedTransferKeyPrefix, []byte(txHash)...)
}