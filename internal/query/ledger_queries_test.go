package query

import (
	"context"
	"testing"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestLedgerQueriesEntriesAsOfAndTrialBalance(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	at := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	tx := domain.Transaction{ID: uuid.New(), IdempotencyKey: "q", Status: "POSTED", Entries: []domain.Entry{
		{ID: uuid.New(), AccountID: a, Amount: decimal.NewFromInt(7), Direction: domain.Debit, Currency: "USD", CreatedAt: at},
		{ID: uuid.New(), AccountID: b, Amount: decimal.NewFromInt(7), Direction: domain.Credit, Currency: "USD", CreatedAt: at},
	}}
	for i := range tx.Entries {
		tx.Entries[i].TransactionID = tx.ID
	}
	if err := repo.SaveTransaction(ctx, tx); err != nil {
		t.Fatal(err)
	}
	h := NewLedgerQueries(repo)
	entries, err := h.Entries(ctx, a, 1, 0)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	bal, err := h.BalanceAsOf(ctx, a, at)
	if err != nil || !bal.Available.Equal(decimal.NewFromInt(7)) {
		t.Fatalf("balance=%+v err=%v", bal, err)
	}
	trial, err := h.TrialBalance(ctx)
	if err != nil || len(trial) != 2 {
		t.Fatalf("trial=%v err=%v", trial, err)
	}
}
