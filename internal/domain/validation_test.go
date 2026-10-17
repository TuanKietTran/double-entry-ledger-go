package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestTransactionValidateRejectsMalformedLegs(t *testing.T) {
	valid := func() Transaction {
		return Transaction{IdempotencyKey: "valid", Entries: []Entry{
			{AccountID: uuid.New(), Amount: decimal.NewFromInt(2), Direction: Debit, Currency: "USD"},
			{AccountID: uuid.New(), Amount: decimal.NewFromInt(2), Direction: Credit, Currency: "USD"},
		}}
	}
	cases := map[string]func(*Transaction){
		"missing key":      func(tx *Transaction) { tx.IdempotencyKey = "" },
		"one leg":          func(tx *Transaction) { tx.Entries = tx.Entries[:1] },
		"empty account":    func(tx *Transaction) { tx.Entries[0].AccountID = uuid.Nil },
		"invalid currency": func(tx *Transaction) { tx.Entries[0].Currency = "usd" },
		"negative amount": func(tx *Transaction) {
			tx.Entries[0].Amount = decimal.NewFromInt(-1)
			tx.Entries[1].Amount = decimal.NewFromInt(-1)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			tx := valid()
			mutate(&tx)
			if err := tx.Validate(); err == nil {
				t.Fatal("invalid transaction accepted")
			}
		})
	}
}
