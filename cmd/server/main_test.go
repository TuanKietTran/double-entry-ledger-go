package main

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TuanKietTran/double-entry-ledger-go/internal/repository"
)

func TestLoadConfigValidation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{name: "defaults", env: map[string]string{}},
		{name: "custom addr", env: map[string]string{"HTTP_ADDR": "127.0.0.1:8101"}},
		{name: "invalid addr", env: map[string]string{"HTTP_ADDR": "broken"}, wantErr: true},
		{name: "invalid database", env: map[string]string{"DATABASE_URL": "http://example.invalid/db"}, wantErr: true},
		{name: "postgres database", env: map[string]string{"DATABASE_URL": "postgres://localhost/ledger"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := loadConfig(func(key string) string { return tc.env[key] })
			if (err != nil) != tc.wantErr {
				t.Fatalf("loadConfig() = %+v, %v; wantErr=%v", cfg, err, tc.wantErr)
			}
			if err == nil && cfg.addr == "" {
				t.Fatal("address is empty")
			}
		})
	}
}

func TestOperationalEndpointsAndMetrics(t *testing.T) {
	h := buildRouter(repository.NewInMemoryLedger())
	for _, path := range []string{"/healthz", "/readyz"} {
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest("GET", path, nil))
		if res.Code != 200 {
			t.Errorf("GET %s = %d %s", path, res.Code, res.Body.String())
		}
	}
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest("GET", "/metrics", nil))
	if res.Code != 200 || !strings.Contains(res.Body.String(), "ledger_http_requests_total") {
		t.Fatalf("metrics = %d %q", res.Code, res.Body.String())
	}
}
