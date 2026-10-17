package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestMemoryRepositoryPostsAndReversesDirectly(t *testing.T) {
	ctx := context.Background()
	r := NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := r.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	tx := domain.Transaction{ID: uuid.New(), IdempotencyKey: "repo-direct", Description: "direct", Status: "POSTED", Entries: []domain.Entry{
		{ID: uuid.New(), AccountID: a, Amount: decimal.NewFromInt(5), Direction: domain.Debit, Currency: "USD"},
		{ID: uuid.New(), AccountID: b, Amount: decimal.NewFromInt(5), Direction: domain.Credit, Currency: "USD"},
	}}
	for i := range tx.Entries {
		tx.Entries[i].TransactionID = tx.ID
	}
	if err := r.SaveTransaction(ctx, tx); err != nil {
		t.Fatal(err)
	}
	balance, currency, err := r.GetBalance(ctx, a)
	if err != nil || currency != "USD" || !balance.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("balance=%s %s err=%v", balance, currency, err)
	}
	entries, err := r.GetEntries(ctx, a)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	reversal := domain.Transaction{ID: uuid.New(), ReversalOf: tx.ID, IdempotencyKey: "repo-reverse", Description: "undo", Status: "POSTED", Entries: []domain.Entry{
		{ID: uuid.New(), AccountID: a, Amount: decimal.NewFromInt(5), Direction: domain.Credit, Currency: "USD"},
		{ID: uuid.New(), AccountID: b, Amount: decimal.NewFromInt(5), Direction: domain.Debit, Currency: "USD"},
	}}
	for i := range reversal.Entries {
		reversal.Entries[i].TransactionID = reversal.ID
	}
	if err := r.SaveReversal(ctx, tx.ID, reversal); err != nil {
		t.Fatal(err)
	}
	if err := r.SaveReversal(ctx, tx.ID, reversal); err != nil {
		t.Fatalf("replay=%v", err)
	}
	if err := r.SaveTransaction(ctx, reversal); err == nil {
		t.Fatal("normal post accepted reversal metadata")
	}
	second := reversal
	second.ID = uuid.New()
	second.IdempotencyKey = "repo-reverse-second"
	for i := range second.Entries {
		second.Entries[i].ID = uuid.New()
		second.Entries[i].TransactionID = second.ID
	}
	if err := r.SaveReversal(ctx, tx.ID, second); err == nil {
		t.Fatal("second reversal accepted")
	}
	balance, _, err = r.GetBalance(ctx, a)
	if err != nil || !balance.IsZero() {
		t.Fatalf("reversed balance=%s err=%v", balance, err)
	}
	if err := r.SaveReversal(ctx, uuid.New(), second); err == nil {
		t.Fatal("unknown original accepted")
	}
	if err := r.SaveReversal(ctx, tx.ID, domain.Transaction{ID: uuid.New(), IdempotencyKey: "bad"}); err == nil {
		t.Fatal("mismatched reversal reference accepted")
	}
	changed := tx
	changed.ID = uuid.New()
	changed.Description = "changed"
	if err := r.SaveTransaction(ctx, changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed idempotency payload error=%v", err)
	}
}
