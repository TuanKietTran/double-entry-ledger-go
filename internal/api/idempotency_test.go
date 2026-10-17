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
)

func TestPostTransactionPayloadBoundIdempotency(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryLedger()
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if err := repo.CreateAccount(ctx, domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
			t.Fatal(err)
		}
	}
	h := NewHandlers(command.NewPostTransactionHandler(repo), query.NewGetBalanceHandler(repo))
	body := `{"idempotency_key":"bound","description":"x","entries":[{"account_id":"` + a.String() + `","amount":"1","direction":"DEBIT","currency":"USD"},{"account_id":"` + b.String() + `","amount":"1","direction":"CREDIT","currency":"USD"}]}`
	post := func(payload string) *httptest.ResponseRecorder {
		res := httptest.NewRecorder()
		h.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(payload)))
		return res
	}
	first := post(body)
	replay := post(body)
	if first.Code != http.StatusOK || replay.Code != first.Code || replay.Body.String() != first.Body.String() {
		t.Fatalf("first=%d %s replay=%d %s", first.Code, first.Body.String(), replay.Code, replay.Body.String())
	}
	changed := strings.Replace(body, `"description":"x"`, `"description":"y"`, 1)
	if got := post(changed); got.Code != http.StatusConflict {
		t.Fatalf("changed-payload status=%d body=%s, want 409", got.Code, got.Body.String())
	}
	stale := strings.Replace(body, `"idempotency_key":"bound"`, `"idempotency_key":"stale"`, 1)
	stale = strings.Replace(stale, `"entries":`, `"expected_versions":{"`+a.String()+`":0,"`+b.String()+`":0},"entries":`, 1)
	if got := post(stale); got.Code != http.StatusConflict {
		t.Fatalf("stale-version status=%d body=%s, want 409", got.Code, got.Body.String())
	}
}
