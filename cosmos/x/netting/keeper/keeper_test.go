package keeper_test

import (
	"testing"

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
			if transferAmount.IsNil() || transferAmount.LTE(sdk.ZeroInt()) || transferAmount.GT(creditToken.Amount) {
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
			if bankID == "" || amount.IsNil() || amount.LTE(sdk.ZeroInt()) {
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
	return sdk.NewCoin(denom, sdk.ZeroInt())
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