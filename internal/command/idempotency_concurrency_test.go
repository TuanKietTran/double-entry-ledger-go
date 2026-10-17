package command

import (
	"context"
	"sync"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestConcurrentSamePayloadReturnsOneTransaction(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	h := NewPostTransactionHandler(repo)
	cmd := PostTransactionCommand{IdempotencyKey: "concurrent-same", Entries: []EntryInput{
		{AccountID: a, Amount: decimal.NewFromInt(12), Direction: domain.Debit, Currency: "USD"},
		{AccountID: b, Amount: decimal.NewFromInt(12), Direction: domain.Credit, Currency: "USD"},
	}}
	const workers = 16
	var wg sync.WaitGroup
	ids := make(chan uuid.UUID, workers)
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := h.Handle(ctx, cmd)
			if err != nil {
				errs <- err
				return
			}
			ids <- id
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	var first uuid.UUID
	for id := range ids {
		if first == uuid.Nil {
			first = id
		} else if first != id {
			t.Errorf("replay id %s differs from %s", id, first)
		}
	}
	entries, err := repo.GetEntries(ctx, a)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%d err=%v", len(entries), err)
	}
	balance, err := query.NewGetBalanceHandler(repo).Handle(ctx, query.GetBalanceQuery{AccountID: a})
	if err != nil || !balance.Available.Equal(decimal.NewFromInt(12)) {
		t.Fatalf("balance=%s err=%v", balance.Available, err)
	}
}
