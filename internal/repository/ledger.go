package repository

import (
	"context"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/shopspring/decimal"
)

type AccountRepository interface {
	CreateAccount(ctx context.Context, account domain.Account) error
	GetAccount(ctx context.Context, id domain.AccountID) (domain.Account, error)
}

type LedgerRepository interface {
	SaveTransaction(ctx context.Context, tx domain.Transaction) error
	GetBalance(ctx context.Context, accountID domain.AccountID) (decimal.Decimal, string, error)
	GetEntries(ctx context.Context, accountID domain.AccountID) ([]domain.Entry, error)
}
