package api

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

func TestOpenAPIListsCurrentRoutesAndStatuses(t *testing.T) {
	spec, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		OpenAPI string `yaml:"openapi"`
		Paths   map[string]map[string]struct {
			Responses map[string]any `yaml:"responses"`
		} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(spec, &document); err != nil {
		t.Fatalf("parse OpenAPI YAML: %v", err)
	}
	if document.OpenAPI != "3.0.3" {
		t.Fatalf("unexpected OpenAPI version %q", document.OpenAPI)
	}
	registered := map[string]bool{}
	mem := repository.NewInMemoryLedger()
	handlers := NewHandlersWithLedger(nil, nil, command.NewCreateAccountHandler(mem), query.NewGetTransactionHandler(mem), query.NewLedgerQueries(mem), command.NewReverseTransactionHandler(mem))
	if err := chi.Walk(handlers.Routes(), func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		registered[method+" /v1"+route] = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	declared := map[string]bool{}
	for path, operations := range document.Paths {
		for method, operation := range operations {
			key := strings.ToUpper(method) + " " + path
			declared[key] = true
			if !registered[key] {
				t.Errorf("OpenAPI declares unregistered route: %s", key)
			}
			if len(operation.Responses) == 0 {
				t.Errorf("OpenAPI operation has no responses: %s", key)
			}
		}
	}
	for route := range registered {
		if !declared[route] {
			t.Errorf("OpenAPI missing registered route: %s", route)
		}
	}
	for route, statuses := range map[string][]string{
		"POST /v1/accounts": {"201", "400"}, "POST /v1/transactions": {"200", "400", "409"},
		"GET /v1/transactions/{id}":                     {"200", "400", "404"},
		"GET /v1/transactions/by-idempotency-key/{key}": {"200", "404"},
		"POST /v1/transactions/{id}/reversals":          {"200", "400", "409"},
		"GET /v1/accounts/{id}":                         {"200", "400", "404"},
		"GET /v1/accounts/{id}/entries":                 {"200", "400"},
		"GET /v1/trial-balance":                         {"200", "500"},
		"GET /v1/accounts/{id}/balance":                 {"200", "400", "500"},
	} {
		method, path, _ := strings.Cut(route, " ")
		operation := document.Paths[path][strings.ToLower(method)]
		for _, status := range statuses {
			if _, ok := operation.Responses[status]; !ok {
				t.Errorf("OpenAPI missing response %s for %s", status, route)
			}
		}
	}
}
