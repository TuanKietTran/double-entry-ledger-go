package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestReverseTransactionRoute(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	original, err := command.NewPostTransactionHandler(repo).Handle(ctx, command.PostTransactionCommand{IdempotencyKey: "api-original", Entries: []command.EntryInput{
		{AccountID: a, Amount: decimal.NewFromInt(1), Direction: domain.Debit, Currency: "USD"},
		{AccountID: b, Amount: decimal.NewFromInt(1), Direction: domain.Credit, Currency: "USD"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandlersWithLedger(nil, nil, nil, query.NewGetTransactionHandler(repo), query.NewLedgerQueries(repo), command.NewReverseTransactionHandler(repo))
	request := func() *httptest.ResponseRecorder {
		res := httptest.NewRecorder()
		h.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/transactions/"+original.String()+"/reversals", strings.NewReader(`{"idempotency_key":"api-reversal"}`)))
		return res
	}
	first, replay := request(), request()
	if first.Code != http.StatusOK || replay.Code != http.StatusOK || first.Body.String() != replay.Body.String() {
		t.Fatalf("first=%d %s replay=%d %s", first.Code, first.Body.String(), replay.Code, replay.Body.String())
	}
}
