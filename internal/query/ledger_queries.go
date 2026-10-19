package query

import (
	"context"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
)

type LedgerQueries struct{ repo repository.LedgerRepository }

func NewLedgerQueries(repo repository.LedgerRepository) *LedgerQueries {
	return &LedgerQueries{repo: repo}
}
func (h *LedgerQueries) Account(ctx context.Context, id domain.AccountID) (domain.Account, error) {
	return h.repo.GetAccount(ctx, id)
}

func (h *LedgerQueries) Entries(ctx context.Context, id domain.AccountID, limit, offset int) ([]domain.Entry, error) {
	return h.repo.GetEntriesPage(ctx, id, limit, offset)
}
func (h *LedgerQueries) EntriesAfter(ctx context.Context, id domain.AccountID, limit int, createdAt time.Time, entryID uuid.UUID) ([]domain.Entry, error) {
	return h.repo.GetEntriesAfter(ctx, id, limit, createdAt, entryID)
}

func (h *LedgerQueries) BalanceAsOf(ctx context.Context, id domain.AccountID, at time.Time) (BalanceResult, error) {
	balance, currency, err := h.repo.GetBalanceAsOf(ctx, id, at)
	if err != nil {
		return BalanceResult{}, err
	}
	return BalanceResult{AccountID: id, Available: balance, Currency: currency}, nil
}
func (h *LedgerQueries) TrialBalance(ctx context.Context) ([]repository.AccountBalance, error) {
	return h.repo.GetTrialBalance(ctx)
}
