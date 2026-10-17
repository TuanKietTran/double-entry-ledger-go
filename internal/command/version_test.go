package command

import (
	"context"
	"errors"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestExpectedAccountVersionsRejectStalePost(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	h := NewPostTransactionHandler(repo)
	post := func(key string, version int64) error {
		_, err := h.Handle(ctx, PostTransactionCommand{IdempotencyKey: key, ExpectedVersions: map[domain.AccountID]int64{a: version, b: version}, Entries: []EntryInput{
			{AccountID: a, Amount: decimal.NewFromInt(1), Direction: domain.Debit, Currency: "USD"},
			{AccountID: b, Amount: decimal.NewFromInt(1), Direction: domain.Credit, Currency: "USD"},
		}})
		return err
	}
	if err := post("v0", 0); err != nil {
		t.Fatal(err)
	}
	if err := post("stale", 0); !errors.Is(err, repository.ErrVersionConflict) {
		t.Fatalf("stale version err=%v, want conflict", err)
	}
	if err := post("v1", 1); err != nil {
		t.Fatalf("current version: %v", err)
	}
	got, err := repo.GetAccount(ctx, a)
	if err != nil || got.Version != 2 {
		t.Fatalf("account version=%d err=%v, want 2", got.Version, err)
	}
}
