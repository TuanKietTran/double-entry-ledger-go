package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
)

func TestErrorsUseJSONEnvelopeAndBoundedBodies(t *testing.T) {
	repo := repository.NewInMemoryLedger()
	h := NewHandlers(command.NewPostTransactionHandler(repo), query.NewGetBalanceHandler(repo))
	bad := httptest.NewRequest("POST", "/transactions", strings.NewReader("not-json"))
	bad.Header.Set("X-Request-ID", "request-fixture-1")
	badResponse := httptest.NewRecorder()
	h.Routes().ServeHTTP(badResponse, bad)
	if badResponse.Code != 400 || badResponse.Header().Get("Content-Type") != "application/json" || badResponse.Header().Get("X-Request-ID") != "request-fixture-1" || !strings.Contains(badResponse.Body.String(), `"request_id":"request-fixture-1"`) {
		t.Fatalf("bad request response = %d headers=%v body=%q", badResponse.Code, badResponse.Header(), badResponse.Body.String())
	}
	large := httptest.NewRequest("POST", "/transactions", strings.NewReader(strings.Repeat(" ", maxJSONBodyBytes+1)))
	largeResponse := httptest.NewRecorder()
	h.Routes().ServeHTTP(largeResponse, large)
	if largeResponse.Code != 400 || !strings.Contains(largeResponse.Body.String(), "invalid json") {
		t.Fatalf("oversized request response = %d %q", largeResponse.Code, largeResponse.Body.String())
	}
	trailing := httptest.NewRequest("POST", "/transactions", strings.NewReader(`{} {}`))
	trailingResponse := httptest.NewRecorder()
	h.Routes().ServeHTTP(trailingResponse, trailing)
	if trailingResponse.Code != 400 || !strings.Contains(trailingResponse.Body.String(), "invalid json") {
		t.Fatalf("trailing JSON response = %d %q", trailingResponse.Code, trailingResponse.Body.String())
	}
}
