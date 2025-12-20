package keeper

import (
	"crypto/sha256"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
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

	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper
	nettingKeeper types.NettingKeeper
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
		cosmosKey := &secp256k1.PubKey{Key: pubKey}
		// Convert to uncompressed format for comparison
		ecdsaPubKey, err := crypto.UnmarshalPubkey(cosmosKey.Bytes())
		if err != nil {
			k.Logger(ctx).Error("failed to unmarshal cosmos public key", "validator", validator, "error", err)
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

func (k Keeper) getConsensusThreshold(ctx sdk.Context) int {
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

	return threshold
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

	var statuses []commontypes.VoteStatus
	iterator := storetypes.KVStorePrefixIterator(store, types.VoteStatusKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var voteStatus commontypes.VoteStatus
		k.cdc.MustUnmarshal(iterator.Value(), &voteStatus)
		statuses = append(statuses, voteStatus)
	}

	return statuses
}