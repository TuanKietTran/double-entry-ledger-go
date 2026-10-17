package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestPostgresLedger(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("embedded Postgres cannot initdb as root")
	}
	dsn := startPostgres(t, "ledger", 55432)
	ctx := context.Background()

	repo, err := NewPostgresLedger(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(repo.Close)

	sqlBytes, err := os.ReadFile(filepath.Join("..", "..", "migrations", "001_init.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.pool.Exec(ctx, string(sqlBytes)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	reversalMigration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "002_reversals.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.pool.Exec(ctx, string(reversalMigration)); err != nil {
		t.Fatalf("reversal migrate: %v", err)
	}

	t.Run("balance and entries", func(t *testing.T) {
		cash := uuid.New()
		revenue := uuid.New()
		for _, id := range []uuid.UUID{cash, revenue} {
			if err := repo.CreateAccount(ctx, domain.Account{ID: id, OwnerID: "fixture", Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
				t.Fatal(err)
			}
		}
		tx := domain.Transaction{
			ID:             uuid.New(),
			IdempotencyKey: "pg-fund-1",
			Description:    "open cash",
			Status:         "POSTED",
			Entries: []domain.Entry{
				{ID: uuid.New(), AccountID: cash, Amount: decimal.NewFromInt(100), Direction: domain.Debit, Currency: "USD"},
				{ID: uuid.New(), AccountID: revenue, Amount: decimal.NewFromInt(100), Direction: domain.Credit, Currency: "USD"},
			},
		}
		tx.Entries[0].TransactionID = tx.ID
		tx.Entries[1].TransactionID = tx.ID
		if err := repo.SaveTransaction(ctx, tx); err != nil {
			t.Fatal(err)
		}
		lookup, err := repo.GetTransaction(ctx, tx.ID)
		if err != nil || lookup.IdempotencyKey != tx.IdempotencyKey || len(lookup.Entries) != 2 {
			t.Fatalf("transaction lookup = %+v, %v", lookup, err)
		}
		byKey, err := repo.GetTransactionByIdempotencyKey(ctx, tx.IdempotencyKey)
		if err != nil || byKey.ID != tx.ID {
			t.Fatalf("key lookup = %+v, %v", byKey, err)
		}

		partial := domain.Transaction{
			ID:             uuid.New(),
			IdempotencyKey: "pg-fee-1",
			Description:    "partial reversal",
			Status:         "POSTED",
			Entries: []domain.Entry{
				{ID: uuid.New(), TransactionID: uuid.Nil, AccountID: revenue, Amount: decimal.RequireFromString("40.25"), Direction: domain.Debit, Currency: "USD"},
				{ID: uuid.New(), AccountID: cash, Amount: decimal.RequireFromString("40.25"), Direction: domain.Credit, Currency: "USD"},
			},
		}
		partial.Entries[0].TransactionID = partial.ID
		partial.Entries[1].TransactionID = partial.ID
		if err := repo.SaveTransaction(ctx, partial); err != nil {
			t.Fatal(err)
		}
		page, err := repo.GetEntriesPage(ctx, cash, 1, 0)
		if err != nil || len(page) != 1 {
			t.Fatalf("entry page = %+v, %v", page, err)
		}
		asOf, asOfCurrency, err := repo.GetBalanceAsOf(ctx, cash, time.Now().Add(time.Hour))
		if err != nil || asOfCurrency != "USD" || !asOf.Equal(decimal.RequireFromString("59.75")) {
			t.Fatalf("as-of balance = %s %s, %v", asOf, asOfCurrency, err)
		}
		trial, err := repo.GetTrialBalance(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var total decimal.Decimal
		for _, row := range trial {
			total = total.Add(row.Balance)
		}
		if !total.IsZero() {
			t.Fatalf("trial balance total = %s", total)
		}

		bal, cur, err := repo.GetBalance(ctx, cash)
		if err != nil {
			t.Fatal(err)
		}
		if !bal.Equal(decimal.RequireFromString("59.75")) || cur != "USD" {
			t.Fatalf("cash balance = %s %s", bal, cur)
		}
		entries, err := repo.GetEntries(ctx, cash)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 2 {
			t.Fatalf("entries = %d, want 2", len(entries))
		}
		if entries[0].Direction != domain.Debit || entries[1].Direction != domain.Credit {
			t.Fatalf("entry order/direction = %+v", entries)
		}
	})

	t.Run("idempotency does not double post", func(t *testing.T) {
		account := uuid.New()
		other := uuid.New()
		for _, id := range []uuid.UUID{account, other} {
			if err := repo.CreateAccount(ctx, domain.Account{ID: id, OwnerID: "fixture", Currency: "EUR", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
				t.Fatal(err)
			}
		}
		tx := domain.Transaction{
			ID:             uuid.New(),
			IdempotencyKey: "pg-same-key",
			Status:         "POSTED",
			Entries: []domain.Entry{
				{ID: uuid.New(), AccountID: account, Amount: decimal.NewFromInt(5), Direction: domain.Debit, Currency: "EUR"},
				{ID: uuid.New(), AccountID: other, Amount: decimal.NewFromInt(5), Direction: domain.Credit, Currency: "EUR"},
			},
		}
		tx.Entries[0].TransactionID = tx.ID
		tx.Entries[1].TransactionID = tx.ID
		if err := repo.SaveTransaction(ctx, tx); err != nil {
			t.Fatal(err)
		}
		dup := tx
		dup.ID = uuid.New()
		dup.Entries = []domain.Entry{
			{ID: uuid.New(), TransactionID: dup.ID, AccountID: account, Amount: decimal.NewFromInt(5), Direction: domain.Debit, Currency: "EUR"},
			{ID: uuid.New(), TransactionID: dup.ID, AccountID: other, Amount: decimal.NewFromInt(5), Direction: domain.Credit, Currency: "EUR"},
		}
		if err := repo.SaveTransaction(ctx, dup); err != nil {
			t.Fatalf("same-payload replay: %v", err)
		}
		changed := dup
		changed.Description = "different payload"
		if err := repo.SaveTransaction(ctx, changed); !errors.Is(err, ErrIdempotencyConflict) {
			t.Fatalf("changed-payload replay error = %v, want conflict", err)
		}
		entries, err := repo.GetEntries(ctx, account)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			t.Fatalf("entries = %d, want 1", len(entries))
		}
		bal, cur, err := repo.GetBalance(ctx, account)
		if err != nil {
			t.Fatal(err)
		}
		if !bal.Equal(decimal.NewFromInt(5)) || cur != "EUR" {
			t.Fatalf("balance = %s %s", bal, cur)
		}
	})

	t.Run("reversal is append-only and unique", func(t *testing.T) {
		a, b := uuid.New(), uuid.New()
		for _, id := range []uuid.UUID{a, b} {
			if err := repo.CreateAccount(ctx, domain.Account{ID: id, OwnerID: "fixture", Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
				t.Fatal(err)
			}
		}
		original := domain.Transaction{ID: uuid.New(), IdempotencyKey: "reversal-original", Status: "POSTED", Entries: []domain.Entry{
			{ID: uuid.New(), AccountID: a, Amount: decimal.NewFromInt(1), Direction: domain.Debit, Currency: "USD"},
			{ID: uuid.New(), AccountID: b, Amount: decimal.NewFromInt(1), Direction: domain.Credit, Currency: "USD"},
		}}
		for i := range original.Entries {
			original.Entries[i].TransactionID = original.ID
		}
		if err := repo.SaveTransaction(ctx, original); err != nil {
			t.Fatal(err)
		}
		reversal := domain.Transaction{ID: uuid.New(), ReversalOf: original.ID, IdempotencyKey: "reversal-entry", Status: "POSTED", Entries: []domain.Entry{
			{ID: uuid.New(), AccountID: a, Amount: decimal.NewFromInt(1), Direction: domain.Credit, Currency: "USD"},
			{ID: uuid.New(), AccountID: b, Amount: decimal.NewFromInt(1), Direction: domain.Debit, Currency: "USD"},
		}}
		for i := range reversal.Entries {
			reversal.Entries[i].TransactionID = reversal.ID
		}
		if err := repo.SaveReversal(ctx, original.ID, reversal); err != nil {
			t.Fatal(err)
		}
		if err := repo.SaveReversal(ctx, original.ID, reversal); err != nil {
			t.Fatalf("identical reversal replay: %v", err)
		}
		duplicate := reversal
		duplicate.ID, duplicate.IdempotencyKey = uuid.New(), "second-reversal"
		for i := range duplicate.Entries {
			duplicate.Entries[i].ID = uuid.New()
			duplicate.Entries[i].TransactionID = duplicate.ID
		}
		if err := repo.SaveReversal(ctx, original.ID, duplicate); err == nil {
			t.Fatal("second reversal unexpectedly accepted")
		}
		stored, err := repo.GetTransaction(ctx, reversal.ID)
		if err != nil || stored.ReversalOf != original.ID {
			t.Fatalf("reversal lookup=%+v err=%v", stored, err)
		}
	})

	t.Run("explicit account provisioning is idempotent and immutable", func(t *testing.T) {
		account := domain.Account{ID: uuid.New(), OwnerID: "fixture", Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}
		if err := repo.CreateAccount(ctx, account); err != nil {
			t.Fatal(err)
		}
		if err := repo.CreateAccount(ctx, account); err != nil {
			t.Fatalf("same account retry: %v", err)
		}
		got, err := repo.GetAccount(ctx, account.ID)
		if err != nil || got != account {
			t.Fatalf("GetAccount() = %+v, %v; want %+v", got, err, account)
		}
		changed := account
		changed.Currency = "EUR"
		if err := repo.CreateAccount(ctx, changed); err == nil {
			t.Fatal("changed account attributes unexpectedly accepted")
		}
	})

	t.Run("posting does not auto-create an account", func(t *testing.T) {
		account, other := uuid.New(), uuid.New()
		tx := domain.Transaction{ID: uuid.New(), IdempotencyKey: "missing-account", Status: "POSTED", Entries: []domain.Entry{
			{ID: uuid.New(), AccountID: account, Amount: decimal.NewFromInt(1), Direction: domain.Debit, Currency: "USD"},
			{ID: uuid.New(), AccountID: other, Amount: decimal.NewFromInt(1), Direction: domain.Credit, Currency: "USD"},
		}}
		tx.Entries[0].TransactionID, tx.Entries[1].TransactionID = tx.ID, tx.ID
		if err := repo.SaveTransaction(ctx, tx); err == nil {
			t.Fatal("post with unprovisioned accounts unexpectedly succeeded")
		}
		if _, err := repo.GetAccount(ctx, account); err == nil {
			t.Fatal("post implicitly created an account")
		}
		entries, err := repo.GetEntries(ctx, account)
		if err != nil || len(entries) != 0 {
			t.Fatalf("unexpected entries after rejected post: %v %v", entries, err)
		}
	})

	t.Run("unknown account is zero usd", func(t *testing.T) {
		bal, cur, err := repo.GetBalance(ctx, uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		if !bal.Equal(decimal.Zero) || cur != "USD" {
			t.Fatalf("balance = %s %s", bal, cur)
		}
		entries, err := repo.GetEntries(ctx, uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatalf("entries = %d", len(entries))
		}
	})
}

func startPostgres(t *testing.T, dbName string, port uint32) string {
	t.Helper()
	cfg := embeddedpostgres.DefaultConfig().
		Username("postgres").
		Password("postgres").
		Database(dbName).
		Port(port).
		Version(embeddedpostgres.V16).
		CachePath("/tmp/embedded-pg-cache").
		RuntimePath(filepath.Join(t.TempDir(), "rt")).
		DataPath(filepath.Join(t.TempDir(), "data")).
		StartTimeout(90 * time.Second).
		Logger(os.Stdout)
	ep := embeddedpostgres.NewDatabase(cfg)
	if err := ep.Start(); err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = ep.Stop() })
	return fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/%s?sslmode=disable", port, dbName)
}
