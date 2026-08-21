package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// deadlineRegistrar reports whether the request context carries a deadline
// (middleware.Timeout sets one) — the cheap, deterministic way to prove a
// route is or isn't wrapped by the 30s request timeout, without waiting out
// a real 30-second clock in a unit test.
type deadlineRegistrar struct {
	hasDeadline *bool
}

func (d deadlineRegistrar) RegisterRoutes(r chi.Router) {
	r.Get("/normal", func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Deadline()
		*d.hasDeadline = ok
		w.WriteHeader(http.StatusOK)
	})
}

func (d deadlineRegistrar) RegisterLongLivedRoutes(r chi.Router) {
	r.Get("/long-lived", func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Deadline()
		*d.hasDeadline = ok
		w.WriteHeader(http.StatusOK)
	})
}

func TestNormalRoutes_HaveRequestTimeoutDeadline(t *testing.T) {
	var hasDeadline bool
	s := New(nil, "http://localhost", nil, deadlineRegistrar{hasDeadline: &hasDeadline})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/normal", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if !hasDeadline {
		t.Fatal("expected a normal route's request context to carry the 30s Timeout deadline")
	}
}

func TestLongLivedRoutes_HaveNoRequestTimeoutDeadline(t *testing.T) {
	var hasDeadline bool
	s := New(nil, "http://localhost", nil, deadlineRegistrar{hasDeadline: &hasDeadline})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/long-lived", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if hasDeadline {
		t.Fatal("expected a long-lived (SSE) route's request context to carry NO Timeout deadline")
	}
}

// TestLongLivedRoutes_OutliveTheStandardTimeout is the slower, end-to-end
// confirmation: a long-lived handler that blocks past 30ms (well under the
// real 30s timeout, but proving no shorter timeout sneaked in) still
// completes normally.
func TestLongLivedRoutes_OutliveTheStandardTimeout(t *testing.T) {
	reg := &slowLongLivedRegistrar{blockFor: 200 * time.Millisecond, served: make(chan struct{})}
	s := New(nil, "http://localhost", nil, reg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/slow-stream", nil)
	rr := httptest.NewRecorder()

	start := time.Now()
	s.Handler().ServeHTTP(rr, req)
	elapsed := time.Since(start)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if elapsed < reg.blockFor {
		t.Fatalf("handler returned after %v, want >= %v", elapsed, reg.blockFor)
	}
}

type slowLongLivedRegistrar struct {
	blockFor time.Duration
	served   chan struct{}
}

func (s *slowLongLivedRegistrar) RegisterRoutes(_ chi.Router) {}

func (s *slowLongLivedRegistrar) RegisterLongLivedRoutes(r chi.Router) {
	r.Get("/slow-stream", func(w http.ResponseWriter, r *http.Request) {
		<-time.After(s.blockFor)
		w.WriteHeader(http.StatusOK)
		close(s.served)
	})
}
