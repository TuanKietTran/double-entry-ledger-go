package api

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestLedgerQueryRoutes(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := command.NewPostTransactionHandler(repo).Handle(ctx, command.PostTransactionCommand{IdempotencyKey: "query-http", Entries: []command.EntryInput{
		{AccountID: a, Amount: decimal.NewFromInt(2), Direction: domain.Debit, Currency: "USD"},
		{AccountID: b, Amount: decimal.NewFromInt(2), Direction: domain.Credit, Currency: "USD"},
	}}); err != nil {
		t.Fatal(err)
	}
	h := NewHandlersWithTransactionQueries(nil, query.NewGetBalanceHandler(repo), nil, nil, query.NewLedgerQueries(repo))
	at := time.Now().UTC().Add(time.Minute).Format(time.RFC3339Nano)
	for _, path := range []string{"/accounts/" + a.String() + "/entries?limit=1&offset=0", "/accounts/" + a.String() + "/balance?as_of=" + at, "/trial-balance"} {
		res := httptest.NewRecorder()
		h.Routes().ServeHTTP(res, httptest.NewRequest("GET", path, nil))
		if res.Code != 200 {
			t.Errorf("GET %s = %d %s", path, res.Code, res.Body.String())
		}
	}
}
