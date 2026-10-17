package repository

import (
	"context"
	"testing"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestMemoryPaginationAsOfAndTrialBalance(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryLedger()
	account, offset := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{account, offset} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	firstAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	secondAt := firstAt.Add(time.Hour)
	post := func(key string, at time.Time, amount int64) {
		t.Helper()
		tx := domain.Transaction{ID: uuid.New(), IdempotencyKey: key, Status: "POSTED", Entries: []domain.Entry{
			{ID: uuid.New(), AccountID: account, Amount: decimal.NewFromInt(amount), Direction: domain.Debit, Currency: "USD", CreatedAt: at},
			{ID: uuid.New(), AccountID: offset, Amount: decimal.NewFromInt(amount), Direction: domain.Credit, Currency: "USD", CreatedAt: at},
		}}
		for i := range tx.Entries {
			tx.Entries[i].TransactionID = tx.ID
		}
		if err := repo.SaveTransaction(ctx, tx); err != nil {
			t.Fatal(err)
		}
	}
	post("first", firstAt, 4)
	post("second", secondAt, 6)
	page, err := repo.GetEntriesPage(ctx, account, 1, 0)
	if err != nil || len(page) != 1 || page[0].Amount.String() != "4" {
		t.Fatalf("first page = %+v, %v", page, err)
	}
	beforeSecond, currency, err := repo.GetBalanceAsOf(ctx, account, firstAt)
	if err != nil || currency != "USD" || !beforeSecond.Equal(decimal.NewFromInt(4)) {
		t.Fatalf("as-of = %s %s, %v", beforeSecond, currency, err)
	}
	trial, err := repo.GetTrialBalance(ctx)
	if err != nil || len(trial) != 2 {
		t.Fatalf("trial balance = %+v, %v", trial, err)
	}
	var total decimal.Decimal
	for _, row := range trial {
		total = total.Add(row.Balance)
	}
	if !total.IsZero() {
		t.Fatalf("trial balance does not sum to zero: %s", total)
	}
}
