package keeper

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"strconv"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/interbank-netting/cosmos/types"
	multisigtypes "github.com/interbank-netting/cosmos/x/multisig/types"
)

// Keeper of the multisig store
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	memKey     storetypes.StoreKey
	paramstore paramtypes.Subspace

	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper
}

// NewKeeper creates a new multisig Keeper instance
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

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", multisigtypes.ModuleName))
}

// GetValidatorSet retrieves the current validator set
func (k Keeper) GetValidatorSet(ctx sdk.Context) types.ValidatorSet {
	store := ctx.KVStore(k.storeKey)
	key := multisigtypes.GetValidatorSetKey()
	
	bz := store.Get(key)
	if bz == nil {
		// Return default validator set if none exists
		return k.getDefaultValidatorSet(ctx)
	}

	var validatorSet types.ValidatorSet
	k.cdc.MustUnmarshal(bz, &validatorSet)
	return validatorSet
}

// UpdateValidatorSet updates the validator set
func (k Keeper) UpdateValidatorSet(ctx sdk.Context, validators []types.Validator) error {
	if len(validators) == 0 {
		return multisigtypes.ErrValidatorSetEmpty
	}

	// Calculate 2/3 threshold
	threshold := (len(validators) * 2) / 3
	if (len(validators)*2)%3 != 0 {
		threshold++ // Round up for 2/3+ majority
	}
	if threshold < 1 {
		threshold = 1
	}

	// Get current validator set for version increment
	currentSet := k.GetValidatorSet(ctx)
	newVersion := currentSet.Version + 1

	// Create new validator set
	validatorSet := types.ValidatorSet{
		Validators:   validators,
		Threshold:    int32(threshold),
		UpdateHeight: ctx.BlockHeight(),
		Version:      newVersion,
	}

	// Store validator set
	k.setValidatorSet(ctx, validatorSet)

	// Store individual validators
	for _, validator := range validators {
		k.setValidator(ctx, validator)
	}

	// Emit validator set updated event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			multisigtypes.EventTypeValidatorSetUpdated,
			sdk.NewAttribute(multisigtypes.AttributeKeyValidatorCount, strconv.Itoa(len(validators))),
			sdk.NewAttribute(multisigtypes.AttributeKeyThreshold, strconv.Itoa(threshold)),
			sdk.NewAttribute(multisigtypes.AttributeKeyVersion, strconv.FormatUint(newVersion, 10)),
			sdk.NewAttribute(multisigtypes.AttributeKeyUpdateHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
		),
	)

	return nil
}

// AddValidator adds a new validator to the set
func (k Keeper) AddValidator(ctx sdk.Context, validator types.Validator) error {
	// Check if validator already exists
	if k.validatorExists(ctx, validator.Address) {
		return multisigtypes.ErrValidatorAlreadyExists
	}

	// Get current validator set
	validatorSet := k.GetValidatorSet(ctx)
	
	// Add new validator
	validatorSet.Validators = append(validatorSet.Validators, validator)
	
	// Recalculate threshold
	threshold := (len(validatorSet.Validators) * 2) / 3
	if (len(validatorSet.Validators)*2)%3 != 0 {
		threshold++
	}
	if threshold < 1 {
		threshold = 1
	}
	validatorSet.Threshold = int32(threshold)
	validatorSet.Version++
	validatorSet.UpdateHeight = ctx.BlockHeight()

	// Update validator set
	k.setValidatorSet(ctx, validatorSet)
	k.setValidator(ctx, validator)

	// Emit validator added event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			multisigtypes.EventTypeValidatorAdded,
			sdk.NewAttribute(multisigtypes.AttributeKeyValidatorAddress, validator.Address),
			sdk.NewAttribute(multisigtypes.AttributeKeyValidatorPower, strconv.FormatInt(validator.Power, 10)),
			sdk.NewAttribute(multisigtypes.AttributeKeyThreshold, strconv.Itoa(threshold)),
		),
	)

	return nil
}

// RemoveValidator removes a validator from the set
func (k Keeper) RemoveValidator(ctx sdk.Context, address string) error {
	// Get current validator set
	validatorSet := k.GetValidatorSet(ctx)
	
	// Find and remove validator
	found := false
	newValidators := make([]types.Validator, 0, len(validatorSet.Validators))
	for _, validator := range validatorSet.Validators {
		if validator.Address != address {
			newValidators = append(newValidators, validator)
		} else {
			found = true
		}
	}

	if !found {
		return multisigtypes.ErrValidatorNotFound
	}

	if len(newValidators) == 0 {
		return multisigtypes.ErrValidatorSetEmpty
	}

	// Update validator set
	validatorSet.Validators = newValidators
	
	// Recalculate threshold
	threshold := (len(newValidators) * 2) / 3
	if (len(newValidators)*2)%3 != 0 {
		threshold++
	}
	if threshold < 1 {
		threshold = 1
	}
	validatorSet.Threshold = int32(threshold)
	validatorSet.Version++
	validatorSet.UpdateHeight = ctx.BlockHeight()

	k.setValidatorSet(ctx, validatorSet)
	k.removeValidator(ctx, address)

	// Emit validator removed event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			multisigtypes.EventTypeValidatorRemoved,
			sdk.NewAttribute(multisigtypes.AttributeKeyValidatorAddress, address),
			sdk.NewAttribute(multisigtypes.AttributeKeyThreshold, strconv.Itoa(int(threshold))),
		),
	)

	return nil
}

// GenerateMintCommand generates a new mint command
func (k Keeper) GenerateMintCommand(ctx sdk.Context, targetChain, recipient string, amount math.Int) (types.MintCommand, error) {
	// Generate unique command ID
	commandID := k.generateCommandID(ctx, targetChain, recipient, amount)

	// Create mint command
	command := types.MintCommand{
		CommandID:   commandID,
		TargetChain: targetChain,
		Recipient:   recipient,
		Amount:      amount,
		Signatures:  []types.ECDSASignature{},
		CreatedAt:   ctx.BlockTime().Unix(),
		Status:      int32(types.CommandStatusPending),
	}

	// Store command
	k.setMintCommand(ctx, command)

	// Emit mint command generated event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			multisigtypes.EventTypeMintCommandGenerated,
			sdk.NewAttribute(multisigtypes.AttributeKeyCommandID, commandID),
			sdk.NewAttribute(multisigtypes.AttributeKeyTargetChain, targetChain),
			sdk.NewAttribute(multisigtypes.AttributeKeyRecipient, recipient),
			sdk.NewAttribute(multisigtypes.AttributeKeyAmount, amount.String()),
		),
	)

	return command, nil
}

// CollectSignatures collects signatures for a mint command
func (k Keeper) CollectSignatures(ctx sdk.Context, commandID string) error {
	// Get command
	command, found := k.GetCommand(ctx, commandID)
	if !found {
		return multisigtypes.ErrCommandNotFound
	}

	// Get validator set
	validatorSet := k.GetValidatorSet(ctx)

	// Check if we have enough signatures
	if int32(len(command.Signatures)) >= validatorSet.Threshold {
		// Mark command as signed
		command.Status = int32(types.CommandStatusSigned)
		k.setMintCommand(ctx, command)

		// Emit threshold reached event
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				multisigtypes.EventTypeThresholdReached,
				sdk.NewAttribute(multisigtypes.AttributeKeyCommandID, commandID),
				sdk.NewAttribute(multisigtypes.AttributeKeySignatureCount, strconv.Itoa(len(command.Signatures))),
				sdk.NewAttribute(multisigtypes.AttributeKeyThreshold, strconv.FormatInt(int64(validatorSet.Threshold), 10)),
			),
		)
	}

	return nil
}

// VerifyCommand verifies a mint command's signatures
func (k Keeper) VerifyCommand(ctx sdk.Context, command types.MintCommand) bool {
	validatorSet := k.GetValidatorSet(ctx)

	// Check if we have enough signatures
	if int32(len(command.Signatures)) < validatorSet.Threshold {
		return false
	}

	// Verify each signature
	validSignatures := int32(0)
	commandHash := k.hashCommand(command)

	for _, signature := range command.Signatures {
		if k.VerifyECDSASignature(ctx, commandHash, signature) {
			validSignatures++
		}
	}

	return validSignatures >= validatorSet.Threshold
}

// GetCommand retrieves a mint command by ID
func (k Keeper) GetCommand(ctx sdk.Context, commandID string) (types.MintCommand, bool) {
	store := ctx.KVStore(k.storeKey)
	key := multisigtypes.GetMintCommandKey(commandID)
	
	bz := store.Get(key)
	if bz == nil {
		return types.MintCommand{}, false
	}

	var command types.MintCommand
	k.cdc.MustUnmarshal(bz, &command)
	return command, true
}

// SignData signs data with a validator's key (mock implementation)
func (k Keeper) SignData(ctx sdk.Context, validator string, data []byte) (types.ECDSASignature, error) {
	// Get validator info
	_, found := k.getValidator(ctx, validator)
	if !found {
		return types.ECDSASignature{}, multisigtypes.ErrValidatorNotFound
	}

	// Mock ECDSA signature generation
	// In a real implementation, this would use the validator's private key
	r := make([]byte, 32)
	s := make([]byte, 32)
	rand.Read(r)
	rand.Read(s)

	signature := types.ECDSASignature{
		Validator: validator,
		R:         r,
		S:         s,
		V:         27, // Standard recovery ID
		Timestamp: ctx.BlockTime().Unix(),
	}

	return signature, nil
}

// VerifyECDSASignature verifies an ECDSA signature
func (k Keeper) VerifyECDSASignature(ctx sdk.Context, data []byte, signature types.ECDSASignature) bool {
	// Get validator info
	validator, found := k.getValidator(ctx, signature.Validator)
	if !found {
		return false
	}

	// Mock signature verification
	// In a real implementation, this would use ECDSA verification with the validator's public key
	return len(signature.R) == 32 && len(signature.S) == 32 && len(validator.PubKey) > 0
}

// AddSignatureToCommand adds a signature to a mint command
func (k Keeper) AddSignatureToCommand(ctx sdk.Context, commandID string, signature types.ECDSASignature) error {
	// Get command
	command, found := k.GetCommand(ctx, commandID)
	if !found {
		return multisigtypes.ErrCommandNotFound
	}

	// Check if validator already signed
	for _, sig := range command.Signatures {
		if sig.Validator == signature.Validator {
			return multisigtypes.ErrDuplicateSignature
		}
	}

	// Verify signature
	commandHash := k.hashCommand(command)
	if !k.VerifyECDSASignature(ctx, commandHash, signature) {
		return multisigtypes.ErrInvalidECDSASignature
	}

	// Add signature
	command.Signatures = append(command.Signatures, signature)
	k.setMintCommand(ctx, command)

	// Emit signature added event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			multisigtypes.EventTypeCommandSigned,
			sdk.NewAttribute(multisigtypes.AttributeKeyCommandID, commandID),
			sdk.NewAttribute(multisigtypes.AttributeKeyValidator, signature.Validator),
			sdk.NewAttribute(multisigtypes.AttributeKeySignatureCount, strconv.Itoa(len(command.Signatures))),
		),
	)

	// Check if threshold is reached
	return k.CollectSignatures(ctx, commandID)
}

// Private helper methods

func (k Keeper) getDefaultValidatorSet(ctx sdk.Context) types.ValidatorSet {
	// Get validators from staking module
	stakingValidators, err := k.stakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return types.ValidatorSet{Threshold: 1}
	}

	validators := make([]types.Validator, 0, len(stakingValidators))
	for _, stakingVal := range stakingValidators {
		pubKey, err := stakingVal.ConsPubKey()
		if err != nil {
			continue
		}

		validator := types.Validator{
			Address:  stakingVal.GetOperator(),
			PubKey:   pubKey.Bytes(),
			Power:    stakingVal.GetTokens().Int64(),
			Active:   stakingVal.IsBonded(),
			JoinedAt: ctx.BlockTime().Unix(),
		}
		validators = append(validators, validator)
	}

	threshold := (len(validators) * 2) / 3
	if (len(validators)*2)%3 != 0 {
		threshold++
	}
	if threshold < 1 {
		threshold = 1
	}

	return types.ValidatorSet{
		Validators:   validators,
		Threshold:    int32(threshold),
		UpdateHeight: ctx.BlockHeight(),
		Version:      1,
	}
}

func (k Keeper) setValidatorSet(ctx sdk.Context, validatorSet types.ValidatorSet) {
	store := ctx.KVStore(k.storeKey)
	key := multisigtypes.GetValidatorSetKey()
	bz := k.cdc.MustMarshal(&validatorSet)
	store.Set(key, bz)
}

func (k Keeper) setValidator(ctx sdk.Context, validator types.Validator) {
	store := ctx.KVStore(k.storeKey)
	key := multisigtypes.GetValidatorKey(validator.Address)
	bz := k.cdc.MustMarshal(&validator)
	store.Set(key, bz)
}

func (k Keeper) getValidator(ctx sdk.Context, address string) (types.Validator, bool) {
	store := ctx.KVStore(k.storeKey)
	key := multisigtypes.GetValidatorKey(address)
	
	bz := store.Get(key)
	if bz == nil {
		return types.Validator{}, false
	}

	var validator types.Validator
	k.cdc.MustUnmarshal(bz, &validator)
	return validator, true
}

func (k Keeper) removeValidator(ctx sdk.Context, address string) {
	store := ctx.KVStore(k.storeKey)
	key := multisigtypes.GetValidatorKey(address)
	store.Delete(key)
}

func (k Keeper) validatorExists(ctx sdk.Context, address string) bool {
	store := ctx.KVStore(k.storeKey)
	key := multisigtypes.GetValidatorKey(address)
	return store.Has(key)
}

func (k Keeper) setMintCommand(ctx sdk.Context, command types.MintCommand) {
	store := ctx.KVStore(k.storeKey)
	key := multisigtypes.GetMintCommandKey(command.CommandID)
	bz := k.cdc.MustMarshal(&command)
	store.Set(key, bz)
}

func (k Keeper) generateCommandID(ctx sdk.Context, targetChain, recipient string, amount math.Int) string {
	// Generate deterministic command ID based on block height, target chain, recipient, and amount
	data := fmt.Sprintf("%d-%s-%s-%s", ctx.BlockHeight(), targetChain, recipient, amount.String())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("cmd-%x", hash[:8]) // Use first 8 bytes of hash
}

func (k Keeper) hashCommand(command types.MintCommand) []byte {
	// Create hash of command for signing
	data := fmt.Sprintf("%s-%s-%s-%s", command.CommandID, command.TargetChain, command.Recipient, command.Amount.String())
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

// GetAllPendingCommands returns all commands awaiting signatures
func (k Keeper) GetAllPendingCommands(ctx sdk.Context) []types.MintCommand {
	return k.getCommandsByStatus(ctx, int32(types.CommandStatusPending))
}

// GetSignedCommands returns all commands that have collected enough signatures
func (k Keeper) GetSignedCommands(ctx sdk.Context) []types.MintCommand {
	return k.getCommandsByStatus(ctx, int32(types.CommandStatusSigned))
}

// GetAllCommands returns all mint commands in the store
func (k Keeper) GetAllCommands(ctx sdk.Context) []types.MintCommand {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, multisigtypes.MintCommandKeyPrefix)
	defer iterator.Close()

	commands := make([]types.MintCommand, 0)
	for ; iterator.Valid(); iterator.Next() {
		var command types.MintCommand
		k.cdc.MustUnmarshal(iterator.Value(), &command)
		commands = append(commands, command)
	}
	return commands
}

// getCommandsByStatus returns commands filtered by status
func (k Keeper) getCommandsByStatus(ctx sdk.Context, status int32) []types.MintCommand {
	allCommands := k.GetAllCommands(ctx)
	filtered := make([]types.MintCommand, 0)
	for _, cmd := range allCommands {
		if cmd.Status == status {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}

// ProcessPendingCommands processes all pending commands and collects signatures
// This is called in EndBlock to automatically collect signatures from validators
// Requirement 5.2: Collect ECDSA signatures from active validators
func (k Keeper) ProcessPendingCommands(ctx sdk.Context) error {
	pendingCommands := k.GetAllPendingCommands(ctx)
	validatorSet := k.GetValidatorSet(ctx)

	for _, command := range pendingCommands {
		// Each active validator signs the pending command
		for _, validator := range validatorSet.Validators {
			if !validator.Active {
				continue
			}

			// Check if validator already signed
			alreadySigned := false
			for _, sig := range command.Signatures {
				if sig.Validator == validator.Address {
					alreadySigned = true
					break
				}
			}

			if alreadySigned {
				continue
			}

			// Sign the command
			commandHash := k.hashCommand(command)
			signature, err := k.SignData(ctx, validator.Address, commandHash)
			if err != nil {
				// Log error but continue with other validators
				k.Logger(ctx).Error("failed to sign command", "command_id", command.CommandID, "validator", validator.Address, "error", err)
				continue
			}

			// Add signature to command
			if err := k.AddSignatureToCommand(ctx, command.CommandID, signature); err != nil {
				k.Logger(ctx).Error("failed to add signature", "command_id", command.CommandID, "validator", validator.Address, "error", err)
				continue
			}
		}
	}

	return nil
}

// MarkCommandExecuted marks a command as executed after Relayer confirms on-chain execution
func (k Keeper) MarkCommandExecuted(ctx sdk.Context, commandID string) error {
	command, found := k.GetCommand(ctx, commandID)
	if !found {
		return multisigtypes.ErrCommandNotFound
	}

	if command.Status != int32(types.CommandStatusSigned) {
		return multisigtypes.ErrInvalidCommandStatus
	}

	command.Status = int32(types.CommandStatusExecuted)
	k.setMintCommand(ctx, command)

	// Emit command executed event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			multisigtypes.EventTypeCommandExecuted,
			sdk.NewAttribute(multisigtypes.AttributeKeyCommandID, commandID),
		),
	)

	return nil
}