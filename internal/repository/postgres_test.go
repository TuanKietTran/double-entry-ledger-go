package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	t.Run("balance and entries", func(t *testing.T) {
		cash := uuid.New()
		revenue := uuid.New()
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
		err := repo.SaveTransaction(ctx, dup)
		if err == nil || !strings.Contains(err.Error(), "idempotency key already used") || !strings.Contains(err.Error(), tx.ID.String()) {
			t.Fatalf("err = %v", err)
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
