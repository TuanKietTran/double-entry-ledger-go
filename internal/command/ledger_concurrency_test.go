package command

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestConcurrentDistinctPostsPreserveJournalConservation(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	h := NewPostTransactionHandler(repo)
	const n = 32
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := h.Handle(ctx, PostTransactionCommand{IdempotencyKey: fmt.Sprintf("distinct-%d", i), Entries: []EntryInput{
				{AccountID: a, Amount: decimal.NewFromInt(1), Direction: domain.Debit, Currency: "USD"},
				{AccountID: b, Amount: decimal.NewFromInt(1), Direction: domain.Credit, Currency: "USD"},
			}})
			if err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	for _, id := range []uuid.UUID{a, b} {
		entries, err := repo.GetEntries(ctx, id)
		if err != nil || len(entries) != n {
			t.Fatalf("account %s entries=%d err=%v", id, len(entries), err)
		}
	}
	trial, err := repo.GetTrialBalance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var sum decimal.Decimal
	for _, row := range trial {
		sum = sum.Add(row.Balance)
	}
	if !sum.IsZero() {
		t.Fatalf("debits-credit conservation failed: %s", sum)
	}
}
