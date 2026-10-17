package command

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func provisionAccounts(t *testing.T, repo *repository.InMemoryLedger, currency string, ids ...uuid.UUID) {
	t.Helper()
	for _, id := range ids {
		if err := repo.CreateAccount(context.Background(), domain.Account{ID: id, Currency: currency, Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPostTransactionHandler_BalancedPostUpdatesBalances(t *testing.T) {
	repo := repository.NewInMemoryLedger()
	post := NewPostTransactionHandler(repo)
	bal := query.NewGetBalanceHandler(repo)

	cash := uuid.New()
	revenue := uuid.New()
	provisionAccounts(t, repo, "USD", cash, revenue)
	ctx := context.Background()

	first, err := post.Handle(ctx, PostTransactionCommand{
		IdempotencyKey: "txn-fund-1",
		Description:    "open cash",
		Entries: []EntryInput{
			{AccountID: cash, Amount: decimal.NewFromInt(100), Direction: domain.Debit, Currency: "USD"},
			{AccountID: revenue, Amount: decimal.NewFromInt(100), Direction: domain.Credit, Currency: "USD"},
		},
	})
	if err != nil {
		t.Fatalf("first post: %v", err)
	}
	if first == uuid.Nil {
		t.Fatal("expected transaction id")
	}

	_, err = post.Handle(ctx, PostTransactionCommand{
		IdempotencyKey: "txn-fee-1",
		Description:    "partial reversal",
		Entries: []EntryInput{
			{AccountID: revenue, Amount: decimal.RequireFromString("40.25"), Direction: domain.Debit, Currency: "USD"},
			{AccountID: cash, Amount: decimal.RequireFromString("40.25"), Direction: domain.Credit, Currency: "USD"},
		},
	})
	if err != nil {
		t.Fatalf("second post: %v", err)
	}

	cashBal, err := bal.Handle(ctx, query.GetBalanceQuery{AccountID: cash})
	if err != nil {
		t.Fatal(err)
	}
	if !cashBal.Available.Equal(decimal.RequireFromString("59.75")) {
		t.Fatalf("cash balance = %s, want 59.75", cashBal.Available)
	}
	if cashBal.Currency != "USD" || cashBal.AccountID != cash {
		t.Fatalf("unexpected cash result: %+v", cashBal)
	}

	revBal, err := bal.Handle(ctx, query.GetBalanceQuery{AccountID: revenue})
	if err != nil {
		t.Fatal(err)
	}
	if !revBal.Available.Equal(decimal.RequireFromString("-59.75")) {
		t.Fatalf("revenue balance = %s, want -59.75", revBal.Available)
	}

	entries, err := repo.GetEntries(ctx, cash)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("cash entries = %d, want 2", len(entries))
	}
}

func TestPostTransactionHandler_RejectsUnbalanced(t *testing.T) {
	repo := repository.NewInMemoryLedger()
	post := NewPostTransactionHandler(repo)
	bal := query.NewGetBalanceHandler(repo)

	account := uuid.New()
	other := uuid.New()
	provisionAccounts(t, repo, "USD", account, other)
	_, err := post.Handle(context.Background(), PostTransactionCommand{
		IdempotencyKey: "bad-1",
		Entries: []EntryInput{
			{AccountID: account, Amount: decimal.NewFromInt(10), Direction: domain.Debit, Currency: "USD"},
			{AccountID: other, Amount: decimal.NewFromInt(9), Direction: domain.Credit, Currency: "USD"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "not balanced") {
		t.Fatalf("err = %v, want not balanced", err)
	}

	got, err := bal.Handle(context.Background(), query.GetBalanceQuery{AccountID: account})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Available.Equal(decimal.Zero) {
		t.Fatalf("balance changed on rejected post: %s", got.Available)
	}
}

func TestPostTransactionHandler_IdempotencyKey(t *testing.T) {
	repo := repository.NewInMemoryLedger()
	post := NewPostTransactionHandler(repo)
	account := uuid.New()
	other := uuid.New()
	provisionAccounts(t, repo, "EUR", account, other)
	cmd := PostTransactionCommand{
		IdempotencyKey: "same-key",
		Entries: []EntryInput{
			{AccountID: account, Amount: decimal.NewFromInt(5), Direction: domain.Debit, Currency: "EUR"},
			{AccountID: other, Amount: decimal.NewFromInt(5), Direction: domain.Credit, Currency: "EUR"},
		},
	}

	id, err := post.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	_, err = post.Handle(context.Background(), cmd)
	if err == nil || !strings.Contains(err.Error(), "idempotency key already used") {
		t.Fatalf("err = %v, want idempotency conflict", err)
	}
	if !strings.Contains(err.Error(), id.String()) {
		t.Fatalf("err = %v, want original id %s", err, id)
	}

	entries, err := repo.GetEntries(context.Background(), account)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
}

func TestGetBalanceHandler_UnknownAccount(t *testing.T) {
	repo := repository.NewInMemoryLedger()
	bal := query.NewGetBalanceHandler(repo)
	id := uuid.New()

	got, err := bal.Handle(context.Background(), query.GetBalanceQuery{AccountID: id})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Available.Equal(decimal.Zero) || got.Currency != "USD" || got.AccountID != id {
		t.Fatalf("unexpected zero balance: %+v", got)
	}
}

type errRepo struct {
	err error
}

func (e errRepo) SaveTransaction(context.Context, domain.Transaction) error { return e.err }
func (e errRepo) GetTransaction(context.Context, domain.TransactionID) (domain.Transaction, error) {
	return domain.Transaction{}, e.err
}
func (e errRepo) GetTransactionByIdempotencyKey(context.Context, string) (domain.Transaction, error) {
	return domain.Transaction{}, e.err
}
func (e errRepo) GetBalance(context.Context, domain.AccountID) (decimal.Decimal, string, error) {
	return decimal.Zero, "", e.err
}
func (e errRepo) GetEntries(context.Context, domain.AccountID) ([]domain.Entry, error) {
	return nil, e.err
}
func (e errRepo) GetEntriesPage(context.Context, domain.AccountID, int, int) ([]domain.Entry, error) {
	return nil, e.err
}
func (e errRepo) GetBalanceAsOf(context.Context, domain.AccountID, time.Time) (decimal.Decimal, string, error) {
	return decimal.Zero, "", e.err
}
func (e errRepo) GetTrialBalance(context.Context) ([]repository.AccountBalance, error) {
	return nil, e.err
}

func TestHandlers_PropagateRepositoryErrors(t *testing.T) {
	repo := errRepo{err: context.Canceled}
	post := NewPostTransactionHandler(repo)
	_, err := post.Handle(context.Background(), PostTransactionCommand{
		IdempotencyKey: "ok",
		Entries: []EntryInput{
			{AccountID: uuid.New(), Amount: decimal.NewFromInt(1), Direction: domain.Debit, Currency: "USD"},
			{AccountID: uuid.New(), Amount: decimal.NewFromInt(1), Direction: domain.Credit, Currency: "USD"},
		},
	})
	if err != context.Canceled {
		t.Fatalf("post err = %v", err)
	}

	_, err = query.NewGetBalanceHandler(repo).Handle(context.Background(), query.GetBalanceQuery{AccountID: uuid.New()})
	if err != context.Canceled {
		t.Fatalf("balance err = %v", err)
	}
}
