package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type InMemoryLedger struct {
	mu           sync.RWMutex
	accounts     map[domain.AccountID]domain.Account
	transactions map[domain.TransactionID]domain.Transaction
	entries      map[domain.AccountID][]domain.Entry
	idempotency  map[string]domain.TransactionID
}

func NewInMemoryLedger() *InMemoryLedger {
	return &InMemoryLedger{
		accounts:     make(map[domain.AccountID]domain.Account),
		transactions: make(map[domain.TransactionID]domain.Transaction),
		entries:      make(map[domain.AccountID][]domain.Entry),
		idempotency:  make(map[string]domain.TransactionID),
	}
}

func (r *InMemoryLedger) CreateAccount(ctx context.Context, account domain.Account) error {
	if err := account.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.accounts[account.ID]; ok {
		if existing.SameDefinition(account) {
			return nil
		}
		return fmt.Errorf("account already exists with different attributes")
	}
	r.accounts[account.ID] = account
	return nil
}

func (r *InMemoryLedger) GetAccount(ctx context.Context, id domain.AccountID) (domain.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	account, ok := r.accounts[id]
	if !ok {
		return domain.Account{}, fmt.Errorf("account not found")
	}
	return account, nil
}

func (r *InMemoryLedger) SaveTransaction(ctx context.Context, tx domain.Transaction) error {
	if tx.ReversalOf != uuid.Nil {
		return fmt.Errorf("reversal transactions must use SaveReversal")
	}
	return r.save(ctx, tx, uuid.Nil)
}

func (r *InMemoryLedger) SaveReversal(ctx context.Context, originalID domain.TransactionID, tx domain.Transaction) error {
	if originalID == uuid.Nil || tx.ReversalOf != originalID {
		return fmt.Errorf("reversal reference does not match original transaction")
	}
	return r.save(ctx, tx, originalID)
}

func (r *InMemoryLedger) save(ctx context.Context, tx domain.Transaction, reversalOf domain.TransactionID) error {
	if err := tx.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.idempotency[tx.IdempotencyKey]; ok {
		stored := r.transactions[existing]
		if stored.SamePayload(tx) {
			return nil
		}
		return fmt.Errorf("%w: %s", ErrIdempotencyConflict, existing)
	}
	if reversalOf != uuid.Nil {
		original, ok := r.transactions[reversalOf]
		if !ok || original.ReversalOf != uuid.Nil {
			return fmt.Errorf("original transaction not found or is itself a reversal")
		}
		for _, existing := range r.transactions {
			if existing.ReversalOf == reversalOf {
				return fmt.Errorf("transaction already reversed")
			}
		}
	}
	usedAccounts := make(map[domain.AccountID]struct{})
	for _, entry := range tx.Entries {
		account, ok := r.accounts[entry.AccountID]
		if !ok {
			return fmt.Errorf("account not found: %s", entry.AccountID)
		}
		if account.Status != domain.AccountActive || account.Currency != entry.Currency {
			return fmt.Errorf("account is inactive or currency does not match: %s", entry.AccountID)
		}
		if expected, ok := tx.ExpectedVersions[entry.AccountID]; ok && account.Version != expected {
			return fmt.Errorf("%w: account %s expected %d actual %d", ErrVersionConflict, entry.AccountID, expected, account.Version)
		}
		usedAccounts[entry.AccountID] = struct{}{}
	}
	for id := range usedAccounts {
		account := r.accounts[id]
		account.Version++
		r.accounts[id] = account
	}
	r.idempotency[tx.IdempotencyKey] = tx.ID
	for i := range tx.Entries {
		if tx.Entries[i].CreatedAt.IsZero() {
			tx.Entries[i].CreatedAt = time.Now().UTC()
		}
		e := tx.Entries[i]
		r.entries[e.AccountID] = append(r.entries[e.AccountID], e)
	}
	r.transactions[tx.ID] = tx
	return nil
}

func (r *InMemoryLedger) GetTransaction(ctx context.Context, id domain.TransactionID) (domain.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tx, ok := r.transactions[id]
	if !ok {
		return domain.Transaction{}, fmt.Errorf("transaction not found")
	}
	tx.Entries = append([]domain.Entry(nil), tx.Entries...)
	return tx, nil
}

func (r *InMemoryLedger) GetTransactionByIdempotencyKey(ctx context.Context, key string) (domain.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.idempotency[key]
	if !ok {
		return domain.Transaction{}, fmt.Errorf("transaction not found")
	}
	tx := r.transactions[id]
	tx.Entries = append([]domain.Entry(nil), tx.Entries...)
	return tx, nil
}

func (r *InMemoryLedger) GetBalance(ctx context.Context, accountID domain.AccountID) (decimal.Decimal, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var bal decimal.Decimal
	currency := "USD"
	if account, ok := r.accounts[accountID]; ok {
		currency = account.Currency
	}
	for _, e := range r.entries[accountID] {
		if e.Direction == domain.Debit {
			bal = bal.Add(e.Amount)
		} else {
			bal = bal.Sub(e.Amount)
		}
	}
	return bal, currency, nil
}

func (r *InMemoryLedger) GetEntries(ctx context.Context, accountID domain.AccountID) ([]domain.Entry, error) {
	return r.GetEntriesPage(ctx, accountID, int(^uint(0)>>1), 0)
}

func (r *InMemoryLedger) GetEntriesPage(ctx context.Context, accountID domain.AccountID, limit, offset int) ([]domain.Entry, error) {
	if limit <= 0 || offset < 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	entries := append([]domain.Entry(nil), r.entries[accountID]...)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].CreatedAt.Equal(entries[j].CreatedAt) {
			return entries[i].ID.String() < entries[j].ID.String()
		}
		return entries[i].CreatedAt.Before(entries[j].CreatedAt)
	})
	if offset >= len(entries) {
		return []domain.Entry{}, nil
	}
	end := offset + limit
	if end > len(entries) {
		end = len(entries)
	}
	return entries[offset:end], nil
}

func (r *InMemoryLedger) GetEntriesAfter(ctx context.Context, accountID domain.AccountID, limit int, createdAt time.Time, entryID uuid.UUID) ([]domain.Entry, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid pagination")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	entries := append([]domain.Entry(nil), r.entries[accountID]...)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].CreatedAt.Equal(entries[j].CreatedAt) {
			return entries[i].ID.String() < entries[j].ID.String()
		}
		return entries[i].CreatedAt.Before(entries[j].CreatedAt)
	})
	out := make([]domain.Entry, 0, limit)
	for _, entry := range entries {
		if entry.CreatedAt.Before(createdAt) || (entry.CreatedAt.Equal(createdAt) && entry.ID.String() <= entryID.String()) {
			continue
		}
		out = append(out, entry)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (r *InMemoryLedger) GetBalanceAsOf(ctx context.Context, accountID domain.AccountID, asOf time.Time) (decimal.Decimal, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	account, ok := r.accounts[accountID]
	currency := "USD"
	if ok {
		currency = account.Currency
	}
	var balance decimal.Decimal
	for _, e := range r.entries[accountID] {
		if e.CreatedAt.After(asOf) {
			continue
		}
		if e.Direction == domain.Debit {
			balance = balance.Add(e.Amount)
		} else {
			balance = balance.Sub(e.Amount)
		}
	}
	return balance, currency, nil
}

func (r *InMemoryLedger) GetTrialBalance(ctx context.Context) ([]AccountBalance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]AccountBalance, 0, len(r.accounts))
	for id, account := range r.accounts {
		var balance decimal.Decimal
		for _, e := range r.entries[id] {
			if e.Direction == domain.Debit {
				balance = balance.Add(e.Amount)
			} else {
				balance = balance.Sub(e.Amount)
			}
		}
		out = append(out, AccountBalance{AccountID: id, Currency: account.Currency, Balance: balance})
	}
	return out, nil
}
