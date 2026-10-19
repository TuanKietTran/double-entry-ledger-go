package serverruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/domain"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/google/uuid"
)

func TestProductionRouterHealthAndLedgerRoutes(t *testing.T) {
	repo := repository.NewInMemoryLedger()
	handler := BuildRouter(repo)
	for _, path := range []string{"/healthz", "/readyz", "/metrics"} {
		r := httptest.NewRecorder()
		handler.ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != http.StatusOK {
			t.Fatalf("GET %s status %d", path, r.Code)
		}
	}
	id := uuid.New()
	if err := repo.CreateAccount(context.Background(), domain.Account{ID: id, Currency: "USD", Type: domain.Asset, Status: domain.AccountActive}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/transactions", strings.NewReader(`{"idempotency_key":"router-test","entries":[{"account_id":"`+id.String()+`","amount":"1","direction":"DEBIT","currency":"USD"},{"account_id":"`+uuid.NewString()+`","amount":"1","direction":"CREDIT","currency":"USD"}]}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unprovisioned peer status %d body %s", response.Code, response.Body)
	}
}
