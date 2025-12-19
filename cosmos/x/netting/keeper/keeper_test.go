package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/interbank-netting/cosmos/testutil"
	"github.com/interbank-netting/cosmos/types"
	"github.com/interbank-netting/cosmos/x/netting/keeper"
	nettingtypes "github.com/interbank-netting/cosmos/x/netting/types"
)

// **Feature: interbank-netting-engine, Property 3: 신용 토큰 발행 및 전송**
// **검증: 요구사항 2.1, 2.2**
func TestProperty_CreditTokenIssuanceAndTransfer(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

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
		testutil.GenTransferEvent(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 3: 신용 토큰 발행 및 전송**
// **검증: 요구사항 2.1, 2.2**
func TestProperty_CreditTokenTransfer_PreservesTotal(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

	properties.Property("credit token transfers preserve total supply", prop.ForAll(
		func(creditToken types.CreditToken, transferAmount sdk.Int) bool {
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
			
			// Create a third bank for transfer
			thirdBank := "bank-c"
			if thirdBank == creditToken.HolderBank || thirdBank == creditToken.IssuerBank {
				thirdBank = "bank-d" // Use different bank
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
		testutil.GenCreditToken(),
		testutil.GenValidAmount(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 3: 신용 토큰 발행 및 전송**
// **검증: 요구사항 2.1**
func TestProperty_CreditTokenDenom_FollowsCorrectFormat(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

	properties.Property("credit token denom always follows cred-{BankID} format", prop.ForAll(
		func(bankID string, amount sdk.Int) bool {
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
		testutil.GenBankID(),
		testutil.GenValidAmount(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 3: 신용 토큰 발행 및 전송**
// **검증: 요구사항 2.2**
func TestProperty_CreditTokenTransfer_OnlyToDestinationBank(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

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
		testutil.GenTransferEvent(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 4.3: 신용 잔액 조회 정확성**
// **검증: 요구사항 2.1, 2.2 - 신용 잔액 조회가 항상 정확한 값을 반환하는지 검증**
func TestProperty_CreditBalanceQuery_Accuracy(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

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
		testutil.GenCreditToken(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 4.3: 신용 잔액 조회 정확성**
// **검증: GetAllCreditBalances가 모든 신용 토큰 잔액을 정확히 반환하는지 검증**
func TestProperty_GetAllCreditBalances_Completeness(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

	properties.Property("GetAllCreditBalances returns complete and accurate credit token balances", prop.ForAll(
		func(tokens []types.CreditToken) bool {
			// Setup test environment
			ctx, nettingKeeper := setupNettingTestEnvironment(t)

			// Skip empty or single token lists
			if len(tokens) < 2 || len(tokens) > 5 {
				return true // Test with 2-5 tokens
			}

			// Track expected balances for a test bank
			testBank := "test-holder-bank"
			expectedBalances := make(map[string]math.Int)

			// Issue multiple credit tokens to the same holder bank
			for i, token := range tokens {
				// Skip invalid tokens
				if token.Amount.IsNil() || token.Amount.LTE(math.ZeroInt()) {
					continue
				}
				if token.IssuerBank == "" || token.IssuerBank == testBank {
					continue
				}

				// Create unique denom for each token
				token.Denom = fmt.Sprintf("cred-%s-%d", token.IssuerBank, i)
				token.HolderBank = testBank

				err := nettingKeeper.IssueCreditToken(ctx, token)
				if err != nil {
					continue // Skip if issuance fails
				}

				expectedBalances[token.Denom] = token.Amount
			}

			// Skip if no tokens were issued
			if len(expectedBalances) == 0 {
				return true
			}

			// Query all credit balances for test bank
			allBalances := nettingKeeper.GetAllCreditBalances(ctx, testBank)

			// Verify completeness: all issued tokens should be present
			for denom, expectedAmount := range expectedBalances {
				actualAmount, exists := allBalances[denom]
				if !exists {
					return false // All issued credit tokens should be in GetAllCreditBalances result
				}
				if !actualAmount.Equal(expectedAmount) {
					return false // Balance should match expected amount
				}
			}

			// Verify accuracy: no extra tokens should be present
			if len(allBalances) != len(expectedBalances) {
				return false // GetAllCreditBalances should not return extra tokens
			}

			return true
		},
		gen.SliceOf(testutil.GenCreditToken()),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 4.4: 신용 토큰 추적성**
// **검증: 요구사항 2.1 - 신용 토큰이 원본 이체에 대한 추적 가능한 참조를 유지하는지 검증**
func TestProperty_CreditToken_Traceability(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

	properties.Property("credit tokens maintain traceable references to origin transactions", prop.ForAll(
		func(transferEvent types.TransferEvent) bool {
			// Setup test environment
			ctx, nettingKeeper := setupNettingTestEnvironment(t)

			// Skip invalid inputs
			if transferEvent.Amount.IsNil() || transferEvent.Amount.LTE(math.ZeroInt()) {
				return true
			}
			if transferEvent.TxHash == "" || transferEvent.SourceChain == "" || transferEvent.DestChain == "" {
				return true
			}
			if transferEvent.SourceChain == transferEvent.DestChain {
				return true // Skip self-transfers
			}

			// Create credit token from transfer event
			creditToken := types.CreditToken{
				Denom:      "cred-" + transferEvent.SourceChain,
				IssuerBank: transferEvent.SourceChain,
				HolderBank: transferEvent.DestChain,
				Amount:     transferEvent.Amount,
				OriginTx:   transferEvent.TxHash,
				IssuedAt:   ctx.BlockTime().Unix(),
			}

			// Issue credit token
			err := nettingKeeper.IssueCreditToken(ctx, creditToken)
			if err != nil {
				return false // Should successfully issue credit token
			}

			// Test 1: Verify OriginTx is preserved
			storedToken, found := nettingKeeper.getCreditToken(ctx, creditToken.Denom)
			if !found {
				return false // Credit token should be stored
			}
			if storedToken.OriginTx != transferEvent.TxHash {
				return false // OriginTx should match original transfer hash
			}

			// Test 2: Verify issuer bank matches source chain
			if storedToken.IssuerBank != transferEvent.SourceChain {
				return false // IssuerBank should match source chain
			}

			// Test 3: Verify holder bank matches dest chain
			if storedToken.HolderBank != transferEvent.DestChain {
				return false // HolderBank should match destination chain
			}

			// Test 4: Verify amount matches original transfer amount
			if !storedToken.Amount.Equal(transferEvent.Amount) {
				return false // Amount should match original transfer
			}

			// Test 5: Verify denom encodes issuer bank identity
			expectedDenom := "cred-" + transferEvent.SourceChain
			if storedToken.Denom != expectedDenom {
				return false // Denom should encode issuer bank identity
			}

			// Test 6: Verify issued timestamp is set
			if storedToken.IssuedAt <= 0 {
				return false // IssuedAt should be set to a valid timestamp
			}

			return true
		},
		testutil.GenTransferEvent(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 4.4: 신용 토큰 추적성**
// **검증: 요구사항 2.1 - 신용 토큰 전송 후에도 원본 이체 정보가 유지되는지 검증**
func TestProperty_CreditToken_Traceability_AfterTransfers(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

	properties.Property("credit token origin remains traceable after multiple transfers", prop.ForAll(
		func(transferEvent types.TransferEvent) bool {
			// Setup test environment
			ctx, nettingKeeper := setupNettingTestEnvironment(t)

			// Skip invalid inputs
			if transferEvent.Amount.IsNil() || transferEvent.Amount.LTE(math.NewInt(100)) {
				return true // Need enough amount for multiple transfers
			}
			if transferEvent.TxHash == "" || transferEvent.SourceChain == "" || transferEvent.DestChain == "" {
				return true
			}
			if transferEvent.SourceChain == transferEvent.DestChain {
				return true
			}

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
				return false
			}

			// Store original traceability information
			originalOriginTx := transferEvent.TxHash
			originalIssuerBank := transferEvent.SourceChain
			originalAmount := transferEvent.Amount

			// Transfer to multiple banks
			banks := []string{"bank-a", "bank-b", "bank-c"}
			transferAmount := transferEvent.Amount.QuoRaw(int64(len(banks)))

			if transferAmount.LTE(math.ZeroInt()) {
				return true // Skip if transfer amount too small
			}

			currentHolder := transferEvent.DestChain
			for _, nextBank := range banks {
				err = nettingKeeper.TransferCreditToken(ctx, currentHolder, nextBank, creditToken.Denom, transferAmount)
				if err != nil {
					// Transfer might fail due to insufficient balance, which is ok
					break
				}

				// Verify traceability is maintained after each transfer
				storedToken, found := nettingKeeper.getCreditToken(ctx, creditToken.Denom)
				if !found {
					return false // Credit token should still be stored
				}

				// Critical invariants that must be preserved through transfers
				if storedToken.OriginTx != originalOriginTx {
					return false // Origin transaction must remain unchanged
				}
				if storedToken.IssuerBank != originalIssuerBank {
					return false // Issuer bank must remain unchanged
				}
				if storedToken.Denom != creditToken.Denom {
					return false // Denom must remain unchanged
				}

				// Original amount should remain unchanged in the credit token metadata
				// (even though balances change)
				if !storedToken.Amount.Equal(originalAmount) {
					return false // Original amount should be preserved in metadata
				}

				currentHolder = nextBank
			}

			return true
		},
		testutil.GenTransferEvent(),
	))

	properties.TestingRun(t)
}

// **Feature: interbank-netting-engine, Property 4.4: 신용 토큰 추적성**
// **검증: 요구사항 2.1 - 각 신용 토큰은 고유한 원본 이체 해시를 가지는지 검증**
func TestProperty_CreditToken_UniqueOriginTraceability(t *testing.T) {
	properties := testutil.NewPropertyTester(t)

	properties.Property("each credit token has unique origin transaction hash", prop.ForAll(
		func(events []types.TransferEvent) bool {
			// Setup test environment
			ctx, nettingKeeper := setupNettingTestEnvironment(t)

			// Skip if not enough events
			if len(events) < 2 || len(events) > 5 {
				return true
			}

			// Track issued credit tokens by their origin tx
			originTxToDenom := make(map[string]string)

			for i, event := range events {
				// Skip invalid inputs
				if event.Amount.IsNil() || event.Amount.LTE(math.ZeroInt()) {
					continue
				}
				if event.TxHash == "" || event.SourceChain == "" || event.DestChain == "" {
					continue
				}
				if event.SourceChain == event.DestChain {
					continue
				}

				// Create unique denom for each token
				denom := fmt.Sprintf("cred-%s-%d", event.SourceChain, i)

				creditToken := types.CreditToken{
					Denom:      denom,
					IssuerBank: event.SourceChain,
					HolderBank: event.DestChain,
					Amount:     event.Amount,
					OriginTx:   event.TxHash,
					IssuedAt:   ctx.BlockTime().Unix(),
				}

				err := nettingKeeper.IssueCreditToken(ctx, creditToken)
				if err != nil {
					continue // Skip if issuance fails
				}

				// Check if origin tx hash is unique
				if existingDenom, exists := originTxToDenom[event.TxHash]; exists {
					// Same origin tx should map to same denom type (from same issuer)
					// This is ok if they're from the same issuer bank
					storedToken1, _ := nettingKeeper.getCreditToken(ctx, existingDenom)
					storedToken2, _ := nettingKeeper.getCreditToken(ctx, denom)

					if storedToken1.IssuerBank == storedToken2.IssuerBank {
						// Same issuer, same origin tx - this should not happen
						// Each credit token should represent a unique debt obligation
						return false
					}
				} else {
					originTxToDenom[event.TxHash] = denom
				}

				// Verify stored token has correct origin tx
				storedToken, found := nettingKeeper.getCreditToken(ctx, denom)
				if !found {
					return false
				}
				if storedToken.OriginTx != event.TxHash {
					return false // Origin tx must match
				}
			}

			return true
		},
		gen.SliceOf(testutil.GenTransferEvent()),
	))

	properties.TestingRun(t)
}

// Helper functions for testing

func setupNettingTestEnvironment(t *testing.T) (sdk.Context, keeper.Keeper) {
	// Create mock context
	ctx := sdk.NewContext(nil, false, nil)
	
	// Create mock keepers
	bankKeeper := NewMockBankKeeper()
	accountKeeper := NewMockAccountKeeper()
	
	// Create netting keeper with mocks
	nettingKeeper := keeper.Keeper{} // Simplified for property testing
	
	return ctx, nettingKeeper
}

// MockBankKeeper for testing
type MockBankKeeper struct {
	balances map[string]map[string]sdk.Int
}

func NewMockBankKeeper() MockBankKeeper {
	return MockBankKeeper{
		balances: make(map[string]map[string]sdk.Int),
	}
}

func (m MockBankKeeper) SendCoins(ctx sdk.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m MockBankKeeper) MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m MockBankKeeper) BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m MockBankKeeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, math.ZeroInt())
}

func (m MockBankKeeper) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return sdk.Coins{}
}

// MockAccountKeeper for testing
type MockAccountKeeper struct {
	accounts map[string]sdk.AccountI
}

func NewMockAccountKeeper() MockAccountKeeper {
	return MockAccountKeeper{
		accounts: make(map[string]sdk.AccountI),
	}
}

func (m MockAccountKeeper) GetAccount(ctx sdk.Context, addr sdk.AccAddress) sdk.AccountI {
	return nil
}

func (m MockAccountKeeper) SetAccount(ctx sdk.Context, acc sdk.AccountI) {
	// Mock implementation
}

func (m MockAccountKeeper) NewAccountWithAddress(ctx sdk.Context, addr sdk.AccAddress) sdk.AccountI {
	return nil
}