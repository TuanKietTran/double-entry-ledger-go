package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/example/double-entry-ledger/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type InMemoryLedger struct {
	mu           sync.RWMutex
	transactions map[domain.TransactionID]domain.Transaction
	entries      map[domain.AccountID][]domain.Entry
	idempotency  map[string]domain.TransactionID
}

func NewInMemoryLedger() *InMemoryLedger {
	return &InMemoryLedger{
		transactions: make(map[domain.TransactionID]domain.Transaction),
		entries:      make(map[domain.AccountID][]domain.Entry),
		idempotency:  make(map[string]domain.TransactionID),
	}
}

func (r *InMemoryLedger) SaveTransaction(ctx context.Context, tx domain.Transaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.idempotency[tx.IdempotencyKey]; ok {
		return fmt.Errorf("idempotency key already used: %s", existing)
	}

	r.transactions[tx.ID] = tx
	r.idempotency[tx.IdempotencyKey] = tx.ID

	for _, e := range tx.Entries {
		r.entries[e.AccountID] = append(r.entries[e.AccountID], e)
	}
	return nil
}

func (r *InMemoryLedger) GetBalance(ctx context.Context, accountID domain.AccountID) (decimal.Decimal, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var bal decimal.Decimal
	currency := "USD"
	for _, e := range r.entries[accountID] {
		currency = e.Currency
		if e.Direction == domain.Debit {
			bal = bal.Add(e.Amount)
		} else {
			bal = bal.Sub(e.Amount)
		}
	}
	return bal, currency, nil
}

func (r *InMemoryLedger) GetEntries(ctx context.Context, accountID domain.AccountID) ([]domain.Entry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.entries[accountID], nil
}
