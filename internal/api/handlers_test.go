package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
)

func TestCreateAccountRouteProvisioning(t *testing.T) {
	mem := repository.NewInMemoryLedger()
	h := NewHandlers(nil, nil, command.NewCreateAccountHandler(mem))
	id := "550e8400-e29b-41d4-a716-446655440000"
	body := `{"account_id":"` + id + `","owner_id":"fixture","currency":"USD","type":"ASSET"}`
	req := httptest.NewRequest("POST", "/accounts", strings.NewReader(body))
	res := httptest.NewRecorder()
	h.Routes().ServeHTTP(res, req)
	if res.Code != 201 || !strings.Contains(res.Body.String(), id) {
		t.Fatalf("response = %d %q, want HTTP 201 with account_id", res.Code, res.Body.String())
	}
	req = httptest.NewRequest("POST", "/accounts", strings.NewReader(body))
	res = httptest.NewRecorder()
	h.Routes().ServeHTTP(res, req)
	if res.Code != 201 {
		t.Fatalf("idempotent provisioning response = %d %q, want 201", res.Code, res.Body.String())
	}
}

func TestPostTransactionRejectsUnknownDirection(t *testing.T) {
	h := NewHandlers(nil, nil)
	body := `{"idempotency_key":"direction-test","entries":[{"account_id":"550e8400-e29b-41d4-a716-446655440000","amount":"1","direction":"credit","currency":"USD"}]}`
	req := httptest.NewRequest("POST", "/transactions", strings.NewReader(body))
	res := httptest.NewRecorder()
	h.Routes().ServeHTTP(res, req)
	if res.Code != 400 || !strings.Contains(res.Body.String(), "invalid direction") {
		t.Fatalf("response = %d %q, want HTTP 400 invalid direction", res.Code, res.Body.String())
	}
}
