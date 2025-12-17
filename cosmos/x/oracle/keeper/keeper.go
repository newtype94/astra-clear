package keeper

import (
	"fmt"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/interbank-netting/cosmos/types"
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

// SubmitVote submits a validator vote on a transfer event
func (k Keeper) SubmitVote(ctx sdk.Context, vote types.Vote) error {
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
		voteStatus = types.VoteStatus{
			TxHash:      vote.TxHash,
			Votes:       []types.Vote{vote},
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
func (k Keeper) GetVoteStatus(ctx sdk.Context, txHash string) (types.VoteStatus, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetVoteStatusKey(txHash)
	
	bz := store.Get(key)
	if bz == nil {
		return types.VoteStatus{}, false
	}

	var voteStatus types.VoteStatus
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
		creditToken := types.CreditToken{
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

	val, found := k.stakingKeeper.GetValidator(ctx, valAddr)
	if !found {
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

	val, found := k.stakingKeeper.GetValidator(ctx, valAddr)
	if !found {
		return nil, false
	}

	pubKey, err := val.ConsPubKey()
	if err != nil {
		return nil, false
	}

	return pubKey.Bytes(), true
}

// VerifySignature verifies a validator's signature
func (k Keeper) VerifySignature(ctx sdk.Context, validator string, data []byte, signature []byte) bool {
	pubKey, found := k.GetValidatorPubKey(ctx, validator)
	if !found {
		return false
	}

	// TODO: Implement actual signature verification using the public key
	// For now, we'll do a basic length check
	return len(signature) > 0 && len(pubKey) > 0
}

// Private helper methods

func (k Keeper) hasVoted(ctx sdk.Context, txHash, validator string) bool {
	store := ctx.KVStore(k.storeKey)
	key := types.GetVoteKey(txHash, validator)
	return store.Has(key)
}

func (k Keeper) setVote(ctx sdk.Context, vote types.Vote) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetVoteKey(vote.TxHash, vote.Validator)
	bz := k.cdc.MustMarshal(&vote)
	store.Set(key, bz)
}

func (k Keeper) setVoteStatus(ctx sdk.Context, voteStatus types.VoteStatus) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetVoteStatusKey(voteStatus.TxHash)
	bz := k.cdc.MustMarshal(&voteStatus)
	store.Set(key, bz)
}

func (k Keeper) setConfirmedTransfer(ctx sdk.Context, txHash string, eventData types.TransferEvent) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetConfirmedTransferKey(txHash)
	bz := k.cdc.MustMarshal(&eventData)
	store.Set(key, bz)
}

func (k Keeper) getConsensusThreshold(ctx sdk.Context) int {
	// Get all bonded validators
	validators := k.stakingKeeper.GetBondedValidatorsByPower(ctx)
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