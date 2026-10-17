package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestTransactionLookupRoutes(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	id, err := command.NewPostTransactionHandler(repo).Handle(ctx, command.PostTransactionCommand{IdempotencyKey: "lookup-api", Entries: []command.EntryInput{
		{AccountID: a, Amount: decimal.NewFromInt(3), Direction: domain.Debit, Currency: "USD"},
		{AccountID: b, Amount: decimal.NewFromInt(3), Direction: domain.Credit, Currency: "USD"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandlersWithTransactionQueries(nil, nil, nil, query.NewGetTransactionHandler(repo))
	for _, path := range []string{"/transactions/" + id.String(), "/transactions/by-idempotency-key/lookup-api"} {
		req := httptest.NewRequest("GET", path, nil)
		res := httptest.NewRecorder()
		h.Routes().ServeHTTP(res, req)
		if res.Code != 200 {
			t.Fatalf("GET %s = %d %s", path, res.Code, res.Body.String())
		}
		var response struct {
			TransactionID string `json:"transaction_id"`
			Entries       []struct {
				Amount string `json:"amount"`
			} `json:"entries"`
		}
		if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.TransactionID != id.String() || len(response.Entries) != 2 || response.Entries[0].Amount != "3" {
			t.Fatalf("lookup response = %+v", response)
		}
	}
}
