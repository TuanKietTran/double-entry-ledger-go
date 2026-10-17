package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestTransactionSamePayload(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	base := Transaction{IdempotencyKey: "stable", Description: "memo", Entries: []Entry{
		{AccountID: a, Amount: decimal.NewFromInt(8), Direction: Debit, Currency: "USD"},
		{AccountID: b, Amount: decimal.NewFromInt(8), Direction: Credit, Currency: "USD"},
	}}
	cases := []struct {
		name  string
		other Transaction
		want  bool
	}{
		{name: "equal", other: base, want: true},
		{name: "entry order does not matter", other: Transaction{IdempotencyKey: "stable", Description: "memo", Entries: []Entry{base.Entries[1], base.Entries[0]}}, want: true},
		{name: "different key", other: Transaction{IdempotencyKey: "other", Description: "memo", Entries: base.Entries}, want: false},
		{name: "different description", other: Transaction{IdempotencyKey: "stable", Description: "other", Entries: base.Entries}, want: false},
		{name: "different amount", other: Transaction{IdempotencyKey: "stable", Description: "memo", Entries: []Entry{{AccountID: a, Amount: decimal.NewFromInt(9), Direction: Debit, Currency: "USD"}, base.Entries[1]}}, want: false},
		{name: "different direction", other: Transaction{IdempotencyKey: "stable", Description: "memo", Entries: []Entry{{AccountID: a, Amount: decimal.NewFromInt(8), Direction: Credit, Currency: "USD"}, base.Entries[1]}}, want: false},
		{name: "different currency", other: Transaction{IdempotencyKey: "stable", Description: "memo", Entries: []Entry{{AccountID: a, Amount: decimal.NewFromInt(8), Direction: Debit, Currency: "EUR"}, base.Entries[1]}}, want: false},
		{name: "different legs", other: Transaction{IdempotencyKey: "stable", Description: "memo", Entries: base.Entries[:1]}, want: false},
		{name: "sort by currency and amount", other: Transaction{IdempotencyKey: "stable", Description: "memo", Entries: []Entry{
			{AccountID: a, Amount: decimal.NewFromInt(2), Direction: Debit, Currency: "EUR"},
			{AccountID: a, Amount: decimal.NewFromInt(1), Direction: Debit, Currency: "USD"},
		}}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := base.SamePayload(tc.other); got != tc.want {
				t.Fatalf("SamePayload()=%v want %v", got, tc.want)
			}
		})
	}
}
