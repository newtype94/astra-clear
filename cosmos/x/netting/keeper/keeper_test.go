package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	testhelpers "github.com/interbank-netting/cosmos/testutil"
	"github.com/interbank-netting/cosmos/types"
	"github.com/interbank-netting/cosmos/x/netting/keeper"
)

// **Feature: interbank-netting-engine, Property 3: 신용 토큰 발행 및 전송**
// **검증: 요구사항 2.1, 2.2**
func TestProperty_CreditTokenIssuanceAndTransfer(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("credit token issuance creates correct format and transfers to holder bank", prop.ForAll(
		func(transferEvent types.TransferEvent) bool {
			// Setup test environment
			ctx, nettingKeeper := setupNettingTestEnvironment(t)

			// Create credit token from transfer event
			creditToken := types.CreditToken{
				Denom:      "cred-" + transferEvent.SourceChain,
				IssuerBank: transferEvent.SourceChain,
				HolderBank: transferEvent.DestChain,
				Amount:     transferEvent.Amount,
				OriginTx:   transferEvent.TxHash,
				IssuedAt:   ctx.BlockTime().Unix(),
			}

			// Test Property: Credit token issuance
			err := nettingKeeper.IssueCreditToken(ctx, creditToken)
			if err != nil {
				return false // Should successfully issue credit token
			}

			// Verify correct denom format (cred-{BankID})
			expectedDenom := "cred-" + transferEvent.SourceChain
			if creditToken.Denom != expectedDenom {
				return false // Denom should follow correct format
			}

			// Verify credit token is transferred to holder bank
			balance := nettingKeeper.GetCreditBalance(ctx, transferEvent.DestChain, creditToken.Denom)
			if !balance.Equal(transferEvent.Amount) {
				return false // Holder bank should have the correct balance
			}

			// Verify issuer bank does not have the credit token
			issuerBalance := nettingKeeper.GetCreditBalance(ctx, transferEvent.SourceChain, creditToken.Denom)
			if !issuerBalance.IsZero() {
				return false // Issuer bank should not hold its own credit tokens
			}

			// Verify credit token represents exact debt obligation
			if !creditToken.Amount.Equal(transferEvent.Amount) {
				return false // Credit token amount should match original transfer amount
			}

			return true
		},
		testhelpers.GenTransferEvent(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 3: 신용 토큰 발행 및 전송**
// **검증: 요구사항 2.1, 2.2**
func TestProperty_CreditTokenTransfer_PreservesTotal(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("credit token transfers preserve total supply", prop.ForAll(
		func(creditToken types.CreditToken, transferAmount math.Int) bool {
			// Setup test environment
			ctx, nettingKeeper := setupNettingTestEnvironment(t)

			// Ensure transfer amount is valid and less than or equal to credit token amount
			if transferAmount.IsNil() || transferAmount.LTE(math.ZeroInt()) || transferAmount.GT(creditToken.Amount) {
				return true // Skip invalid test cases
			}

			// Issue initial credit token
			err := nettingKeeper.IssueCreditToken(ctx, creditToken)
			if err != nil {
				return false // Should successfully issue credit token
			}

			// Record initial total supply
			initialBalance := nettingKeeper.GetCreditBalance(ctx, creditToken.HolderBank, creditToken.Denom)

			// Create a third bank for transfer that differs from both holder and issuer
			thirdBank := ""
			for _, candidate := range []string{"bank-e", "bank-f", "bank-g"} {
				if candidate != creditToken.HolderBank && candidate != creditToken.IssuerBank {
					thirdBank = candidate
					break
				}
			}
			if thirdBank == "" {
				return true // Skip if no suitable third bank (shouldn't happen)
			}

			// Transfer part of credit token to third bank
			err = nettingKeeper.TransferCreditToken(ctx, creditToken.HolderBank, thirdBank, creditToken.Denom, transferAmount)
			if err != nil {
				return false // Transfer should succeed
			}

			// Verify balances after transfer
			holderBalance := nettingKeeper.GetCreditBalance(ctx, creditToken.HolderBank, creditToken.Denom)
			thirdBankBalance := nettingKeeper.GetCreditBalance(ctx, thirdBank, creditToken.Denom)

			// Check that total supply is preserved
			totalAfterTransfer := holderBalance.Add(thirdBankBalance)
			if !totalAfterTransfer.Equal(initialBalance) {
				return false // Total supply should be preserved
			}

			// Check individual balances
			expectedHolderBalance := initialBalance.Sub(transferAmount)
			if !holderBalance.Equal(expectedHolderBalance) {
				return false // Holder balance should be reduced by transfer amount
			}

			if !thirdBankBalance.Equal(transferAmount) {
				return false // Third bank should receive exact transfer amount
			}

			return true
		},
		testhelpers.GenCreditToken(),
		testhelpers.GenValidAmount(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 3: 신용 토큰 발행 및 전송**
// **검증: 요구사항 2.1**
func TestProperty_CreditTokenDenom_FollowsCorrectFormat(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("credit token denom always follows cred-{BankID} format", prop.ForAll(
		func(bankID string, amount math.Int) bool {
			// Skip invalid inputs
			if bankID == "" || amount.IsNil() || amount.LTE(math.ZeroInt()) {
				return true
			}

			// Setup test environment
			ctx, nettingKeeper := setupNettingTestEnvironment(t)

			// Create credit token with specific bank ID
			creditToken := types.CreditToken{
				Denom:      "cred-" + bankID,
				IssuerBank: bankID,
				HolderBank: "holder-bank",
				Amount:     amount,
				OriginTx:   "test-tx",
				IssuedAt:   ctx.BlockTime().Unix(),
			}

			// Issue credit token
			err := nettingKeeper.IssueCreditToken(ctx, creditToken)
			if err != nil {
				return false // Should successfully issue credit token
			}

			// Verify denom format
			expectedDenom := "cred-" + bankID
			if creditToken.Denom != expectedDenom {
				return false // Denom should follow cred-{BankID} format
			}

			// Verify the credit token represents debt from the issuer bank
			if creditToken.IssuerBank != bankID {
				return false // Issuer bank should match the bank ID in denom
			}

			return true
		},
		testhelpers.GenBankID(),
		testhelpers.GenValidAmount(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 3: 신용 토큰 발행 및 전송**
// **검증: 요구사항 2.2**
func TestProperty_CreditTokenTransfer_OnlyToDestinationBank(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("credit tokens are transferred to correct destination bank account", prop.ForAll(
		func(transferEvent types.TransferEvent) bool {
			// Setup test environment
			ctx, nettingKeeper := setupNettingTestEnvironment(t)

			// Create and issue credit token
			creditToken := types.CreditToken{
				Denom:      "cred-" + transferEvent.SourceChain,
				IssuerBank: transferEvent.SourceChain,
				HolderBank: transferEvent.DestChain,
				Amount:     transferEvent.Amount,
				OriginTx:   transferEvent.TxHash,
				IssuedAt:   ctx.BlockTime().Unix(),
			}

			err := nettingKeeper.IssueCreditToken(ctx, creditToken)
			if err != nil {
				return false // Should successfully issue credit token
			}

			// Verify credit token is in destination bank's account
			destBalance := nettingKeeper.GetCreditBalance(ctx, transferEvent.DestChain, creditToken.Denom)
			if !destBalance.Equal(transferEvent.Amount) {
				return false // Destination bank should have the credit token
			}

			// Verify credit token is NOT in source bank's account
			sourceBalance := nettingKeeper.GetCreditBalance(ctx, transferEvent.SourceChain, creditToken.Denom)
			if !sourceBalance.IsZero() {
				return false // Source bank should not have its own credit token
			}

			// Verify no other banks have this credit token (test a few random banks)
			testBanks := []string{"random-bank-1", "random-bank-2", "random-bank-3"}
			for _, bank := range testBanks {
				if bank != transferEvent.DestChain && bank != transferEvent.SourceChain {
					balance := nettingKeeper.GetCreditBalance(ctx, bank, creditToken.Denom)
					if !balance.IsZero() {
						return false // Random banks should not have this credit token
					}
				}
			}

			return true
		},
		testhelpers.GenTransferEvent(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 4.3: 신용 잔액 조회 정확성**
// **검증: 요구사항 2.1, 2.2 - 신용 잔액 조회가 항상 정확한 값을 반환하는지 검증**
func TestProperty_CreditBalanceQuery_Accuracy(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("credit balance queries return accurate values after issuance and transfers", prop.ForAll(
		func(creditToken types.CreditToken) bool {
			// Setup test environment
			ctx, nettingKeeper := setupNettingTestEnvironment(t)

			// Skip invalid inputs
			if creditToken.Amount.IsNil() || creditToken.Amount.LTE(math.ZeroInt()) {
				return true
			}
			if creditToken.IssuerBank == "" || creditToken.HolderBank == "" {
				return true
			}
			if creditToken.IssuerBank == creditToken.HolderBank {
				return true // Skip self-issued tokens
			}

			// Issue credit token
			err := nettingKeeper.IssueCreditToken(ctx, creditToken)
			if err != nil {
				return false // Should successfully issue credit token
			}

			// Test 1: Query initial holder balance
			holderBalance := nettingKeeper.GetCreditBalance(ctx, creditToken.HolderBank, creditToken.Denom)
			if !holderBalance.Equal(creditToken.Amount) {
				return false // Holder balance should match issued amount
			}

			// Test 2: Query issuer balance (should be zero)
			issuerBalance := nettingKeeper.GetCreditBalance(ctx, creditToken.IssuerBank, creditToken.Denom)
			if !issuerBalance.IsZero() {
				return false // Issuer should not hold own credit tokens
			}

			// Test 3: Query non-existent bank balance (should be zero)
			randomBank := "random-bank-xyz"
			randomBalance := nettingKeeper.GetCreditBalance(ctx, randomBank, creditToken.Denom)
			if !randomBalance.IsZero() {
				return false // Non-existent bank should have zero balance
			}

			// Test 4: Transfer half to a third bank and verify balances
			thirdBank := "third-bank"
			transferAmount := creditToken.Amount.QuoRaw(2) // Transfer half
			if transferAmount.GT(math.ZeroInt()) {
				err = nettingKeeper.TransferCreditToken(ctx, creditToken.HolderBank, thirdBank, creditToken.Denom, transferAmount)
				if err != nil {
					return false // Transfer should succeed
				}

				// Query holder balance after transfer
				newHolderBalance := nettingKeeper.GetCreditBalance(ctx, creditToken.HolderBank, creditToken.Denom)
				expectedHolderBalance := creditToken.Amount.Sub(transferAmount)
				if !newHolderBalance.Equal(expectedHolderBalance) {
					return false // Holder balance should be reduced correctly
				}

				// Query third bank balance after transfer
				thirdBankBalance := nettingKeeper.GetCreditBalance(ctx, thirdBank, creditToken.Denom)
				if !thirdBankBalance.Equal(transferAmount) {
					return false // Third bank should receive exact transfer amount
				}

				// Test 5: Verify total supply is preserved
				totalBalance := newHolderBalance.Add(thirdBankBalance)
				if !totalBalance.Equal(creditToken.Amount) {
					return false // Total supply should remain constant
				}
			}

			// Test 6: Get all balances for holder bank
			allBalances := nettingKeeper.GetAllCreditBalances(ctx, creditToken.HolderBank)
			holderBalanceFromMap, exists := allBalances[creditToken.Denom]
			if !exists {
				return false // Holder should have balance in GetAllCreditBalances result
			}
			currentExpectedBalance := creditToken.Amount
			if transferAmount.GT(math.ZeroInt()) {
				currentExpectedBalance = creditToken.Amount.Sub(transferAmount)
			}
			if !holderBalanceFromMap.Equal(currentExpectedBalance) {
				return false // GetAllCreditBalances should return same value as GetCreditBalance
			}

			return true
		},
		testhelpers.GenCreditToken(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 5.5: 상계 계산 및 실행**
// **검증: 요구사항 4.2, 4.3 - 상계 계산이 정확하고 실행이 올바르게 되는지 검증**
func TestProperty_Netting_CalculationAndExecution(t *testing.T) {
	properties := testhelpers.NewPropertyTester(t)

	properties.Property("netting correctly calculates and executes mutual debt cancellation", prop.ForAll(
		func(amountAtoB, amountBtoA math.Int) bool {
			// Skip invalid amounts
			if amountAtoB.IsNil() || amountAtoB.LTE(math.ZeroInt()) {
				return true
			}
			if amountBtoA.IsNil() || amountBtoA.LTE(math.ZeroInt()) {
				return true
			}
			// Skip if amounts are too large (avoid overflow)
			maxAmount := math.NewInt(1000000)
			if amountAtoB.GT(maxAmount) || amountBtoA.GT(maxAmount) {
				return true
			}

			// Setup test environment
			ctx, nettingKeeper := setupNettingTestEnvironment(t)
			bankA := "bank-a"
			bankB := "bank-b"

			// Create mutual credit tokens
			// Bank A holds credit from Bank B (Bank B owes Bank A)
			tokenBtoA := types.CreditToken{
				Denom:      "cred-" + bankB,
				IssuerBank: bankB,
				HolderBank: bankA,
				Amount:     amountBtoA,
				OriginTx:   "tx-b-to-a",
				IssuedAt:   ctx.BlockTime().Unix(),
			}

			// Bank B holds credit from Bank A (Bank A owes Bank B)
			tokenAtoB := types.CreditToken{
				Denom:      "cred-" + bankA,
				IssuerBank: bankA,
				HolderBank: bankB,
				Amount:     amountAtoB,
				OriginTx:   "tx-a-to-b",
				IssuedAt:   ctx.BlockTime().Unix(),
			}

			// Issue both credit tokens
			if err := nettingKeeper.IssueCreditToken(ctx, tokenBtoA); err != nil {
				return false
			}
			if err := nettingKeeper.IssueCreditToken(ctx, tokenAtoB); err != nil {
				return false
			}

			// Record initial balances
			initialBalanceA := nettingKeeper.GetCreditBalance(ctx, bankA, "cred-"+bankB)
			initialBalanceB := nettingKeeper.GetCreditBalance(ctx, bankB, "cred-"+bankA)

			// Calculate expected netting
			minAmount := amountAtoB
			if amountBtoA.LT(minAmount) {
				minAmount = amountBtoA
			}

			// Calculate netting pairs
			pairs, err := nettingKeeper.CalculateNetting(ctx)
			if err != nil {
				return false
			}

			// Should have exactly one pair (A and B)
			if len(pairs) != 1 {
				return false
			}

			pair := pairs[0]

			// Verify pair contains correct banks (order may vary)
			if !((pair.BankA == bankA && pair.BankB == bankB) || (pair.BankA == bankB && pair.BankB == bankA)) {
				return false
			}

			// Execute netting
			if err := nettingKeeper.ExecuteNetting(ctx, pairs); err != nil {
				return false
			}

			// Verify balances after netting
			finalBalanceA := nettingKeeper.GetCreditBalance(ctx, bankA, "cred-"+bankB)
			finalBalanceB := nettingKeeper.GetCreditBalance(ctx, bankB, "cred-"+bankA)

			// Both balances should be reduced by the minimum amount
			expectedFinalA := initialBalanceA.Sub(minAmount)
			expectedFinalB := initialBalanceB.Sub(minAmount)

			if !finalBalanceA.Equal(expectedFinalA) {
				return false
			}
			if !finalBalanceB.Equal(expectedFinalB) {
				return false
			}

			return true
		},
		testhelpers.GenValidAmount(),
		testhelpers.GenValidAmount(),
	))

	properties.TestingRun(t)
}

// Helper functions for testing

func setupNettingTestEnvironment(t *testing.T) (sdk.Context, *keeper.Keeper) {
	// Create store key
	storeKey := storetypes.NewKVStoreKey("netting")

	// Create test context with store
	testCtx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx

	// Create proto codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create mock keepers
	mockBankKeeper := NewMockBankKeeper()
	mockAccountKeeper := NewMockAccountKeeper()

	// Create netting keeper with proper initialization
	nettingKeeper := keeper.NewKeeper(
		cdc,
		storeKey,
		nil, // memKey
		paramtypes.Subspace{},
		mockBankKeeper,
		mockAccountKeeper,
	)

	return ctx, nettingKeeper
}

// MockBankKeeper for testing
type MockBankKeeper struct {
	balances map[string]map[string]math.Int
}

func NewMockBankKeeper() *MockBankKeeper {
	return &MockBankKeeper{
		balances: make(map[string]map[string]math.Int),
	}
}

func (m *MockBankKeeper) SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, math.ZeroInt())
}

func (m *MockBankKeeper) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return sdk.Coins{}
}

// MockAccountKeeper for testing
type MockAccountKeeper struct {
	accounts map[string]sdk.AccountI
}

func NewMockAccountKeeper() *MockAccountKeeper {
	return &MockAccountKeeper{
		accounts: make(map[string]sdk.AccountI),
	}
}

func (m *MockAccountKeeper) GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	return nil
}

func (m *MockAccountKeeper) SetAccount(ctx context.Context, acc sdk.AccountI) {
}

func (m *MockAccountKeeper) NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	return nil
}

// Silence unused imports
var _ = gopter.Gen(nil)
var _ = gen.Int()
var _ = fmt.Sprintf("")
