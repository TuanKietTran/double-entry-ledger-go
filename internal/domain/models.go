package domain

import (
	"fmt"
	"sort"
	"time"

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

type AccountType string

type AccountStatus string

const (
	Asset     AccountType = "ASSET"
	Liability AccountType = "LIABILITY"
	Equity    AccountType = "EQUITY"
	Revenue   AccountType = "REVENUE"
	Expense   AccountType = "EXPENSE"

	AccountActive   AccountStatus = "ACTIVE"
	AccountInactive AccountStatus = "INACTIVE"
)

type Account struct {
	ID       AccountID
	OwnerID  string
	Currency string
	Type     AccountType
	Status   AccountStatus
}

type Entry struct {
	ID            uuid.UUID
	CreatedAt     time.Time
	TransactionID TransactionID
	AccountID     AccountID
	Amount        decimal.Decimal
	Direction     Direction
	Currency      string
}

func (a Account) Validate() error {
	if a.ID == uuid.Nil {
		return fmt.Errorf("account id is required")
	}
	if !validCurrency(a.Currency) {
		return fmt.Errorf("invalid currency %q", a.Currency)
	}
	switch a.Type {
	case Asset, Liability, Equity, Revenue, Expense:
	default:
		return fmt.Errorf("invalid account type %q", a.Type)
	}
	if a.Status != AccountActive && a.Status != AccountInactive {
		return fmt.Errorf("invalid account status %q", a.Status)
	}
	return nil
}

type Transaction struct {
	ID             TransactionID
	IdempotencyKey string
	Description    string
	Entries        []Entry
	Status         string
}

func (t *Transaction) SamePayload(other Transaction) bool {
	if t.IdempotencyKey != other.IdempotencyKey || t.Description != other.Description || len(t.Entries) != len(other.Entries) {
		return false
	}
	left := append([]Entry(nil), t.Entries...)
	right := append([]Entry(nil), other.Entries...)
	less := func(entries []Entry) func(int, int) bool {
		return func(i, j int) bool {
			a, b := entries[i], entries[j]
			if a.AccountID != b.AccountID {
				return a.AccountID.String() < b.AccountID.String()
			}
			if a.Direction != b.Direction {
				return a.Direction < b.Direction
			}
			if a.Currency != b.Currency {
				return a.Currency < b.Currency
			}
			return a.Amount.LessThan(b.Amount)
		}
	}
	sort.Slice(left, less(left))
	sort.Slice(right, less(right))
	for i, entry := range left {
		candidate := right[i]
		if entry.AccountID != candidate.AccountID || !entry.Amount.Equal(candidate.Amount) || entry.Direction != candidate.Direction || entry.Currency != candidate.Currency {
			return false
		}
	}
	return true
}

func (t *Transaction) IsBalanced() bool {
	return t.Validate() == nil
}

// Validate enforces journal invariants without performing I/O.
func (t *Transaction) Validate() error {
	if t.IdempotencyKey == "" {
		return fmt.Errorf("idempotency key is required")
	}
	if len(t.Entries) < 2 {
		return fmt.Errorf("transaction requires at least two entries")
	}
	currency := t.Entries[0].Currency
	if !validCurrency(currency) {
		return fmt.Errorf("invalid currency %q", currency)
	}
	var debit, credit decimal.Decimal
	for i, e := range t.Entries {
		if e.AccountID == uuid.Nil {
			return fmt.Errorf("entry %d has empty account id", i)
		}
		if !e.Amount.IsPositive() {
			return fmt.Errorf("entry %d amount must be positive", i)
		}
		if e.Direction != Debit && e.Direction != Credit {
			return fmt.Errorf("entry %d has invalid direction %q", i, e.Direction)
		}
		if e.Currency != currency {
			return fmt.Errorf("transaction entries must use one currency")
		}
		if e.Direction == Debit {
			debit = debit.Add(e.Amount)
		} else {
			credit = credit.Add(e.Amount)
		}
	}
	if !debit.Equal(credit) {
		return fmt.Errorf("transaction is not balanced")
	}
	return nil
}

func validCurrency(currency string) bool {
	if len(currency) != 3 {
		return false
	}
	for _, c := range currency {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}
