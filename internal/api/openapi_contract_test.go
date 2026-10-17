package api

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestOpenAPIListsCurrentRoutesAndStatuses(t *testing.T) {
	spec, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(spec)
	routes := map[string]string{
		"POST /transactions":         "  /v1/transactions:\n    post:",
		"GET /accounts/{id}/balance": "  /v1/accounts/{id}/balance:\n    get:",
	}
	registered := map[string]bool{}
	if err := chi.Walk(NewHandlers(nil, nil).Routes(), func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		registered[method+" "+route] = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for route, path := range routes {
		if !registered[route] {
			t.Errorf("route is not registered: %s", route)
		}
		if !strings.Contains(text, path) {
			t.Errorf("OpenAPI missing route: %s", route)
		}
	}
	for _, status := range []string{"'200':", "'400':", "'500':"} {
		if !strings.Contains(text, status) {
			t.Errorf("OpenAPI missing implemented status %s", status)
		}
	}
	for _, planned := range []string{"POST /v1/accounts", "GET /v1/accounts/{id}/entries", "POST /v1/transactions/{id}/reversals"} {
		if strings.Contains(text, "\n  "+planned) {
			t.Errorf("planned-only route must not be an OpenAPI path: %s", planned)
		}
	}
}
