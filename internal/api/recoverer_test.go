package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecovererReturnsJSONEnvelopeWithRequestID(t *testing.T) {
	h := Recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Request-ID", "req-123")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError || res.Header().Get("Content-Type") != "application/json" || res.Header().Get("X-Request-ID") != "req-123" {
		t.Fatalf("got %d %v", res.Code, res.Header())
	}
	var body errorEnvelope
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "internal_error" || body.Error.Message != "internal server error" || body.RequestID != "req-123" {
		t.Fatalf("body = %+v", body)
	}
}

func TestRecovererGeneratesRequestIDAndRepanicsAbort(t *testing.T) {
	h := Recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/x", nil))
	if res.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing generated request id")
	}
	abort := Recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	defer func() {
		if recover() != http.ErrAbortHandler {
			t.Fatal("ErrAbortHandler was swallowed")
		}
	}()
	abort.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
}
