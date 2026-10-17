package query

import (
	"context"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/shopspring/decimal"
)

type GetBalanceQuery struct {
	AccountID domain.AccountID
}

type BalanceResult struct {
	AccountID domain.AccountID `json:"account_id"`
	Available decimal.Decimal  `json:"available"`
	Currency  string           `json:"currency"`
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
