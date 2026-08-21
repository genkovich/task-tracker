package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type highTrafficRegistrar struct{}

func (highTrafficRegistrar) RegisterRoutes(_ chi.Router) {}

func (highTrafficRegistrar) RegisterHighTrafficRoutes(r chi.Router, timeoutMW func(http.Handler) http.Handler) {
	r.With(timeoutMW).Get("/high-traffic", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestHighTrafficRoutes_GetTheHigherLimit(t *testing.T) {
	s := New(nil, "http://localhost", nil, highTrafficRegistrar{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/high-traffic", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("X-Ratelimit-Limit"); got != "300" {
		t.Fatalf("expected X-Ratelimit-Limit 300, got %q", got)
	}
}

// TestHighTrafficRoutes_IndependentFromStandardLimit proves the two tiers
// don't stack: exhausting the standard 60/min group must not throttle the
// high-traffic route, and vice versa.
func TestHighTrafficRoutes_IndependentFromStandardLimit(t *testing.T) {
	s := New(nil, "http://localhost", nil, highTrafficRegistrar{})

	for i := 0; i < 65; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		req.RemoteAddr = "192.0.2.7:1234"
		rr := httptest.NewRecorder()
		s.Handler().ServeHTTP(rr, req)
	}

	// The standard group is now over its 60/min limit for this IP — the
	// high-traffic route must be unaffected.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/high-traffic", nil)
	req.RemoteAddr = "192.0.2.7:1234"
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected the high-traffic route to be unaffected by the standard group's limit, got %d", rr.Code)
	}
}
