package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/command"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/query"
	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
	"gopkg.in/yaml.v3"
)

type responseSchema struct {
	Ref        string                    `yaml:"$ref"`
	Type       string                    `yaml:"type"`
	Required   []string                  `yaml:"required"`
	Properties map[string]responseSchema `yaml:"properties"`
	Items      *responseSchema           `yaml:"items"`
	Enum       []string                  `yaml:"enum"`
}

type contractResponse struct {
	Ref     string `yaml:"$ref"`
	Content map[string]struct {
		Schema responseSchema `yaml:"schema"`
	} `yaml:"content"`
}

type responseContract struct {
	Paths map[string]map[string]struct {
		Responses map[string]contractResponse `yaml:"responses"`
	} `yaml:"paths"`
	Components struct {
		Schemas   map[string]responseSchema   `yaml:"schemas"`
		Responses map[string]contractResponse `yaml:"responses"`
	} `yaml:"components"`
}

func loadResponseContract(t *testing.T) responseContract {
	t.Helper()
	data, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc responseContract
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func (d responseContract) schema(s responseSchema) (responseSchema, error) {
	if s.Ref == "" {
		return s, nil
	}
	const prefix = "#/components/schemas/"
	found, ok := d.Components.Schemas[strings.TrimPrefix(s.Ref, prefix)]
	if !strings.HasPrefix(s.Ref, prefix) || !ok || found.Ref != "" {
		return s, fmt.Errorf("invalid schema reference %q", s.Ref)
	}
	return found, nil
}

func (d responseContract) response(t *testing.T, path, method string, status int) responseSchema {
	t.Helper()
	r, ok := d.Paths[path][strings.ToLower(method)].Responses[strconv.Itoa(status)]
	if !ok {
		t.Fatalf("missing response %s %s %d", method, path, status)
	}
	if r.Ref != "" {
		const prefix = "#/components/responses/"
		var found bool
		r, found = d.Components.Responses[strings.TrimPrefix(r.Ref, prefix)]
		if !found {
			t.Fatal("unresolved response reference")
		}
	}
	media, ok := r.Content["application/json"]
	if !ok {
		t.Fatalf("missing JSON response schema: %s %s %d", method, path, status)
	}
	if _, err := d.schema(media.Schema); err != nil {
		t.Fatal(err)
	}
	return media.Schema
}

func TestOpenAPIResponseProperties(t *testing.T) {
	doc := loadResponseContract(t)
	for name, fields := range map[string]string{
		"AccountCreated": "account_id", "TransactionCreated": "transaction_id",
		"Account":     "account_id owner_id currency type status version",
		"Entry":       "entry_id account_id created_at amount direction currency",
		"Transaction": "transaction_id reversal_of idempotency_key description status entries",
		"Balance":     "account_id available currency", "TrialBalanceRow": "account_id currency balance",
		"ErrorEnvelope": "error request_id",
	} {
		t.Run(name, func(t *testing.T) {
			s, ok := doc.Components.Schemas[name]
			if !ok {
				t.Fatal("missing schema")
			}
			var got []string
			for key := range s.Properties {
				got = append(got, key)
			}
			want := strings.Fields(fields)
			sort.Strings(got)
			sort.Strings(want)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("properties %v; want %v", got, want)
			}
			required := append([]string(nil), s.Required...)
			if name == "Transaction" {
				want = strings.Fields("transaction_id idempotency_key description status entries")
				sort.Strings(want)
			}
			sort.Strings(required)
			if !reflect.DeepEqual(required, want) {
				t.Fatalf("required %v; want %v", required, want)
			}
		})
	}
	for path, operations := range doc.Paths {
		for method, operation := range operations {
			for status := range operation.Responses {
				code, err := strconv.Atoi(status)
				if err != nil {
					t.Fatal(err)
				}
				s := doc.response(t, path, method, code)
				if code >= 400 && s.Ref != "#/components/schemas/ErrorEnvelope" {
					t.Errorf("%s %s %s must use ErrorEnvelope", method, path, status)
				}
			}
		}
	}
}

// This deliberately checks the schema subset used by response bodies, not all
// OpenAPI semantics (for example request constraints and numeric bounds).
func (d responseContract) validate(s responseSchema, value any) error {
	s, err := d.schema(s)
	if err != nil {
		return err
	}
	switch s.Type {
	case "object":
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("want object, got %T", value)
		}
		for _, key := range s.Required {
			if _, ok := object[key]; !ok {
				return fmt.Errorf("missing required %s", key)
			}
		}
		for key, value := range object {
			child, ok := s.Properties[key]
			if !ok {
				return fmt.Errorf("undocumented property %s", key)
			}
			if err := d.validate(child, value); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
		}
	case "array":
		array, ok := value.([]any)
		if !ok || s.Items == nil {
			return fmt.Errorf("want array with items, got %T", value)
		}
		for i, value := range array {
			if err := d.validate(*s.Items, value); err != nil {
				return fmt.Errorf("item %d: %w", i, err)
			}
		}
	case "string":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("want string, got %T", value)
		}
		if len(s.Enum) > 0 {
			found := false
			for _, allowed := range s.Enum {
				if text == allowed {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("value %q not in enum", text)
			}
		}
	case "integer":
		number, ok := value.(json.Number)
		if !ok {
			return fmt.Errorf("want integer, got %T", value)
		}
		if _, err := number.Int64(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported or missing schema type %q", s.Type)
	}
	return nil
}

func TestHandlerBodiesMatchOpenAPIResponses(t *testing.T) {
	doc := loadResponseContract(t)
	repo := repository.NewInMemoryLedger()
	h := NewHandlersWithLedger(command.NewPostTransactionHandler(repo), query.NewGetBalanceHandler(repo), command.NewCreateAccountHandler(repo), query.NewGetTransactionHandler(repo), query.NewLedgerQueries(repo), command.NewReverseTransactionHandler(repo))
	router := h.Routes()
	call := func(method, path, template, body string, status int) map[string]any {
		t.Helper()
		r := httptest.NewRecorder()
		router.ServeHTTP(r, httptest.NewRequest(method, path, strings.NewReader(body)))
		if r.Code != status {
			t.Fatalf("%s %s = %d: %s", method, path, r.Code, r.Body.String())
		}
		if r.Header().Get("Content-Type") != "application/json" {
			t.Fatal("missing JSON content type")
		}
		var value any
		decoder := json.NewDecoder(r.Body)
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
		if err := doc.validate(doc.response(t, template, method, status), value); err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		object, _ := value.(map[string]any)
		return object
	}
	const a = "11111111-1111-4111-8111-111111111111"
	const b = "22222222-2222-4222-8222-222222222222"
	for _, id := range []string{a, b} {
		call("POST", "/accounts", "/v1/accounts", fmt.Sprintf(`{"account_id":%q,"currency":"USD","type":"ASSET"}`, id), http.StatusCreated)
	}
	payload := fmt.Sprintf(`{"idempotency_key":"contract-post","entries":[{"account_id":%q,"amount":"12.34","direction":"DEBIT","currency":"USD"},{"account_id":%q,"amount":"12.34","direction":"CREDIT","currency":"USD"}]}`, a, b)
	tx := call("POST", "/transactions", "/v1/transactions", payload, 200)["transaction_id"].(string)
	call("GET", "/transactions/"+tx, "/v1/transactions/{id}", "", 200)
	call("GET", "/transactions/by-idempotency-key/contract-post", "/v1/transactions/by-idempotency-key/{key}", "", 200)
	call("GET", "/accounts/"+a, "/v1/accounts/{id}", "", 200)
	call("GET", "/accounts/"+a+"/entries", "/v1/accounts/{id}/entries", "", 200)
	call("GET", "/accounts/"+a+"/balance", "/v1/accounts/{id}/balance", "", 200)
	call("GET", "/accounts/"+a+"/balance?as_of=2100-01-01T00:00:00Z", "/v1/accounts/{id}/balance", "", 200)
	call("GET", "/trial-balance", "/v1/trial-balance", "", 200)
	reversal := call("POST", "/transactions/"+tx+"/reversals", "/v1/transactions/{id}/reversals", `{"idempotency_key":"contract-reverse"}`, 200)["transaction_id"].(string)
	call("GET", "/transactions/"+reversal, "/v1/transactions/{id}", "", 200)
	call("GET", "/accounts/"+b+"/entries?offset=500", "/v1/accounts/{id}/entries", "", 200)
	for _, tc := range []struct {
		method, path, template, body string
		status                       int
	}{
		{"POST", "/accounts", "/v1/accounts", `{}`, 400},
		{"POST", "/transactions", "/v1/transactions", `{}`, 400},
		{"POST", "/transactions", "/v1/transactions", strings.Replace(payload, "12.34", "13.34", -1), 409},
		{"GET", "/transactions/invalid", "/v1/transactions/{id}", "", 400},
		{"GET", "/transactions/" + a, "/v1/transactions/{id}", "", 404},
		{"GET", "/transactions/by-idempotency-key/missing", "/v1/transactions/by-idempotency-key/{key}", "", 404},
		{"POST", "/transactions/invalid/reversals", "/v1/transactions/{id}/reversals", `{}`, 400},
		{"POST", "/transactions/" + tx + "/reversals", "/v1/transactions/{id}/reversals", `{"idempotency_key":"contract-post"}`, 409},
		{"GET", "/accounts/invalid", "/v1/accounts/{id}", "", 400},
		{"GET", "/accounts/33333333-3333-4333-8333-333333333333", "/v1/accounts/{id}", "", 404},
		{"GET", "/accounts/invalid/entries", "/v1/accounts/{id}/entries", "", 400},
		{"GET", "/accounts/invalid/balance", "/v1/accounts/{id}/balance", "", 400},
	} {
		t.Run(tc.method+tc.path+strconv.Itoa(tc.status), func(t *testing.T) {
			call(tc.method, tc.path, tc.template, tc.body, tc.status)
		})
	}
}

func TestResponseSchemaValidatorRejectsDrift(t *testing.T) {
	doc := loadResponseContract(t)
	for _, value := range []any{nil, map[string]any{}, map[string]any{"transaction_id": 1}, map[string]any{"transaction_id": "id", "extra": true}} {
		if err := doc.validate(responseSchema{Ref: "#/components/schemas/TransactionCreated"}, value); err == nil {
			t.Fatalf("accepted invalid body %#v", value)
		}
	}
}
