package testutil

import (
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	
	"github.com/interbank-netting/cosmos/types"
)

// PropertyTestConfig holds configuration for property-based tests
type PropertyTestConfig struct {
	MinSuccessfulTests int
	MaxDiscardRatio    float64
	Workers            int
	Rng                *gopter.LockedSource
}

// DefaultPropertyTestConfig returns default configuration for property tests
func DefaultPropertyTestConfig() *PropertyTestConfig {
	return &PropertyTestConfig{
		MinSuccessfulTests: 100, // Minimum 100 iterations as specified
		MaxDiscardRatio:    5.0,
		Workers:            1,
		Rng:                gopter.NewLockedSource(time.Now().UnixNano()),
	}
}

// NewPropertyTester creates a new property tester with default configuration
func NewPropertyTester(t *testing.T) *gopter.Properties {
	config := DefaultPropertyTestConfig()
	parameters := &gopter.TestParameters{
		MinSuccessfulTests: config.MinSuccessfulTests,
		MaxDiscardRatio:    config.MaxDiscardRatio,
		Workers:            config.Workers,
		Rng:                config.Rng,
	}
	return gopter.NewProperties(parameters)
}

// Generators for property-based testing

// GenValidAddress generates valid Cosmos addresses
func GenValidAddress() gopter.Gen {
	return gen.SliceOfN(20, gen.UInt8()).Map(func(bytes []byte) sdk.AccAddress {
		return sdk.AccAddress(bytes)
	})
}

// GenValidAmount generates valid positive amounts
func GenValidAmount() gopter.Gen {
	return gen.Int64Range(1, 1000000).Map(func(i int64) sdk.Int {
		return sdk.NewInt(i)
	})
}

// GenBankID generates valid bank identifiers
func GenBankID() gopter.Gen {
	return gen.OneConstOf("bank-a", "bank-b", "bank-c", "bank-d")
}

// GenTransferEvent generates valid transfer events
func GenTransferEvent() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		GenValidAmount(),
		gen.UInt64(),
		GenBankID(),
		GenBankID(),
		gen.UInt64(),
		gen.Int64(),
	).Map(func(values []interface{}) types.TransferEvent {
		return types.TransferEvent{
			TxHash:      values[0].(string),
			Sender:      values[1].(string),
			Recipient:   values[2].(string),
			Amount:      values[3].(sdk.Int),
			Nonce:       values[4].(uint64),
			SourceChain: values[5].(string),
			DestChain:   values[6].(string),
			BlockHeight: values[7].(uint64),
			Timestamp:   values[8].(int64),
		}
	})
}

// GenCreditToken generates valid credit tokens
func GenCreditToken() gopter.Gen {
	return gopter.CombineGens(
		GenBankID(),
		GenBankID(),
		GenValidAmount(),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.Int64(),
	).Map(func(values []interface{}) types.CreditToken {
		issuerBank := values[0].(string)
		return types.CreditToken{
			Denom:      "cred-" + issuerBank,
			IssuerBank: issuerBank,
			HolderBank: values[1].(string),
			Amount:     values[2].(sdk.Int),
			OriginTx:   values[3].(string),
			IssuedAt:   values[4].(int64),
		}
	})
}

// GenBankPair generates valid bank pairs for netting
func GenBankPair() gopter.Gen {
	return gopter.CombineGens(
		GenBankID(),
		GenBankID(),
		GenValidAmount(),
		GenValidAmount(),
	).SuchThat(func(values []interface{}) bool {
		// Ensure banks are different
		return values[0].(string) != values[1].(string)
	}).Map(func(values []interface{}) types.BankPair {
		bankA := values[0].(string)
		bankB := values[1].(string)
		amountA := values[2].(sdk.Int)
		amountB := values[3].(sdk.Int)
		
		var netAmount sdk.Int
		var netDebtor string
		
		if amountA.GT(amountB) {
			netAmount = amountA.Sub(amountB)
			netDebtor = bankB
		} else {
			netAmount = amountB.Sub(amountA)
			netDebtor = bankA
		}
		
		return types.BankPair{
			BankA:     bankA,
			BankB:     bankB,
			AmountA:   amountA,
			AmountB:   amountB,
			NetAmount: netAmount,
			NetDebtor: netDebtor,
		}
	})
}

// GenValidator generates valid validators
func GenValidator() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.SliceOfN(33, gen.UInt8()), // ECDSA public key length
		gen.Int64Range(1, 100),
		gen.Bool(),
		gen.Int64(),
	).Map(func(values []interface{}) types.Validator {
		return types.Validator{
			Address:  values[0].(string),
			PubKey:   values[1].([]byte),
			Power:    values[2].(int64),
			Active:   values[3].(bool),
			JoinedAt: values[4].(int64),
		}
	})
}

// GenValidatorSet generates valid validator sets
func GenValidatorSet() gopter.Gen {
	return gopter.CombineGens(
		gen.SliceOfN(5, GenValidator()), // Generate 5 validators
		gen.Int64(),
		gen.UInt64(),
	).Map(func(values []interface{}) types.ValidatorSet {
		validators := values[0].([]types.Validator)
		threshold := len(validators)*2/3 + 1 // 2/3 + 1 threshold
		
		return types.ValidatorSet{
			Validators:   validators,
			Threshold:    threshold,
			UpdateHeight: values[1].(int64),
			Version:      values[2].(uint64),
		}
	})
}

// GenMintCommand generates valid mint commands
func GenMintCommand() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		GenBankID(),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		GenValidAmount(),
		gen.Int64(),
	).Map(func(values []interface{}) types.MintCommand {
		return types.MintCommand{
			CommandID:   values[0].(string),
			TargetChain: values[1].(string),
			Recipient:   values[2].(string),
			Amount:      values[3].(sdk.Int),
			Signatures:  []types.ECDSASignature{}, // Empty initially
			CreatedAt:   values[4].(int64),
			Status:      types.CommandStatusPending,
		}
	})
}

// TestHelper provides utility functions for testing
type TestHelper struct {
	t *testing.T
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{t: t}
}

// AssertNoError asserts that no error occurred
func (h *TestHelper) AssertNoError(err error) {
	if err != nil {
		h.t.Fatalf("Unexpected error: %v", err)
	}
}

// AssertError asserts that an error occurred
func (h *TestHelper) AssertError(err error) {
	if err == nil {
		h.t.Fatal("Expected error but got none")
	}
}

// AssertEqual asserts that two values are equal
func (h *TestHelper) AssertEqual(expected, actual interface{}) {
	if expected != actual {
		h.t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

// AssertTrue asserts that a condition is true
func (h *TestHelper) AssertTrue(condition bool, message string) {
	if !condition {
		h.t.Fatal(message)
	}
}

// AssertFalse asserts that a condition is false
func (h *TestHelper) AssertFalse(condition bool, message string) {
	if condition {
		h.t.Fatal(message)
	}
}