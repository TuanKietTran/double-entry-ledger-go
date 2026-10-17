package command

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestConcurrentReversalsAllowExactlyOneEffect(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	originalID, err := NewPostTransactionHandler(repo).Handle(ctx, PostTransactionCommand{IdempotencyKey: "race-original", Entries: []EntryInput{
		{AccountID: a, Amount: decimal.NewFromInt(2), Direction: domain.Debit, Currency: "USD"},
		{AccountID: b, Amount: decimal.NewFromInt(2), Direction: domain.Credit, Currency: "USD"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	reverse := NewReverseTransactionHandler(repo)
	const workers = 12
	var wg sync.WaitGroup
	successes := make(chan uuid.UUID, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id, err := reverse.Handle(ctx, originalID, fmt.Sprintf("reverse-race-%d", i))
			if err == nil {
				successes <- id
			}
		}(i)
	}
	wg.Wait()
	close(successes)
	var count int
	for range successes {
		count++
	}
	if count != 1 {
		t.Fatalf("successful reversals=%d, want exactly one", count)
	}
	entries, err := repo.GetEntries(ctx, a)
	if err != nil || len(entries) != 2 {
		t.Fatalf("account entries=%d err=%v", len(entries), err)
	}
}

func TestReverseTransactionIsAppendOnlyAndIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	post := NewPostTransactionHandler(repo)
	originalID, err := post.Handle(ctx, PostTransactionCommand{IdempotencyKey: "original", Description: "payment", Entries: []EntryInput{
		{AccountID: a, Amount: decimal.NewFromInt(5), Direction: domain.Debit, Currency: "USD"},
		{AccountID: b, Amount: decimal.NewFromInt(5), Direction: domain.Credit, Currency: "USD"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	reverse := NewReverseTransactionHandler(repo)
	first, err := reverse.Handle(ctx, originalID, "reverse-key")
	if err != nil {
		t.Fatal(err)
	}
	replay, err := reverse.Handle(ctx, originalID, "reverse-key")
	if err != nil || replay != first {
		t.Fatalf("replay=%s err=%v first=%s", replay, err, first)
	}
	if _, err := reverse.Handle(ctx, originalID, "another-reversal"); err == nil {
		t.Fatal("second reversal accepted")
	}
	tx, err := repo.GetTransaction(ctx, first)
	if err != nil || tx.ReversalOf != originalID || len(tx.Entries) != 2 {
		t.Fatalf("reversal=%+v err=%v", tx, err)
	}
	balance, err := query.NewGetBalanceHandler(repo).Handle(ctx, query.GetBalanceQuery{AccountID: a})
	if err != nil || !balance.Available.IsZero() {
		t.Fatalf("reversed balance=%s err=%v", balance.Available, err)
	}
	entries, err := repo.GetEntries(ctx, a)
	if err != nil || len(entries) != 2 {
		t.Fatalf("entries=%d err=%v", len(entries), err)
	}
}
