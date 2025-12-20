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

	// AuditLogKeyPrefix is the prefix for audit log storage
	AuditLogKeyPrefix = []byte{0x05}

	// AuditLogCounterKey is the key for audit log ID counter
	AuditLogCounterKey = []byte{0x06}

	// AuditLogByTimeKeyPrefix is the prefix for time-indexed audit logs
	AuditLogByTimeKeyPrefix = []byte{0x07}

	// AuditLogByTypeKeyPrefix is the prefix for type-indexed audit logs
	AuditLogByTypeKeyPrefix = []byte{0x08}
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

// GetAuditLogKey returns the store key for an audit log by ID
func GetAuditLogKey(id uint64) []byte {
	bz := make([]byte, 8)
	bz[0] = byte(id >> 56)
	bz[1] = byte(id >> 48)
	bz[2] = byte(id >> 40)
	bz[3] = byte(id >> 32)
	bz[4] = byte(id >> 24)
	bz[5] = byte(id >> 16)
	bz[6] = byte(id >> 8)
	bz[7] = byte(id)
	return append(AuditLogKeyPrefix, bz...)
}

// GetAuditLogByTimeKey returns the store key for time-indexed audit logs
// Format: prefix + timestamp (8 bytes) + id (8 bytes)
func GetAuditLogByTimeKey(timestamp int64, id uint64) []byte {
	key := make([]byte, 16)
	// Store timestamp as big-endian for proper ordering
	key[0] = byte(timestamp >> 56)
	key[1] = byte(timestamp >> 48)
	key[2] = byte(timestamp >> 40)
	key[3] = byte(timestamp >> 32)
	key[4] = byte(timestamp >> 24)
	key[5] = byte(timestamp >> 16)
	key[6] = byte(timestamp >> 8)
	key[7] = byte(timestamp)
	// Add ID for uniqueness
	key[8] = byte(id >> 56)
	key[9] = byte(id >> 48)
	key[10] = byte(id >> 40)
	key[11] = byte(id >> 32)
	key[12] = byte(id >> 24)
	key[13] = byte(id >> 16)
	key[14] = byte(id >> 8)
	key[15] = byte(id)
	return append(AuditLogByTimeKeyPrefix, key...)
}

// GetAuditLogByTypeKey returns the store key for type-indexed audit logs
// Format: prefix + eventType + "/" + id (8 bytes)
func GetAuditLogByTypeKey(eventType string, id uint64) []byte {
	key := append(AuditLogByTypeKeyPrefix, []byte(eventType)...)
	key = append(key, byte('/'))
	bz := make([]byte, 8)
	bz[0] = byte(id >> 56)
	bz[1] = byte(id >> 48)
	bz[2] = byte(id >> 40)
	bz[3] = byte(id >> 32)
	bz[4] = byte(id >> 24)
	bz[5] = byte(id >> 16)
	bz[6] = byte(id >> 8)
	bz[7] = byte(id)
	return append(key, bz...)
}

// GetAuditLogByTypePrefix returns the prefix for a specific event type
func GetAuditLogByTypePrefix(eventType string) []byte {
	key := append(AuditLogByTypeKeyPrefix, []byte(eventType)...)
	return append(key, byte('/'))
}

// GetAuditLogTimeRangePrefix returns prefix for time range queries
func GetAuditLogTimeRangePrefix(startTime int64) []byte {
	key := make([]byte, 8)
	key[0] = byte(startTime >> 56)
	key[1] = byte(startTime >> 48)
	key[2] = byte(startTime >> 40)
	key[3] = byte(startTime >> 32)
	key[4] = byte(startTime >> 24)
	key[5] = byte(startTime >> 16)
	key[6] = byte(startTime >> 8)
	key[7] = byte(startTime)
	return append(AuditLogByTimeKeyPrefix, key...)
}