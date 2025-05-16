package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
)

// Dummy handler to check context and header
func makeTestHandler(t *testing.T, wantID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gotID, _ := r.Context().Value(RequestIDKey).(string)
		if wantID != "" && gotID != wantID {
			t.Errorf("want request id in context %q, got %q", wantID, gotID)
		}
		// Also check response header
		respID := w.Header().Get("X-Request-Id")
		if wantID != "" && respID != wantID {
			t.Errorf("want X-Request-Id header %q, got %q", wantID, respID)
		}
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "ok")
	}
}

func Test_requestIDMiddlewareHeaderExists(t *testing.T) {
	api := &API{}
	wantID := "test-req-id-123"
	handler := api.requestIDMiddleware(makeTestHandler(t, wantID))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", wantID)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want status code %v, got %v", http.StatusOK, rr.Code)
	}
	got := rr.Header().Get("X-Request-Id")
	if got != wantID {
		t.Errorf("want X-Request-Id header %q, got %q", wantID, got)
	}
}

func Test_requestIDMiddlewareHeaderNotExists(t *testing.T) {
	api := &API{}
	handler := api.requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, _ := r.Context().Value(RequestIDKey).(string)
		if gotID == "" {
			t.Error("want non-empty request id in context when header is missing")
		}
		// Check that the header is also set
		respID := w.Header().Get("X-Request-Id")
		if respID == "" {
			t.Error("want non-empty X-Request-Id header when header is missing")
		}
		// Check that it's a valid UUID
		_, err := uuid.FromString(respID)
		if err != nil {
			t.Errorf("want valid UUID for generated request id, got %q", respID)
		}
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("want status code %v, got %v", http.StatusOK, rr.Code)
	}
}

func Test_requestIDMiddlewareUUIDFormat(t *testing.T) {
	api := &API{}
	handler := api.requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, _ := r.Context().Value(RequestIDKey).(string)
		if gotID == "" {
			t.Error("want non-empty request id in context")
		}
		// Check UUID format (should contain 4 dashes)
		if strings.Count(gotID, "-") != 4 {
			t.Errorf("want UUID format for request id, got %q", gotID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}
