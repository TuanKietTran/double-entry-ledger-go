package query

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type failingRepo struct{ repository.LedgerRepository }

func (f failingRepo) GetBalance(context.Context, domain.AccountID) (decimal.Decimal, string, error) {
	return decimal.Zero, "", errors.New("balance failed")
}
func (f failingRepo) GetBalanceAsOf(context.Context, domain.AccountID, time.Time) (decimal.Decimal, string, error) {
	return decimal.Zero, "", errors.New("as-of failed")
}
func (f failingRepo) GetEntriesPage(context.Context, domain.AccountID, int, int) ([]domain.Entry, error) {
	return nil, errors.New("entries failed")
}
func (f failingRepo) GetTrialBalance(context.Context) ([]repository.AccountBalance, error) {
	return nil, errors.New("trial failed")
}
func (f failingRepo) GetTransaction(context.Context, domain.TransactionID) (domain.Transaction, error) {
	return domain.Transaction{}, errors.New("transaction failed")
}
func (f failingRepo) GetTransactionByIdempotencyKey(context.Context, string) (domain.Transaction, error) {
	return domain.Transaction{}, errors.New("key failed")
}

func TestQueryHandlersSuccessAndErrorPaths(t *testing.T) {
	ctx := context.Background()
	mem := repository.NewInMemoryLedger()
	balance, err := NewGetBalanceHandler(mem).Handle(ctx, GetBalanceQuery{AccountID: uuid.New()})
	if err != nil || !balance.Available.IsZero() {
		t.Fatalf("balance=%+v err=%v", balance, err)
	}
	failing := failingRepo{LedgerRepository: mem}
	if _, err := NewGetBalanceHandler(failing).Handle(ctx, GetBalanceQuery{}); err == nil {
		t.Fatal("balance error lost")
	}
	queries := NewLedgerQueries(mem)
	if _, err := queries.Account(ctx, uuid.New()); err == nil {
		t.Fatal("missing account accepted")
	}
	if _, err := queries.BalanceAsOf(ctx, uuid.New(), time.Now()); err != nil {
		t.Fatalf("memory zero as-of: %v", err)
	}
	if _, err := NewLedgerQueries(failing).BalanceAsOf(ctx, uuid.New(), time.Now()); err == nil {
		t.Fatal("as-of error lost")
	}
	if _, err := NewLedgerQueries(failing).Entries(ctx, uuid.New(), 10, 0); err == nil {
		t.Fatal("entries error lost")
	}
	if _, err := NewLedgerQueries(failing).TrialBalance(ctx); err == nil {
		t.Fatal("trial error lost")
	}
	txQueries := NewGetTransactionHandler(failing)
	if _, err := txQueries.ByID(ctx, uuid.New()); err == nil {
		t.Fatal("transaction error lost")
	}
	if _, err := txQueries.ByIdempotencyKey(ctx, "missing"); err == nil {
		t.Fatal("key error lost")
	}
}
