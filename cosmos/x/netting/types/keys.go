package types

import (
	"encoding/binary"
)

const (
	// ModuleName defines the module name
	ModuleName = "netting"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_netting"
)

// Store key prefixes
var (
	// CreditTokenKeyPrefix is the prefix for credit token storage
	CreditTokenKeyPrefix = []byte{0x01}
	
	// CreditBalanceKeyPrefix is the prefix for credit balance storage
	CreditBalanceKeyPrefix = []byte{0x02}
	
	// NettingCycleKeyPrefix is the prefix for netting cycle storage
	NettingCycleKeyPrefix = []byte{0x03}
	
	// DebtPositionKeyPrefix is the prefix for debt position storage
	DebtPositionKeyPrefix = []byte{0x04}
	
	// LastNettingBlockKeyPrefix is the prefix for last netting block height
	LastNettingBlockKeyPrefix = []byte{0x05}
)

// GetCreditTokenKey returns the store key for a credit token
func GetCreditTokenKey(denom string) []byte {
	return append(CreditTokenKeyPrefix, []byte(denom)...)
}

// GetCreditBalanceKey returns the store key for a bank's credit balance
func GetCreditBalanceKey(bank, denom string) []byte {
	key := append(CreditBalanceKeyPrefix, []byte(bank)...)
	key = append(key, []byte("/")...)
	return append(key, []byte(denom)...)
}

// GetNettingCycleKey returns the store key for a netting cycle
func GetNettingCycleKey(cycleID uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, cycleID)
	return append(NettingCycleKeyPrefix, bz...)
}

// GetDebtPositionKey returns the store key for debt position between two banks
func GetDebtPositionKey(bankA, bankB string) []byte {
	// Ensure consistent ordering
	if bankA > bankB {
		bankA, bankB = bankB, bankA
	}
	key := append(DebtPositionKeyPrefix, []byte(bankA)...)
	key = append(key, []byte("/")...)
	return append(key, []byte(bankB)...)
}

// GetLastNettingBlockKey returns the store key for last netting block height
func GetLastNettingBlockKey() []byte {
	return LastNettingBlockKeyPrefix
}

