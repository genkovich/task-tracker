package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metrics holds the Prometheus registry and HTTP collectors for one Server
// instance. A per-instance registry keeps repeated server.New calls (tests
// spin up many) from tripping duplicate-registration panics on the global
// registry.
type metrics struct {
	registry *prometheus.Registry
	duration *prometheus.HistogramVec
	requests *prometheus.CounterVec
}

func newMetrics() *metrics {
	m := &metrics{
		registry: prometheus.NewRegistry(),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method, route and status.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route", "status"}),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests by method, route and status.",
		}, []string{"method", "route", "status"}),
	}
	m.registry.MustRegister(m.duration, m.requests)
	return m
}

// handler serves the /metrics scrape endpoint for this instance's registry.
func (m *metrics) handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// middleware records request count and latency. The route label is the chi
// route pattern (e.g. /api/v1/users/{userId}), read after the handler runs so
// the pattern is fully resolved instead of the raw URL.
func (m *metrics) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		next.ServeHTTP(ww, r)

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		labels := []string{r.Method, route, strconv.Itoa(status)}
		m.duration.WithLabelValues(labels...).Observe(time.Since(start).Seconds())
		m.requests.WithLabelValues(labels...).Inc()
	})
}
