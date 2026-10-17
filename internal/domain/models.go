package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AccountID = uuid.UUID
type TransactionID = uuid.UUID

type Direction string

const (
	Debit  Direction = "DEBIT"
	Credit Direction = "CREDIT"
)

type Account struct {
	ID       AccountID
	OwnerID  string
	Currency string
	Type     string
	Status   string
}

type Entry struct {
	ID            uuid.UUID
	TransactionID TransactionID
	AccountID     AccountID
	Amount        decimal.Decimal
	Direction     Direction
	Currency      string
}

type Transaction struct {
	ID             TransactionID
	IdempotencyKey string
	Description    string
	Entries        []Entry
	Status         string
}

func (t *Transaction) IsBalanced() bool {
	var debit, credit decimal.Decimal
	for _, e := range t.Entries {
		if e.Direction == Debit {
			debit = debit.Add(e.Amount)
		} else {
			credit = credit.Add(e.Amount)
		}
	}
	return debit.Equal(credit)
}
