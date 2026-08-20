package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersPresent(t *testing.T) {
	s := New(nil, "*", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	expected := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
		"Content-Security-Policy": "default-src 'self'",
	}

	for header, want := range expected {
		got := rec.Header().Get(header)
		if got != want {
			t.Errorf("header %s = %q, want %q", header, got, want)
		}
	}
}

func TestRateLimitHeadersPresent(t *testing.T) {
	s := New(nil, "http://localhost", nil, WithAppEnv("development"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	limitHeader := rr.Header().Get("X-Ratelimit-Limit")
	if limitHeader == "" {
		t.Error("expected X-Ratelimit-Limit header to be present")
	}

	remainingHeader := rr.Header().Get("X-Ratelimit-Remaining")
	if remainingHeader == "" {
		t.Error("expected X-Ratelimit-Remaining header to be present")
	}
}

func TestRateLimitExceeded(t *testing.T) {
	s := New(nil, "http://localhost", nil, WithAppEnv("development"))

	// The general rate limit is 60 req/min. Send 65 requests.
	var lastStatus int
	for i := 0; i < 65; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		rr := httptest.NewRecorder()
		s.Handler().ServeHTTP(rr, req)
		lastStatus = rr.Code
	}

	if lastStatus != http.StatusTooManyRequests {
		t.Errorf("expected 429 after exceeding rate limit, got %d", lastStatus)
	}
}

func TestLivez(t *testing.T) {
	s := New(nil, "http://localhost", nil)

	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /livez, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode /livez body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf(`expected status "ok", got %q`, body["status"])
	}
}

func TestReadyzWithoutDatabase(t *testing.T) {
	s := New(nil, "http://localhost", nil)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for /readyz without database, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode /readyz body: %v", err)
	}
	if body["status"] != "unavailable" {
		t.Errorf(`expected status "unavailable", got %q`, body["status"])
	}
}

func TestOperationalEndpointsBypassRateLimit(t *testing.T) {
	s := New(nil, "http://localhost", nil)

	for _, path := range []string{"/livez", "/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		s.Handler().ServeHTTP(rr, req)

		if rr.Header().Get("X-Ratelimit-Limit") != "" {
			t.Errorf("expected %s to bypass the rate limiter, but rate-limit headers are present", path)
		}
	}
}

func TestMetricsRecordsAPIRequestsOnly(t *testing.T) {
	s := New(nil, "http://localhost", nil)

	// One API request that must show up in the metrics...
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	// ...and one probe request that must not.
	req = httptest.NewRequest(http.MethodGet, "/livez", nil)
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /metrics, got %d", rr.Code)
	}
	body := rr.Body.String()

	if !strings.Contains(body, `http_requests_total{method="GET",route="/api/v1/health",status="200"} 1`) {
		t.Error("expected http_requests_total counter for /api/v1/health")
	}
	if !strings.Contains(body, `http_request_duration_seconds_count{method="GET",route="/api/v1/health",status="200"} 1`) {
		t.Error("expected http_request_duration_seconds histogram for /api/v1/health")
	}
	if strings.Contains(body, `route="/livez"`) || strings.Contains(body, `route="/metrics"`) {
		t.Error("operational endpoints must not appear as route labels in HTTP metrics")
	}
}
