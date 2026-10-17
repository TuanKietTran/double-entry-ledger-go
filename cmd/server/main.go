package main

import (
	"context"
	"log"
	"net/http"

	"github.com/example/double-entry-ledger/internal/api"
	"github.com/example/double-entry-ledger/internal/command"
	"github.com/example/double-entry-ledger/internal/domain"
	"github.com/example/double-entry-ledger/internal/query"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/shopspring/decimal"
)

type memRepo struct{}

func (m *memRepo) SaveTransaction(ctx context.Context, tx domain.Transaction) error {
	return nil
}
func (m *memRepo) GetBalance(ctx context.Context, id domain.AccountID) (decimal.Decimal, string, error) {
	return decimal.NewFromInt(1000), "USD", nil
}
func (m *memRepo) GetEntries(ctx context.Context, id domain.AccountID) ([]domain.Entry, error) {
	return nil, nil
}

func main() {
	repo := &memRepo{}
	postH := command.NewPostTransactionHandler(repo)
	balH := query.NewGetBalanceHandler(repo)
	handlers := api.NewHandlers(postH, balH)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Mount("/v1", handlers.Routes())

	log.Println("Ledger API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
