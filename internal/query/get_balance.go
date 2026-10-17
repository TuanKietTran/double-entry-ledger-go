package query

import (
	"context"

	"github.com/example/double-entry-ledger/internal/domain"
	"github.com/example/double-entry-ledger/internal/repository"
	"github.com/shopspring/decimal"
)

type GetBalanceQuery struct {
	AccountID domain.AccountID
}

type BalanceResult struct {
	AccountID domain.AccountID
	Available decimal.Decimal
	Currency  string
}

type GetBalanceHandler struct {
	repo repository.LedgerRepository
}

func NewGetBalanceHandler(repo repository.LedgerRepository) *GetBalanceHandler {
	return &GetBalanceHandler{repo: repo}
}

func (h *GetBalanceHandler) Handle(ctx context.Context, q GetBalanceQuery) (BalanceResult, error) {
	bal, currency, err := h.repo.GetBalance(ctx, q.AccountID)
	if err != nil {
		return BalanceResult{}, err
	}
	return BalanceResult{
		AccountID: q.AccountID,
		Available: bal,
		Currency:  currency,
	}, nil
}
