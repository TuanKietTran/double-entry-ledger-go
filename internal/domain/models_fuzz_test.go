package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func FuzzCurrencyValidation(f *testing.F) {
	for _, seed := range []string{"USD", "EUR", "usd", "US", "USDD", "1SD", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, currency string) {
		valid := validCurrency(currency)
		if valid {
			if len(currency) != 3 {
				t.Fatalf("accepted currency with length %d", len(currency))
			}
			for _, c := range currency {
				if c < 'A' || c > 'Z' {
					t.Fatalf("accepted non-uppercase ASCII currency %q", currency)
				}
			}
		}
	})
}

func FuzzBalancedTotalsInvariant(f *testing.F) {
	f.Add(int64(1), int64(1))
	f.Add(int64(99), int64(99))
	f.Add(int64(2), int64(3))
	f.Fuzz(func(t *testing.T, debit, credit int64) {
		if debit <= 0 || credit <= 0 {
			t.Skip()
		}
		tx := Transaction{Entries: []Entry{
			{AccountID: uuid.New(), Amount: decimal.NewFromInt(debit), Direction: Debit, Currency: "USD"},
			{AccountID: uuid.New(), Amount: decimal.NewFromInt(credit), Direction: Credit, Currency: "USD"},
		}}
		if tx.IsBalanced() != (debit == credit) {
			t.Fatalf("balance invariant mismatch: debit=%d credit=%d", debit, credit)
		}
	})
}
