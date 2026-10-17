package repository

import (
	"context"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/shopspring/decimal"
)

type AccountRepository interface {
	CreateAccount(ctx context.Context, account domain.Account) error
	GetAccount(ctx context.Context, id domain.AccountID) (domain.Account, error)
}

type AccountBalance struct {
	AccountID domain.AccountID
	Currency  string
	Balance   decimal.Decimal
}

type LedgerRepository interface {
	SaveTransaction(ctx context.Context, tx domain.Transaction) error
	GetTransaction(ctx context.Context, id domain.TransactionID) (domain.Transaction, error)
	GetTransactionByIdempotencyKey(ctx context.Context, key string) (domain.Transaction, error)
	GetBalance(ctx context.Context, accountID domain.AccountID) (decimal.Decimal, string, error)
	GetEntries(ctx context.Context, accountID domain.AccountID) ([]domain.Entry, error)
	GetEntriesPage(ctx context.Context, accountID domain.AccountID, limit, offset int) ([]domain.Entry, error)
	GetBalanceAsOf(ctx context.Context, accountID domain.AccountID, asOf time.Time) (decimal.Decimal, string, error)
	GetTrialBalance(ctx context.Context) ([]AccountBalance, error)
}
