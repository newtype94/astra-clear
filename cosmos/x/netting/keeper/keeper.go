package keeper

import (
	"fmt"
	"strconv"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
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
	oracleKeeper  nettingtypes.OracleKeeper
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

// SetOracleKeeper sets the oracle keeper (to avoid circular dependency)
func (k *Keeper) SetOracleKeeper(oracleKeeper nettingtypes.OracleKeeper) {
	k.oracleKeeper = oracleKeeper
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

	// Log credit issuance (Requirement 7.1)
	if k.oracleKeeper != nil {
		if err := k.oracleKeeper.LogCreditIssued(ctx, token); err != nil {
			k.Logger(ctx).Error("failed to log credit issuance", "error", err)
			// Don't fail for logging errors
		}
	}

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
	if err := balance.Unmarshal(bz); err != nil {
		return math.ZeroInt()
	}
	return balance
}

// GetAllCreditBalances returns all credit balances for a bank
func (k Keeper) GetAllCreditBalances(ctx sdk.Context, bank string) map[string]math.Int {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, append(nettingtypes.CreditBalanceKeyPrefix, []byte(bank)...))
	defer iterator.Close()

	balances := make(map[string]math.Int)
	for ; iterator.Valid(); iterator.Next() {
		var balance math.Int
		if err := balance.Unmarshal(iterator.Value()); err != nil {
			continue
		}

		// Extract denom from key - key format is prefix + bank + denom
		key := iterator.Key()
		// Find the last separator to extract denom
		keyStr := string(key)
		if idx := lastIndexByte(keyStr, '/'); idx != -1 {
			denom := keyStr[idx+1:]
			balances[denom] = balance
		}
	}

	return balances
}

// lastIndexByte returns the index of the last instance of c in s, or -1 if c is not present in s.
func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
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
		NetAmounts:  make(map[string]math.Int),
		StartTime:   ctx.BlockTime().Unix(),
		Status:      int32(types.NettingStatusInProgress),
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

		// Update net amounts (initialize to zero if not present)
		if _, ok := cycle.NetAmounts[pair.BankA]; !ok {
			cycle.NetAmounts[pair.BankA] = math.ZeroInt()
		}
		if _, ok := cycle.NetAmounts[pair.BankB]; !ok {
			cycle.NetAmounts[pair.BankB] = math.ZeroInt()
		}
		cycle.NetAmounts[pair.BankA] = cycle.NetAmounts[pair.BankA].Add(minAmount)
		cycle.NetAmounts[pair.BankB] = cycle.NetAmounts[pair.BankB].Add(minAmount)
	}

	// Mark cycle as completed
	cycle.EndTime = ctx.BlockTime().Unix()
	cycle.Status = int32(types.NettingStatusCompleted)

	// Store netting cycle
	k.setNettingCycle(ctx, cycle)

	// Log netting completion (Requirement 7.2)
	if k.oracleKeeper != nil {
		// Calculate total netted amount
		totalNetted := math.ZeroInt()
		for _, pair := range pairs {
			minAmount := pair.AmountA
			if pair.AmountB.LT(minAmount) {
				minAmount = pair.AmountB
			}
			totalNetted = totalNetted.Add(minAmount)
		}

		auditLog := types.AuditLog{
			EventType: types.EventTypeNettingCompleted,
			Timestamp: ctx.BlockTime().Unix(),
			Details: map[string]string{
				"cycle_id":      strconv.FormatUint(cycleID, 10),
				"pair_count":    strconv.Itoa(len(pairs)),
				"total_netted":  totalNetted.String(),
				"start_time":    strconv.FormatInt(cycle.StartTime, 10),
				"end_time":      strconv.FormatInt(cycle.EndTime, 10),
			},
		}
		if _, err := k.oracleKeeper.SaveAuditLog(ctx, auditLog); err != nil {
			k.Logger(ctx).Error("failed to log netting completion", "error", err)
			// Don't fail for logging errors
		}
	}

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
	bz, _ := balance.Marshal()
	store.Set(key, bz)
}

func (k Keeper) getAllBanksWithCredits(ctx sdk.Context) []string {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, nettingtypes.CreditBalanceKeyPrefix)
	defer iterator.Close()

	prefixLen := len(nettingtypes.CreditBalanceKeyPrefix)
	bankSet := make(map[string]bool)
	for ; iterator.Valid(); iterator.Next() {
		// Key format: prefix + bank + "/" + denom
		key := iterator.Key()
		if len(key) <= prefixLen {
			continue
		}
		// Strip the prefix, then find the first '/' to extract bank name
		withoutPrefix := string(key[prefixLen:])
		if idx := indexByte(withoutPrefix, '/'); idx != -1 {
			bank := withoutPrefix[:idx]
			if bank != "" {
				bankSet[bank] = true
			}
		}
	}

	banks := make([]string, 0, len(bankSet))
	for bank := range bankSet {
		banks = append(banks, bank)
	}

	return banks
}

// indexByte returns the index of the first instance of c in s, or -1 if c is not present in s.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func (k Keeper) getLastNettingBlock(ctx sdk.Context) int64 {
	store := ctx.KVStore(k.storeKey)
	key := nettingtypes.GetLastNettingBlockKey()

	bz := store.Get(key)
	if bz == nil {
		return 0
	}

	// Parse int64 from bytes
	if len(bz) != 8 {
		return 0
	}
	blockHeight := int64(bz[0]) | int64(bz[1])<<8 | int64(bz[2])<<16 | int64(bz[3])<<24 |
		int64(bz[4])<<32 | int64(bz[5])<<40 | int64(bz[6])<<48 | int64(bz[7])<<56
	return blockHeight
}

func (k Keeper) setLastNettingBlock(ctx sdk.Context, blockHeight int64) {
	store := ctx.KVStore(k.storeKey)
	key := nettingtypes.GetLastNettingBlockKey()
	// Encode int64 to bytes
	bz := make([]byte, 8)
	bz[0] = byte(blockHeight)
	bz[1] = byte(blockHeight >> 8)
	bz[2] = byte(blockHeight >> 16)
	bz[3] = byte(blockHeight >> 24)
	bz[4] = byte(blockHeight >> 32)
	bz[5] = byte(blockHeight >> 40)
	bz[6] = byte(blockHeight >> 48)
	bz[7] = byte(blockHeight >> 56)
	store.Set(key, bz)
}

func (k Keeper) setNettingCycle(ctx sdk.Context, cycle types.NettingCycle) {
	store := ctx.KVStore(k.storeKey)
	key := nettingtypes.GetNettingCycleKey(cycle.CycleID)
	bz := k.cdc.MustMarshal(&cycle)
	store.Set(key, bz)
}

// =============================================================================
// Error Handling and Recovery (Task 12.3)
// =============================================================================

// NettingSnapshot stores balances before netting for potential rollback
type NettingSnapshot struct {
	CycleID  uint64
	Balances map[string]map[string]math.Int // bank -> denom -> amount
}

// CreateNettingSnapshot creates a snapshot of current credit balances for rollback
func (k Keeper) CreateNettingSnapshot(ctx sdk.Context, pairs []types.BankPair) NettingSnapshot {
	snapshot := NettingSnapshot{
		CycleID:  uint64(ctx.BlockHeight()),
		Balances: make(map[string]map[string]math.Int),
	}

	// Collect all affected banks
	affectedBanks := make(map[string]bool)
	for _, pair := range pairs {
		affectedBanks[pair.BankA] = true
		affectedBanks[pair.BankB] = true
	}

	// Store current balances
	for bank := range affectedBanks {
		balances := k.GetAllCreditBalances(ctx, bank)
		if len(balances) > 0 {
			snapshot.Balances[bank] = balances
		}
	}

	return snapshot
}

// RollbackNetting restores balances from snapshot after a failed netting
func (k Keeper) RollbackNetting(ctx sdk.Context, snapshot NettingSnapshot) error {
	k.Logger(ctx).Info("rolling back netting cycle",
		"cycle_id", snapshot.CycleID,
		"affected_banks", len(snapshot.Balances),
	)

	for bank, balances := range snapshot.Balances {
		for denom, amount := range balances {
			k.setCreditBalance(ctx, bank, denom, amount)
		}
	}

	// Emit rollback event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			nettingtypes.EventTypeNettingRollback,
			sdk.NewAttribute(nettingtypes.AttributeKeyCycleID, strconv.FormatUint(snapshot.CycleID, 10)),
			sdk.NewAttribute(nettingtypes.AttributeKeyBlockHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
		),
	)

	return nil
}

// ExecuteNettingWithRollback executes netting with automatic rollback on failure
// Requirement 12.3: 부분 소각 실패 시 전체 롤백
func (k Keeper) ExecuteNettingWithRollback(ctx sdk.Context, pairs []types.BankPair) error {
	if len(pairs) == 0 {
		return nettingtypes.ErrNettingNotRequired
	}

	// Create snapshot before netting
	snapshot := k.CreateNettingSnapshot(ctx, pairs)

	// Attempt netting
	err := k.ExecuteNetting(ctx, pairs)
	if err != nil {
		// Rollback on failure
		k.Logger(ctx).Error("netting failed, initiating rollback",
			"cycle_id", snapshot.CycleID,
			"error", err,
		)

		rollbackErr := k.RollbackNetting(ctx, snapshot)
		if rollbackErr != nil {
			k.Logger(ctx).Error("rollback failed",
				"cycle_id", snapshot.CycleID,
				"error", rollbackErr,
			)
			return fmt.Errorf("netting failed and rollback failed: %w (rollback: %v)", err, rollbackErr)
		}

		return fmt.Errorf("netting failed, rolled back: %w", err)
	}

	return nil
}

// ValidateNettingPairs validates pairs before executing netting
// Requirement 12.3: 계산 오류 시 상태 복원
func (k Keeper) ValidateNettingPairs(ctx sdk.Context, pairs []types.BankPair) error {
	for i, pair := range pairs {
		// Validate bank IDs
		if pair.BankA == "" || pair.BankB == "" {
			return fmt.Errorf("pair %d: invalid bank ID", i)
		}

		// Validate amounts are positive
		if pair.AmountA.IsNil() || pair.AmountA.IsNegative() {
			return fmt.Errorf("pair %d: invalid AmountA", i)
		}
		if pair.AmountB.IsNil() || pair.AmountB.IsNegative() {
			return fmt.Errorf("pair %d: invalid AmountB", i)
		}

		// Validate sufficient balances exist
		balanceA := k.GetCreditBalance(ctx, pair.BankA, "cred-"+pair.BankB)
		balanceB := k.GetCreditBalance(ctx, pair.BankB, "cred-"+pair.BankA)

		minAmount := pair.AmountA
		if pair.AmountB.LT(minAmount) {
			minAmount = pair.AmountB
		}

		if balanceA.LT(minAmount) || balanceB.LT(minAmount) {
			return fmt.Errorf("pair %d: insufficient balance for netting", i)
		}
	}

	return nil
}

// TriggerNettingWithErrorHandling triggers netting with comprehensive error handling
func (k Keeper) TriggerNettingWithErrorHandling(ctx sdk.Context) error {
	// Check cooldown
	lastNettingBlock := k.getLastNettingBlock(ctx)
	currentBlock := ctx.BlockHeight()

	if currentBlock-lastNettingBlock < 10 {
		return nettingtypes.ErrNettingNotRequired
	}

	// Calculate pairs
	pairs, err := k.CalculateNetting(ctx)
	if err != nil {
		k.Logger(ctx).Error("failed to calculate netting", "error", err)
		return err
	}

	if len(pairs) == 0 {
		return nettingtypes.ErrNettingNotRequired
	}

	// Validate pairs
	if err := k.ValidateNettingPairs(ctx, pairs); err != nil {
		k.Logger(ctx).Error("netting validation failed", "error", err)
		return err
	}

	// Execute with rollback protection
	if err := k.ExecuteNettingWithRollback(ctx, pairs); err != nil {
		return err
	}

	// Update last netting block
	k.setLastNettingBlock(ctx, currentBlock)

	return nil
}

// GetNettingStatus returns the status of the netting system
func (k Keeper) GetNettingStatus(ctx sdk.Context) NettingSystemStatus {
	lastBlock := k.getLastNettingBlock(ctx)
	currentBlock := ctx.BlockHeight()
	blocksUntilNext := int64(10) - (currentBlock - lastBlock)
	if blocksUntilNext < 0 {
		blocksUntilNext = 0
	}

	// Count pending netting pairs
	pairs, _ := k.CalculateNetting(ctx)

	return NettingSystemStatus{
		LastNettingBlock:   lastBlock,
		CurrentBlock:       currentBlock,
		BlocksUntilNext:    blocksUntilNext,
		PendingPairCount:   len(pairs),
		IsNettingAvailable: blocksUntilNext == 0 && len(pairs) > 0,
	}
}

// NettingSystemStatus represents the current status of the netting system
type NettingSystemStatus struct {
	LastNettingBlock   int64
	CurrentBlock       int64
	BlocksUntilNext    int64
	PendingPairCount   int
	IsNettingAvailable bool
}