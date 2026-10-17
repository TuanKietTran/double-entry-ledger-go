package api

import (
	"net/http/httptest"
	"strings"
	"testing"
)

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
