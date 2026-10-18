package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	RequestID string `json:"request_id"`
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = uuid.NewString()
		}
		r.Header.Set("X-Request-ID", requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

// Recoverer assigns a request ID and converts handler panics into the JSON error envelope.
func Recoverer(next http.Handler) http.Handler {
	return requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				writeError(w, r, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	}))
}

func writeError(w http.ResponseWriter, r *http.Request, message string, status int) {
	requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
	if requestID == "" {
		requestID = uuid.NewString()
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(status)
	var response errorEnvelope
	response.RequestID = requestID
	switch status {
	case http.StatusConflict:
		response.Error.Code = "conflict"
	case http.StatusNotFound:
		response.Error.Code = "not_found"
	case http.StatusServiceUnavailable:
		response.Error.Code = "unavailable"
	case http.StatusInternalServerError:
		response.Error.Code = "internal_error"
	default:
		response.Error.Code = "bad_request"
	}
	response.Error.Message = message
	_ = json.NewEncoder(w).Encode(response)
}
