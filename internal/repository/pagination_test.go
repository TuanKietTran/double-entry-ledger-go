package repository

import (
	"context"
	"testing"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestMemoryPaginationIsStableWithTimestampTies(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryLedger()
	account, offset := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{account, offset} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		tx := domain.Transaction{ID: uuid.New(), IdempotencyKey: uuid.NewString(), Status: "POSTED", Entries: []domain.Entry{
			{ID: uuid.New(), AccountID: account, Amount: decimal.NewFromInt(1), Direction: domain.Debit, Currency: "USD", CreatedAt: at},
			{ID: uuid.New(), AccountID: offset, Amount: decimal.NewFromInt(1), Direction: domain.Credit, Currency: "USD", CreatedAt: at},
		}}
		for j := range tx.Entries {
			tx.Entries[j].TransactionID = tx.ID
		}
		if err := repo.SaveTransaction(ctx, tx); err != nil {
			t.Fatal(err)
		}
	}
	all, err := repo.GetEntriesPage(ctx, account, 100, 0)
	if err != nil || len(all) != 5 {
		t.Fatalf("all = %d, %v", len(all), err)
	}
	var paged []domain.Entry
	for off := 0; off < 6; off += 2 {
		page, err := repo.GetEntriesPage(ctx, account, 2, off)
		if err != nil {
			t.Fatal(err)
		}
		paged = append(paged, page...)
	}
	if len(paged) != len(all) {
		t.Fatalf("paged %d entries, want %d", len(paged), len(all))
	}
	for i := range all {
		if paged[i].ID != all[i].ID {
			t.Fatalf("page order differs at %d", i)
		}
	}
	if page, err := repo.GetEntriesPage(ctx, account, 2, 99); err != nil || len(page) != 0 {
		t.Fatalf("offset past end = %d, %v", len(page), err)
	}
	if _, err := repo.GetEntriesPage(ctx, account, 0, 0); err == nil {
		t.Fatal("limit 0 must be rejected")
	}
	if _, err := repo.GetEntriesPage(ctx, account, 1, -1); err == nil {
		t.Fatal("negative offset must be rejected")
	}
}
