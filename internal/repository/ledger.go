package repository

import (
	"context"

	"github.com/example/double-entry-ledger/internal/domain"
	"github.com/shopspring/decimal"
)

type LedgerRepository interface {
	SaveTransaction(ctx context.Context, tx domain.Transaction) error
	GetBalance(ctx context.Context, accountID domain.AccountID) (decimal.Decimal, string, error)
	GetEntries(ctx context.Context, accountID domain.AccountID) ([]domain.Entry, error)
}
