package keeper

import (
	"crypto/sha256"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/crypto"

	commontypes "github.com/interbank-netting/cosmos/types"
	"github.com/interbank-netting/cosmos/x/oracle/types"
)

// Keeper of the oracle store
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	memKey     storetypes.StoreKey
	paramstore paramtypes.Subspace

	bankKeeper     types.BankKeeper
	stakingKeeper  types.StakingKeeper
	nettingKeeper  types.NettingKeeper
	multisigKeeper types.MultisigKeeper
}

// NewKeeper creates a new oracle Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
) *Keeper {
	return &Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		memKey:        memKey,
		paramstore:    ps,
		bankKeeper:    bankKeeper,
		stakingKeeper: stakingKeeper,
	}
}

// SetNettingKeeper sets the netting keeper (to avoid circular dependency)
func (k *Keeper) SetNettingKeeper(nettingKeeper types.NettingKeeper) {
	k.nettingKeeper = nettingKeeper
}

// SetMultisigKeeper sets the multisig keeper (to avoid circular dependency)
func (k *Keeper) SetMultisigKeeper(multisigKeeper types.MultisigKeeper) {
	k.multisigKeeper = multisigKeeper
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetStoreKey returns the store key
func (k Keeper) GetStoreKey() storetypes.StoreKey {
	return k.storeKey
}

// SubmitVote submits a validator vote on a transfer event
func (k Keeper) SubmitVote(ctx sdk.Context, vote commontypes.Vote) error {
	// Validate that the validator is active
	if !k.IsActiveValidator(ctx, vote.Validator) {
		return types.ErrValidatorNotActive
	}

	// Verify the signature
	if !k.VerifySignature(ctx, vote.Validator, []byte(vote.TxHash), vote.Signature) {
		return types.ErrInvalidSignature
	}

	// Check for duplicate vote
	if k.hasVoted(ctx, vote.TxHash, vote.Validator) {
		return types.ErrDuplicateVote
	}

	// Store the vote
	k.setVote(ctx, vote)

	// Update vote status
	voteStatus, found := k.GetVoteStatus(ctx, vote.TxHash)
	if !found {
		// Create new vote status
		voteStatus = commontypes.VoteStatus{
			TxHash:      vote.TxHash,
			Votes:       []commontypes.Vote{vote},
			Confirmed:   false,
			Threshold:   k.getConsensusThreshold(ctx),
			VoteCount:   1,
			CreatedAt:   ctx.BlockTime().Unix(),
			ConfirmedAt: 0,
		}
	} else {
		// Add vote to existing status
		voteStatus.Votes = append(voteStatus.Votes, vote)
		voteStatus.VoteCount++
	}

	k.setVoteStatus(ctx, voteStatus)

	// Emit vote submitted event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeVoteSubmitted,
			sdk.NewAttribute(types.AttributeKeyTxHash, vote.TxHash),
			sdk.NewAttribute(types.AttributeKeyValidator, vote.Validator),
			sdk.NewAttribute(types.AttributeKeyVoteCount, fmt.Sprintf("%d", voteStatus.VoteCount)),
			sdk.NewAttribute(types.AttributeKeyThreshold, fmt.Sprintf("%d", voteStatus.Threshold)),
		),
	)

	// Check if consensus is reached
	if voteStatus.VoteCount >= voteStatus.Threshold {
		return k.ConfirmTransfer(ctx, vote.TxHash)
	}

	return nil
}

// GetVoteStatus retrieves the vote status for a transaction hash
func (k Keeper) GetVoteStatus(ctx sdk.Context, txHash string) (commontypes.VoteStatus, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetVoteStatusKey(txHash)
	
	bz := store.Get(key)
	if bz == nil {
		return commontypes.VoteStatus{}, false
	}

	var voteStatus commontypes.VoteStatus
	k.cdc.MustUnmarshal(bz, &voteStatus)
	return voteStatus, true
}

// CheckConsensus checks if consensus has been reached for a transfer
func (k Keeper) CheckConsensus(ctx sdk.Context, txHash string) (bool, error) {
	voteStatus, found := k.GetVoteStatus(ctx, txHash)
	if !found {
		return false, types.ErrTransferNotFound
	}

	return voteStatus.VoteCount >= voteStatus.Threshold, nil
}

// ConfirmTransfer confirms a transfer after consensus is reached
func (k Keeper) ConfirmTransfer(ctx sdk.Context, txHash string) error {
	voteStatus, found := k.GetVoteStatus(ctx, txHash)
	if !found {
		return types.ErrTransferNotFound
	}

	if voteStatus.Confirmed {
		return types.ErrTransferAlreadyConfirmed
	}

	if voteStatus.VoteCount < voteStatus.Threshold {
		return types.ErrInsufficientVotes
	}

	// Mark as confirmed
	voteStatus.Confirmed = true
	voteStatus.ConfirmedAt = ctx.BlockTime().Unix()
	k.setVoteStatus(ctx, voteStatus)

	// Get the transfer event data from the first vote (all votes should have the same event data)
	if len(voteStatus.Votes) == 0 {
		return fmt.Errorf("no votes found for confirmed transfer")
	}

	eventData := voteStatus.Votes[0].EventData

	// Store confirmed transfer
	k.setConfirmedTransfer(ctx, txHash, eventData)

	// Log transfer confirmation (Requirement 7.1)
	if err := k.LogTransferConfirmed(ctx, txHash, eventData); err != nil {
		k.Logger(ctx).Error("failed to log transfer confirmation", "error", err)
		// Don't fail the transfer for logging errors
	}

	// Trigger credit token issuance through netting keeper
	if k.nettingKeeper != nil {
		creditToken := commontypes.CreditToken{
			Denom:      fmt.Sprintf("cred-%s", eventData.SourceChain),
			IssuerBank: eventData.SourceChain,
			HolderBank: eventData.DestChain,
			Amount:     eventData.Amount,
			OriginTx:   txHash,
			IssuedAt:   ctx.BlockTime().Unix(),
		}

		if err := k.nettingKeeper.IssueCreditToken(ctx, creditToken); err != nil {
			return fmt.Errorf("failed to issue credit token: %w", err)
		}
	}

	// Generate mint command through multisig keeper (Requirement 5.1)
	// This creates a command for minting tokens on the destination chain
	if k.multisigKeeper != nil {
		_, err := k.multisigKeeper.GenerateMintCommand(
			ctx,
			eventData.DestChain,  // Target chain where tokens will be minted
			eventData.Recipient,  // Recipient address on the destination chain
			eventData.Amount,     // Amount to mint
		)
		if err != nil {
			return fmt.Errorf("failed to generate mint command: %w", err)
		}
	}

	// Emit consensus reached event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeConsensusReached,
			sdk.NewAttribute(types.AttributeKeyTxHash, txHash),
			sdk.NewAttribute(types.AttributeKeyVoteCount, fmt.Sprintf("%d", voteStatus.VoteCount)),
			sdk.NewAttribute(types.AttributeKeyThreshold, fmt.Sprintf("%d", voteStatus.Threshold)),
		),
	)

	// Emit transfer confirmed event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeTransferConfirmed,
			sdk.NewAttribute(types.AttributeKeyTxHash, txHash),
			sdk.NewAttribute(types.AttributeKeySender, eventData.Sender),
			sdk.NewAttribute(types.AttributeKeyRecipient, eventData.Recipient),
			sdk.NewAttribute(types.AttributeKeyAmount, eventData.Amount.String()),
			sdk.NewAttribute(types.AttributeKeySourceChain, eventData.SourceChain),
			sdk.NewAttribute(types.AttributeKeyDestChain, eventData.DestChain),
		),
	)

	return nil
}

// IsActiveValidator checks if a validator is active
func (k Keeper) IsActiveValidator(ctx sdk.Context, validator string) bool {
	valAddr, err := sdk.ValAddressFromBech32(validator)
	if err != nil {
		return false
	}

	val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return false
	}

	return val.IsBonded()
}

// GetValidatorPubKey retrieves a validator's public key
func (k Keeper) GetValidatorPubKey(ctx sdk.Context, validator string) ([]byte, bool) {
	valAddr, err := sdk.ValAddressFromBech32(validator)
	if err != nil {
		return nil, false
	}

	val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return nil, false
	}

	pubKey, err := val.ConsPubKey()
	if err != nil {
		return nil, false
	}

	return pubKey.Bytes(), true
}

// VerifySignature verifies a validator's signature using ECDSA
func (k Keeper) VerifySignature(ctx sdk.Context, validator string, data []byte, signature []byte) bool {
	pubKey, found := k.GetValidatorPubKey(ctx, validator)
	if !found {
		k.Logger(ctx).Error("validator public key not found", "validator", validator)
		return false
	}

	// Basic validation
	if len(signature) == 0 || len(data) == 0 {
		k.Logger(ctx).Error("empty signature or data", "validator", validator)
		return false
	}

	// For ECDSA signatures, we expect 65 bytes (r=32, s=32, v=1)
	if len(signature) != 65 {
		k.Logger(ctx).Error("invalid signature length", "validator", validator, "length", len(signature))
		return false
	}

	// Hash the data using SHA256
	hash := sha256.Sum256(data)

	// Extract r, s, v from signature
	r := signature[:32]
	s := signature[32:64]
	v := signature[64]

	// Recover public key from signature
	recoveredPubKey, err := crypto.SigToPub(hash[:], signature)
	if err != nil {
		k.Logger(ctx).Error("failed to recover public key from signature", "validator", validator, "error", err)
		return false
	}

	// Convert recovered public key to bytes for comparison
	recoveredPubKeyBytes := crypto.FromECDSAPub(recoveredPubKey)

	// For Cosmos validators, we need to handle the public key format conversion
	// The stored pubKey might be in Cosmos secp256k1 format
	var expectedPubKeyBytes []byte
	
	// Try to parse as Cosmos secp256k1 public key first
	if len(pubKey) == 33 { // Compressed secp256k1 public key
		// Decompress the compressed public key to uncompressed format
		ecdsaPubKey, err := crypto.DecompressPubkey(pubKey)
		if err != nil {
			k.Logger(ctx).Error("failed to decompress cosmos public key", "validator", validator, "error", err)
			return false
		}
		expectedPubKeyBytes = crypto.FromECDSAPub(ecdsaPubKey)
	} else if len(pubKey) == 65 { // Uncompressed ECDSA public key
		expectedPubKeyBytes = pubKey
	} else {
		k.Logger(ctx).Error("unsupported public key format", "validator", validator, "length", len(pubKey))
		return false
	}

	// Compare recovered public key with stored public key
	if len(recoveredPubKeyBytes) != len(expectedPubKeyBytes) {
		k.Logger(ctx).Error("public key length mismatch", "validator", validator)
		return false
	}

	for i := 0; i < len(recoveredPubKeyBytes); i++ {
		if recoveredPubKeyBytes[i] != expectedPubKeyBytes[i] {
			k.Logger(ctx).Error("public key mismatch", "validator", validator)
			return false
		}
	}

	// Additional validation: verify signature components are valid
	if !k.isValidSignatureComponents(r, s, v) {
		k.Logger(ctx).Error("invalid signature components", "validator", validator)
		return false
	}

	k.Logger(ctx).Debug("signature verification successful", "validator", validator)
	return true
}

// isValidSignatureComponents validates ECDSA signature components
func (k Keeper) isValidSignatureComponents(r, s []byte, v byte) bool {
	// Check that r and s are not zero
	rIsZero := true
	sIsZero := true
	
	for _, b := range r {
		if b != 0 {
			rIsZero = false
			break
		}
	}
	
	for _, b := range s {
		if b != 0 {
			sIsZero = false
			break
		}
	}
	
	if rIsZero || sIsZero {
		return false
	}
	
	// Check v is valid (should be 27 or 28 for Ethereum-style, or 0/1 for some implementations)
	if v != 0 && v != 1 && v != 27 && v != 28 {
		return false
	}
	
	return true
}

// Private helper methods

func (k Keeper) hasVoted(ctx sdk.Context, txHash, validator string) bool {
	store := ctx.KVStore(k.storeKey)
	key := types.GetVoteKey(txHash, validator)
	return store.Has(key)
}

func (k Keeper) setVote(ctx sdk.Context, vote commontypes.Vote) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetVoteKey(vote.TxHash, vote.Validator)
	bz := k.cdc.MustMarshal(&vote)
	store.Set(key, bz)
}

func (k Keeper) setVoteStatus(ctx sdk.Context, voteStatus commontypes.VoteStatus) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetVoteStatusKey(voteStatus.TxHash)
	bz := k.cdc.MustMarshal(&voteStatus)
	store.Set(key, bz)
}

func (k Keeper) setConfirmedTransfer(ctx sdk.Context, txHash string, eventData commontypes.TransferEvent) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetConfirmedTransferKey(txHash)
	bz := k.cdc.MustMarshal(&eventData)
	store.Set(key, bz)
}

func (k Keeper) getConsensusThreshold(ctx sdk.Context) int32 {
	// Get all bonded validators
	validators, err := k.stakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return 1 // Default minimum threshold
	}
	totalValidators := len(validators)

	// Calculate 2/3 threshold
	threshold := (totalValidators * 2) / 3
	if (totalValidators*2)%3 != 0 {
		threshold++ // Round up for 2/3+ majority
	}

	// Minimum threshold of 1
	if threshold < 1 {
		threshold = 1
	}

	return int32(threshold)
}

// RejectTransfer rejects a transfer due to insufficient votes or timeout
// Requirement 3.4: WHEN 충분하지 않은 투표가 수신되면 THEN 시스템은 이체를 거부하고 현재 상태를 유지해야 합니다
func (k Keeper) RejectTransfer(ctx sdk.Context, txHash string, reason string) error {
	voteStatus, found := k.GetVoteStatus(ctx, txHash)
	if !found {
		return types.ErrTransferNotFound
	}

	if voteStatus.Confirmed {
		return types.ErrTransferAlreadyConfirmed
	}

	// Mark as rejected (we don't modify the vote status, just emit an event)
	// The transfer remains in pending state and won't be processed

	// Emit transfer rejected event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeTransferRejected,
			sdk.NewAttribute(types.AttributeKeyTxHash, txHash),
			sdk.NewAttribute(types.AttributeKeyVoteCount, fmt.Sprintf("%d", voteStatus.VoteCount)),
			sdk.NewAttribute(types.AttributeKeyThreshold, fmt.Sprintf("%d", voteStatus.Threshold)),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	k.Logger(ctx).Info("transfer rejected",
		"tx_hash", txHash,
		"vote_count", voteStatus.VoteCount,
		"threshold", voteStatus.Threshold,
		"reason", reason,
	)

	return nil
}

// CheckConsensusTimeout checks if a transfer has timed out without reaching consensus
func (k Keeper) CheckConsensusTimeout(ctx sdk.Context, txHash string, timeoutBlocks int64) (bool, error) {
	voteStatus, found := k.GetVoteStatus(ctx, txHash)
	if !found {
		return false, types.ErrTransferNotFound
	}

	if voteStatus.Confirmed {
		return false, nil // Already confirmed, no timeout
	}

	// Calculate time elapsed since vote creation
	currentTime := ctx.BlockTime().Unix()
	elapsedTime := currentTime - voteStatus.CreatedAt

	// Check if timeout period has elapsed (assuming ~6 seconds per block)
	timeoutSeconds := timeoutBlocks * 6
	if elapsedTime >= timeoutSeconds {
		return true, nil
	}

	return false, nil
}

// GetConfirmedTransfer retrieves a confirmed transfer by txHash
func (k Keeper) GetConfirmedTransfer(ctx sdk.Context, txHash string) (commontypes.TransferEvent, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetConfirmedTransferKey(txHash)

	bz := store.Get(key)
	if bz == nil {
		return commontypes.TransferEvent{}, false
	}

	var transferEvent commontypes.TransferEvent
	k.cdc.MustUnmarshal(bz, &transferEvent)
	return transferEvent, true
}

// GetAllVoteStatuses retrieves all vote statuses (for queries and auditing)
func (k Keeper) GetAllVoteStatuses(ctx sdk.Context) []commontypes.VoteStatus {
	store := ctx.KVStore(k.storeKey)

	statuses := make([]commontypes.VoteStatus, 0)
	iterator := storetypes.KVStorePrefixIterator(store, types.VoteStatusKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var voteStatus commontypes.VoteStatus
		k.cdc.MustUnmarshal(iterator.Value(), &voteStatus)
		statuses = append(statuses, voteStatus)
	}

	return statuses
}

// =============================================================================
// Error Handling and Recovery (Task 12.2)
// =============================================================================

// GetDynamicThreshold calculates threshold based on active validators only
// This handles validator offline scenario by excluding inactive validators
func (k Keeper) GetDynamicThreshold(ctx sdk.Context) (threshold int32, activeCount int) {
	validators, err := k.stakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return 1, 0
	}

	// Count only active (bonded) validators
	activeCount = 0
	for _, val := range validators {
		if val.IsBonded() && !val.IsJailed() {
			activeCount++
		}
	}

	if activeCount == 0 {
		return 1, 0
	}

	// Calculate 2/3 threshold
	threshold = int32((activeCount * 2) / 3)
	if (activeCount*2)%3 != 0 {
		threshold++
	}

	if threshold < 1 {
		threshold = 1
	}

	return threshold, activeCount
}

// ProcessPendingTransfersWithTimeout processes all pending transfers and rejects timed out ones
// Requirement 12.2: 검증자 오프라인 시 동적 임계값 조정
func (k Keeper) ProcessPendingTransfersWithTimeout(ctx sdk.Context, timeoutBlocks int64) (processed, rejected int) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.VoteStatusKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var voteStatus commontypes.VoteStatus
		k.cdc.MustUnmarshal(iterator.Value(), &voteStatus)

		// Skip already confirmed transfers
		if voteStatus.Confirmed {
			continue
		}

		processed++

		// Check for timeout
		isTimeout, err := k.CheckConsensusTimeout(ctx, voteStatus.TxHash, timeoutBlocks)
		if err != nil {
			k.Logger(ctx).Error("failed to check consensus timeout",
				"tx_hash", voteStatus.TxHash,
				"error", err,
			)
			continue
		}

		if isTimeout {
			// Reject the transfer due to timeout
			if err := k.RejectTransfer(ctx, voteStatus.TxHash, "consensus timeout"); err != nil {
				k.Logger(ctx).Error("failed to reject timed out transfer",
					"tx_hash", voteStatus.TxHash,
					"error", err,
				)
			} else {
				rejected++
			}
		}
	}

	return processed, rejected
}

// ValidateVoteSignatureWithFallback validates signature with fallback for individual signature errors
// Requirement 12.2: 서명 오류 시 개별 서명 제외 처리
func (k Keeper) ValidateVoteSignatureWithFallback(ctx sdk.Context, vote commontypes.Vote) (valid bool, excludeReason string) {
	// First try normal signature verification
	if k.VerifySignature(ctx, vote.Validator, []byte(vote.TxHash), vote.Signature) {
		return true, ""
	}

	// Log the signature error
	k.Logger(ctx).Info("signature verification failed, excluding vote",
		"validator", vote.Validator,
		"tx_hash", vote.TxHash,
	)

	// Return invalid with reason
	return false, "invalid signature"
}

// GetPendingTransferCount returns the count of pending (unconfirmed) transfers
func (k Keeper) GetPendingTransferCount(ctx sdk.Context) int {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.VoteStatusKeyPrefix)
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		var voteStatus commontypes.VoteStatus
		k.cdc.MustUnmarshal(iterator.Value(), &voteStatus)
		if !voteStatus.Confirmed {
			count++
		}
	}

	return count
}

// RecoverFromConsensusFailure attempts to recover from consensus failure
// by recalculating thresholds and retrying confirmation
func (k Keeper) RecoverFromConsensusFailure(ctx sdk.Context, txHash string) error {
	voteStatus, found := k.GetVoteStatus(ctx, txHash)
	if !found {
		return types.ErrTransferNotFound
	}

	if voteStatus.Confirmed {
		return nil // Already confirmed, nothing to do
	}

	// Get dynamic threshold based on current active validators
	threshold, activeCount := k.GetDynamicThreshold(ctx)

	k.Logger(ctx).Info("attempting recovery with dynamic threshold",
		"tx_hash", txHash,
		"current_votes", voteStatus.VoteCount,
		"original_threshold", voteStatus.Threshold,
		"dynamic_threshold", threshold,
		"active_validators", activeCount,
	)

	// If we now have enough votes with the dynamic threshold, confirm
	if voteStatus.VoteCount >= threshold {
		// Update threshold and attempt confirmation
		voteStatus.Threshold = threshold
		k.setVoteStatus(ctx, voteStatus)

		return k.ConfirmTransfer(ctx, txHash)
	}

	return types.ErrInsufficientVotes
}

// =============================================================================
// Audit Logging System (Requirement 7.1 - 7.5)
// =============================================================================

// SaveAuditLog saves an audit log entry with automatic ID assignment
// Requirement 7.1: 거래 로깅 시스템
func (k Keeper) SaveAuditLog(ctx sdk.Context, log commontypes.AuditLog) (uint64, error) {
	store := ctx.KVStore(k.storeKey)

	// Get and increment the counter
	id := k.getNextAuditLogID(ctx)
	log.ID = id
	log.BlockHeight = ctx.BlockHeight()

	if log.Timestamp == 0 {
		log.Timestamp = ctx.BlockTime().Unix()
	}

	// Marshal the log
	bz := k.cdc.MustMarshal(&log)

	// Store by ID (primary index)
	store.Set(types.GetAuditLogKey(id), bz)

	// Store by time (secondary index for time range queries)
	store.Set(types.GetAuditLogByTimeKey(log.Timestamp, id), bz)

	// Store by event type (secondary index for type filtering)
	store.Set(types.GetAuditLogByTypeKey(log.EventType, id), bz)

	k.Logger(ctx).Debug("audit log saved",
		"id", id,
		"event_type", log.EventType,
		"tx_hash", log.TxHash,
	)

	return id, nil
}

// getNextAuditLogID gets and increments the audit log counter
func (k Keeper) getNextAuditLogID(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AuditLogCounterKey)

	var counter uint64 = 1
	if bz != nil {
		counter = uint64(bz[0])<<56 | uint64(bz[1])<<48 | uint64(bz[2])<<40 | uint64(bz[3])<<32 |
			uint64(bz[4])<<24 | uint64(bz[5])<<16 | uint64(bz[6])<<8 | uint64(bz[7])
		counter++
	}

	// Store incremented counter
	newBz := make([]byte, 8)
	newBz[0] = byte(counter >> 56)
	newBz[1] = byte(counter >> 48)
	newBz[2] = byte(counter >> 40)
	newBz[3] = byte(counter >> 32)
	newBz[4] = byte(counter >> 24)
	newBz[5] = byte(counter >> 16)
	newBz[6] = byte(counter >> 8)
	newBz[7] = byte(counter)
	store.Set(types.AuditLogCounterKey, newBz)

	return counter
}

// GetAuditLog retrieves an audit log by ID
func (k Keeper) GetAuditLog(ctx sdk.Context, id uint64) (commontypes.AuditLog, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetAuditLogKey(id))
	if bz == nil {
		return commontypes.AuditLog{}, false
	}

	var log commontypes.AuditLog
	k.cdc.MustUnmarshal(bz, &log)
	return log, true
}

// GetAuditLogsByTimeRange retrieves audit logs within a time range
// Requirement 7.5: 감사 쿼리 API
func (k Keeper) GetAuditLogsByTimeRange(ctx sdk.Context, startTime, endTime int64) []commontypes.AuditLog {
	store := ctx.KVStore(k.storeKey)
	logs := make([]commontypes.AuditLog, 0)

	// Create iterator starting from startTime
	startKey := types.GetAuditLogTimeRangePrefix(startTime)
	endKey := types.GetAuditLogTimeRangePrefix(endTime + 1) // +1 to include endTime

	iterator := store.Iterator(startKey, endKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var log commontypes.AuditLog
		k.cdc.MustUnmarshal(iterator.Value(), &log)
		logs = append(logs, log)
	}

	return logs
}

// GetAuditLogsByEventType retrieves audit logs by event type
// Requirement 7.5: 감사 쿼리 API
func (k Keeper) GetAuditLogsByEventType(ctx sdk.Context, eventType string) []commontypes.AuditLog {
	store := ctx.KVStore(k.storeKey)
	logs := make([]commontypes.AuditLog, 0)

	prefix := types.GetAuditLogByTypePrefix(eventType)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var log commontypes.AuditLog
		k.cdc.MustUnmarshal(iterator.Value(), &log)
		logs = append(logs, log)
	}

	return logs
}

// GetAuditLogsByTxHash retrieves audit logs by transaction hash
// Requirement 7.3: 추적성 이벤트
func (k Keeper) GetAuditLogsByTxHash(ctx sdk.Context, txHash string) []commontypes.AuditLog {
	store := ctx.KVStore(k.storeKey)
	logs := make([]commontypes.AuditLog, 0)

	// Iterate all audit logs and filter by txHash
	iterator := storetypes.KVStorePrefixIterator(store, types.AuditLogKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var log commontypes.AuditLog
		k.cdc.MustUnmarshal(iterator.Value(), &log)
		if log.TxHash == txHash {
			logs = append(logs, log)
		}
	}

	return logs
}

// GetAllAuditLogs retrieves all audit logs
func (k Keeper) GetAllAuditLogs(ctx sdk.Context) []commontypes.AuditLog {
	store := ctx.KVStore(k.storeKey)
	logs := make([]commontypes.AuditLog, 0)

	iterator := storetypes.KVStorePrefixIterator(store, types.AuditLogKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var log commontypes.AuditLog
		k.cdc.MustUnmarshal(iterator.Value(), &log)
		logs = append(logs, log)
	}

	return logs
}

// LogTransferConfirmed logs a transfer confirmation event
// Requirement 7.1: 거래 로깅
func (k Keeper) LogTransferConfirmed(ctx sdk.Context, txHash string, eventData commontypes.TransferEvent) error {
	log := commontypes.AuditLog{
		EventType: commontypes.EventTypeTransferConfirmed,
		TxHash:    txHash,
		Timestamp: ctx.BlockTime().Unix(),
		Details: map[string]string{
			"sender":       eventData.Sender,
			"recipient":    eventData.Recipient,
			"amount":       eventData.Amount.String(),
			"source_chain": eventData.SourceChain,
			"dest_chain":   eventData.DestChain,
			"nonce":        fmt.Sprintf("%d", eventData.Nonce),
		},
	}

	_, err := k.SaveAuditLog(ctx, log)
	return err
}

// LogCreditIssued logs a credit token issuance event
// Requirement 7.1: 거래 로깅
func (k Keeper) LogCreditIssued(ctx sdk.Context, credit commontypes.CreditToken) error {
	log := commontypes.AuditLog{
		EventType: commontypes.EventTypeCreditIssued,
		TxHash:    credit.OriginTx,
		Timestamp: ctx.BlockTime().Unix(),
		Details: map[string]string{
			"denom":       credit.Denom,
			"issuer_bank": credit.IssuerBank,
			"holder_bank": credit.HolderBank,
			"amount":      credit.Amount.String(),
		},
	}

	_, err := k.SaveAuditLog(ctx, log)
	return err
}

// GetAuditLogCount returns the total count of audit logs
func (k Keeper) GetAuditLogCount(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AuditLogCounterKey)

	if bz == nil {
		return 0
	}

	return uint64(bz[0])<<56 | uint64(bz[1])<<48 | uint64(bz[2])<<40 | uint64(bz[3])<<32 |
		uint64(bz[4])<<24 | uint64(bz[5])<<16 | uint64(bz[6])<<8 | uint64(bz[7])
}