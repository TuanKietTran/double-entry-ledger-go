package repository

import (
	"context"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestMemoryTransactionLookup(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	tx := domain.Transaction{ID: uuid.New(), IdempotencyKey: "lookup-key", Status: "POSTED", Entries: []domain.Entry{
		{ID: uuid.New(), AccountID: a, Amount: decimal.NewFromInt(2), Direction: domain.Debit, Currency: "USD"},
		{ID: uuid.New(), AccountID: b, Amount: decimal.NewFromInt(2), Direction: domain.Credit, Currency: "USD"},
	}}
	for i := range tx.Entries {
		tx.Entries[i].TransactionID = tx.ID
	}
	if err := repo.SaveTransaction(ctx, tx); err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string]domain.Transaction{
		"identity":        mustGetTransaction(t, func() (domain.Transaction, error) { return repo.GetTransaction(ctx, tx.ID) }),
		"idempotency key": mustGetTransaction(t, func() (domain.Transaction, error) { return repo.GetTransactionByIdempotencyKey(ctx, tx.IdempotencyKey) }),
	} {
		if got.ID != tx.ID || got.IdempotencyKey != tx.IdempotencyKey || len(got.Entries) != 2 {
			t.Errorf("%s lookup = %+v", name, got)
		}
	}
}

func mustGetTransaction(t *testing.T, get func() (domain.Transaction, error)) domain.Transaction {
	t.Helper()
	tx, err := get()
	if err != nil {
		t.Fatal(err)
	}
	return tx
}
