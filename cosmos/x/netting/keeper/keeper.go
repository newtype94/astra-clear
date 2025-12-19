package keeper

import (
	"fmt"
	"strconv"

	"github.com/cometbft/cometbft/libs/log"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/interbank-netting/cosmos/types"
	nettingtypes "github.com/interbank-netting/cosmos/x/netting/types"
)

// Keeper of the netting store
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	memKey     storetypes.StoreKey
	paramstore paramtypes.Subspace

	bankKeeper    types.BankKeeper
	accountKeeper types.AccountKeeper
}

// NewKeeper creates a new netting Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
) *Keeper {
	return &Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		memKey:        memKey,
		paramstore:    ps,
		bankKeeper:    bankKeeper,
		accountKeeper: accountKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", nettingtypes.ModuleName))
}

// IssueCreditToken issues a new credit token
func (k Keeper) IssueCreditToken(ctx sdk.Context, token types.CreditToken) error {
	// Validate credit token
	if err := k.validateCreditToken(token); err != nil {
		return err
	}

	// Check if credit token already exists
	if k.creditTokenExists(ctx, token.Denom) {
		return nettingtypes.ErrDuplicateCreditToken
	}

	// Store credit token
	k.setCreditToken(ctx, token)

	// Update credit balance for holder bank
	k.addCreditBalance(ctx, token.HolderBank, token.Denom, token.Amount)

	// Emit credit issued event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			nettingtypes.EventTypeCreditIssued,
			sdk.NewAttribute(nettingtypes.AttributeKeyDenom, token.Denom),
			sdk.NewAttribute(nettingtypes.AttributeKeyAmount, token.Amount.String()),
			sdk.NewAttribute(nettingtypes.AttributeKeyIssuerBank, token.IssuerBank),
			sdk.NewAttribute(nettingtypes.AttributeKeyHolderBank, token.HolderBank),
			sdk.NewAttribute(nettingtypes.AttributeKeyOriginTx, token.OriginTx),
		),
	)

	return nil
}

// BurnCreditToken burns credit tokens
func (k Keeper) BurnCreditToken(ctx sdk.Context, denom string, amount math.Int) error {
	// Validate amount
	if amount.IsNil() || amount.LTE(math.ZeroInt()) {
		return nettingtypes.ErrInvalidAmount
	}

	// Get credit token info
	token, found := k.getCreditToken(ctx, denom)
	if !found {
		return nettingtypes.ErrCreditTokenNotFound
	}

	// Check if holder bank has sufficient balance
	balance := k.GetCreditBalance(ctx, token.HolderBank, denom)
	if balance.LT(amount) {
		return nettingtypes.ErrInsufficientBalance
	}

	// Subtract from credit balance
	k.subtractCreditBalance(ctx, token.HolderBank, denom, amount)

	// Emit credit burned event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			nettingtypes.EventTypeCreditBurned,
			sdk.NewAttribute(nettingtypes.AttributeKeyDenom, denom),
			sdk.NewAttribute(nettingtypes.AttributeKeyAmount, amount.String()),
			sdk.NewAttribute(nettingtypes.AttributeKeyHolderBank, token.HolderBank),
		),
	)

	return nil
}

// TransferCreditToken transfers credit tokens between banks
func (k Keeper) TransferCreditToken(ctx sdk.Context, from, to, denom string, amount math.Int) error {
	// Validate amount
	if amount.IsNil() || amount.LTE(math.ZeroInt()) {
		return nettingtypes.ErrInvalidAmount
	}

	// Check if from bank has sufficient balance
	balance := k.GetCreditBalance(ctx, from, denom)
	if balance.LT(amount) {
		return nettingtypes.ErrInsufficientBalance
	}

	// Transfer credit balance
	k.subtractCreditBalance(ctx, from, denom, amount)
	k.addCreditBalance(ctx, to, denom, amount)

	// Emit credit transferred event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			nettingtypes.EventTypeCreditTransferred,
			sdk.NewAttribute(nettingtypes.AttributeKeyDenom, denom),
			sdk.NewAttribute(nettingtypes.AttributeKeyAmount, amount.String()),
			sdk.NewAttribute(nettingtypes.AttributeKeyFromBank, from),
			sdk.NewAttribute(nettingtypes.AttributeKeyToBank, to),
		),
	)

	return nil
}

// GetCreditBalance returns the credit balance for a bank and denom
func (k Keeper) GetCreditBalance(ctx sdk.Context, bank, denom string) math.Int {
	store := ctx.KVStore(k.storeKey)
	key := nettingtypes.GetCreditBalanceKey(bank, denom)
	
	bz := store.Get(key)
	if bz == nil {
		return math.ZeroInt()
	}

	var balance math.Int
	k.cdc.MustUnmarshal(bz, &balance)
	return balance
}

// GetAllCreditBalances returns all credit balances for a bank
func (k Keeper) GetAllCreditBalances(ctx sdk.Context, bank string) map[string]math.Int {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, append(nettingtypes.CreditBalanceKeyPrefix, []byte(bank)...))
	defer iterator.Close()

	balances := make(map[string]math.Int)
	for ; iterator.Valid(); iterator.Next() {
		var balance math.Int
		k.cdc.MustUnmarshal(iterator.Value(), &balance)
		
		// Extract denom from key
		key := string(iterator.Key())
		parts := sdk.SplitPath(key)
		if len(parts) >= 2 {
			denom := parts[len(parts)-1]
			balances[denom] = balance
		}
	}

	return balances
}

// GetDebtPosition returns the debt position between two banks
func (k Keeper) GetDebtPosition(ctx sdk.Context, bankA, bankB string) (math.Int, math.Int) {
	// Get credit tokens that bankA holds from bankB (bankB owes bankA)
	credAFromB := k.GetCreditBalance(ctx, bankA, "cred-"+bankB)
	
	// Get credit tokens that bankB holds from bankA (bankA owes bankB)
	credBFromA := k.GetCreditBalance(ctx, bankB, "cred-"+bankA)
	
	return credAFromB, credBFromA
}

// TriggerNetting triggers the netting process
func (k Keeper) TriggerNetting(ctx sdk.Context) error {
	// Check if enough blocks have passed since last netting
	lastNettingBlock := k.getLastNettingBlock(ctx)
	currentBlock := ctx.BlockHeight()
	
	if currentBlock-lastNettingBlock < 10 {
		return nettingtypes.ErrNettingNotRequired
	}

	// Calculate netting pairs
	pairs, err := k.CalculateNetting(ctx)
	if err != nil {
		return err
	}

	if len(pairs) == 0 {
		return nettingtypes.ErrNettingNotRequired
	}

	// Execute netting
	if err := k.ExecuteNetting(ctx, pairs); err != nil {
		return err
	}

	// Update last netting block
	k.setLastNettingBlock(ctx, currentBlock)

	// Emit netting triggered event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			nettingtypes.EventTypeNettingTriggered,
			sdk.NewAttribute(nettingtypes.AttributeKeyBlockHeight, strconv.FormatInt(currentBlock, 10)),
			sdk.NewAttribute(nettingtypes.AttributeKeyPairCount, strconv.Itoa(len(pairs))),
		),
	)

	return nil
}

// CalculateNetting calculates netting pairs
func (k Keeper) CalculateNetting(ctx sdk.Context) ([]types.BankPair, error) {
	// Get all banks with credit balances
	banks := k.getAllBanksWithCredits(ctx)
	var pairs []types.BankPair

	// Calculate netting for each bank pair
	for i := 0; i < len(banks); i++ {
		for j := i + 1; j < len(banks); j++ {
			bankA := banks[i]
			bankB := banks[j]

			// Get mutual credit positions
			credAFromB, credBFromA := k.GetDebtPosition(ctx, bankA, bankB)

			// Only create pair if both banks have credits from each other
			if credAFromB.GT(math.ZeroInt()) && credBFromA.GT(math.ZeroInt()) {
				var netAmount math.Int
				var netDebtor string

				if credAFromB.GT(credBFromA) {
					netAmount = credAFromB.Sub(credBFromA)
					netDebtor = bankB
				} else {
					netAmount = credBFromA.Sub(credAFromB)
					netDebtor = bankA
				}

				pair := types.BankPair{
					BankA:     bankA,
					BankB:     bankB,
					AmountA:   credBFromA, // Amount A owes to B
					AmountB:   credAFromB, // Amount B owes to A
					NetAmount: netAmount,
					NetDebtor: netDebtor,
				}

				pairs = append(pairs, pair)
			}
		}
	}

	return pairs, nil
}

// ExecuteNetting executes the netting process
func (k Keeper) ExecuteNetting(ctx sdk.Context, pairs []types.BankPair) error {
	cycleID := uint64(ctx.BlockHeight())

	// Create netting cycle
	cycle := types.NettingCycle{
		CycleID:     cycleID,
		BlockHeight: ctx.BlockHeight(),
		Pairs:       pairs,
		NetAmounts:  make(map[string]sdk.Int),
		StartTime:   ctx.BlockTime().Unix(),
		Status:      types.NettingStatusInProgress,
	}

	// Execute netting for each pair
	for _, pair := range pairs {
		// Calculate minimum amount to net
		minAmount := pair.AmountA
		if pair.AmountB.LT(minAmount) {
			minAmount = pair.AmountB
		}

		// Burn credit tokens from both banks
		if err := k.BurnCreditToken(ctx, "cred-"+pair.BankA, minAmount); err != nil {
			return fmt.Errorf("failed to burn credit from %s: %w", pair.BankA, err)
		}

		if err := k.BurnCreditToken(ctx, "cred-"+pair.BankB, minAmount); err != nil {
			return fmt.Errorf("failed to burn credit from %s: %w", pair.BankB, err)
		}

		// Update net amounts
		cycle.NetAmounts[pair.BankA] = cycle.NetAmounts[pair.BankA].Add(minAmount)
		cycle.NetAmounts[pair.BankB] = cycle.NetAmounts[pair.BankB].Add(minAmount)
	}

	// Mark cycle as completed
	cycle.EndTime = ctx.BlockTime().Unix()
	cycle.Status = types.NettingStatusCompleted

	// Store netting cycle
	k.setNettingCycle(ctx, cycle)

	// Emit netting completed event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			nettingtypes.EventTypeNettingCompleted,
			sdk.NewAttribute(nettingtypes.AttributeKeyCycleID, strconv.FormatUint(cycleID, 10)),
			sdk.NewAttribute(nettingtypes.AttributeKeyBlockHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
			sdk.NewAttribute(nettingtypes.AttributeKeyPairCount, strconv.Itoa(len(pairs))),
		),
	)

	return nil
}

// GetNettingCycle retrieves a netting cycle by ID
func (k Keeper) GetNettingCycle(ctx sdk.Context, cycleID uint64) (types.NettingCycle, bool) {
	store := ctx.KVStore(k.storeKey)
	key := nettingtypes.GetNettingCycleKey(cycleID)
	
	bz := store.Get(key)
	if bz == nil {
		return types.NettingCycle{}, false
	}

	var cycle types.NettingCycle
	k.cdc.MustUnmarshal(bz, &cycle)
	return cycle, true
}

// Private helper methods

func (k Keeper) validateCreditToken(token types.CreditToken) error {
	if token.Denom == "" {
		return nettingtypes.ErrInvalidCreditToken
	}
	if token.IssuerBank == "" {
		return nettingtypes.ErrInvalidBankID
	}
	if token.HolderBank == "" {
		return nettingtypes.ErrInvalidBankID
	}
	if token.Amount.IsNil() || token.Amount.LTE(math.ZeroInt()) {
		return nettingtypes.ErrInvalidAmount
	}
	return nil
}

func (k Keeper) creditTokenExists(ctx sdk.Context, denom string) bool {
	store := ctx.KVStore(k.storeKey)
	key := nettingtypes.GetCreditTokenKey(denom)
	return store.Has(key)
}

func (k Keeper) setCreditToken(ctx sdk.Context, token types.CreditToken) {
	store := ctx.KVStore(k.storeKey)
	key := nettingtypes.GetCreditTokenKey(token.Denom)
	bz := k.cdc.MustMarshal(&token)
	store.Set(key, bz)
}

func (k Keeper) getCreditToken(ctx sdk.Context, denom string) (types.CreditToken, bool) {
	store := ctx.KVStore(k.storeKey)
	key := nettingtypes.GetCreditTokenKey(denom)
	
	bz := store.Get(key)
	if bz == nil {
		return types.CreditToken{}, false
	}

	var token types.CreditToken
	k.cdc.MustUnmarshal(bz, &token)
	return token, true
}

func (k Keeper) addCreditBalance(ctx sdk.Context, bank, denom string, amount math.Int) {
	currentBalance := k.GetCreditBalance(ctx, bank, denom)
	newBalance := currentBalance.Add(amount)
	k.setCreditBalance(ctx, bank, denom, newBalance)
}

func (k Keeper) subtractCreditBalance(ctx sdk.Context, bank, denom string, amount math.Int) {
	currentBalance := k.GetCreditBalance(ctx, bank, denom)
	newBalance := currentBalance.Sub(amount)
	if newBalance.LT(math.ZeroInt()) {
		newBalance = math.ZeroInt()
	}
	k.setCreditBalance(ctx, bank, denom, newBalance)
}

func (k Keeper) setCreditBalance(ctx sdk.Context, bank, denom string, balance math.Int) {
	store := ctx.KVStore(k.storeKey)
	key := nettingtypes.GetCreditBalanceKey(bank, denom)
	bz := k.cdc.MustMarshal(&balance)
	store.Set(key, bz)
}

func (k Keeper) getAllBanksWithCredits(ctx sdk.Context) []string {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, nettingtypes.CreditBalanceKeyPrefix)
	defer iterator.Close()

	bankSet := make(map[string]bool)
	for ; iterator.Valid(); iterator.Next() {
		// Extract bank from key
		key := string(iterator.Key())
		parts := sdk.SplitPath(key)
		if len(parts) >= 2 {
			bank := parts[1]
			bankSet[bank] = true
		}
	}

	banks := make([]string, 0, len(bankSet))
	for bank := range bankSet {
		banks = append(banks, bank)
	}

	return banks
}

func (k Keeper) getLastNettingBlock(ctx sdk.Context) int64 {
	store := ctx.KVStore(k.storeKey)
	key := nettingtypes.GetLastNettingBlockKey()
	
	bz := store.Get(key)
	if bz == nil {
		return 0
	}

	var blockHeight int64
	k.cdc.MustUnmarshal(bz, &blockHeight)
	return blockHeight
}

func (k Keeper) setLastNettingBlock(ctx sdk.Context, blockHeight int64) {
	store := ctx.KVStore(k.storeKey)
	key := nettingtypes.GetLastNettingBlockKey()
	bz := k.cdc.MustMarshal(&blockHeight)
	store.Set(key, bz)
}

func (k Keeper) setNettingCycle(ctx sdk.Context, cycle types.NettingCycle) {
	store := ctx.KVStore(k.storeKey)
	key := nettingtypes.GetNettingCycleKey(cycle.CycleID)
	bz := k.cdc.MustMarshal(&cycle)
	store.Set(key, bz)
}