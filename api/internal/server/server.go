package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"github.com/genkovich/task-tracker/api/internal/platform/database"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
)

// RouteRegistrar is implemented by module handlers to register their routes.
type RouteRegistrar interface {
	RegisterRoutes(r chi.Router)
}

// ProtectedRouteRegistrar is optionally implemented by modules that have routes
// requiring authentication.
type ProtectedRouteRegistrar interface {
	RegisterProtectedRoutes(r chi.Router)
}

// StreamingRouteRegistrar is optionally implemented by modules that serve
// long-lived streaming responses (e.g. SSE). These routes share the rate
// limiter and metrics middleware with every other route but are mounted
// outside the per-request timeout, which would otherwise cancel the stream's
// context mid-flight.
type StreamingRouteRegistrar interface {
	RegisterStreamingRoutes(r chi.Router)
}

// Option configures the Server.
type Option func(*Server)

// WithAppEnv sets the application environment (e.g. "production", "development").
func WithAppEnv(env string) Option {
	return func(s *Server) {
		s.appEnv = env
	}
}

// WithRequestTimeout overrides the per-request timeout applied to
// non-streaming routes (default 30s). Streaming routes are never subject to
// it.
func WithRequestTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.requestTimeout = d
	}
}

type Server struct {
	router         *chi.Mux
	db             *database.DB
	corsOrigins    string
	appEnv         string
	authMW         func(http.Handler) http.Handler
	registrars     []RouteRegistrar
	metrics        *metrics
	requestTimeout time.Duration
}

func New(db *database.DB, corsOrigins string, authMW func(http.Handler) http.Handler, opts ...any) *Server {
	s := &Server{
		router:         chi.NewRouter(),
		db:             db,
		corsOrigins:    corsOrigins,
		appEnv:         "development",
		authMW:         authMW,
		metrics:        newMetrics(),
		requestTimeout: 30 * time.Second,
	}

	// Separate Options from RouteRegistrars.
	for _, o := range opts {
		switch v := o.(type) {
		case Option:
			v(s)
		case RouteRegistrar:
			s.registrars = append(s.registrars, v)
		}
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddleware() {
	origins := strings.Split(s.corsOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	s.router.Use(securityHeaders)
	s.router.Use(middleware.RequestID)
	// Caddy is the single trusted hop in front of the API, so the rightmost
	// X-Forwarded-For entry is the client. Read it with middleware.GetClientIP.
	s.router.Use(middleware.ClientIPFromXFF())
	s.router.Use(requestLogger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(requestSizeLimit(1 << 20)) // 1 MB
}

// securityHeaders adds standard security response headers to every response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		next.ServeHTTP(w, r)
	})
}

// requestLogger logs one line per request via slog, in place of
// middleware.Logger, whose formatter prints the full RequestURI — that
// would persist public-link tokens (capability URLs) in the logs. The path
// is redacted instead; the query string is deliberately not logged.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		next.ServeHTTP(ww, r)

		slog.Info("http request",
			"method", r.Method,
			"path", redactPublicToken(r.URL.Path),
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration", time.Since(start).String(),
			"request_id", middleware.GetReqID(r.Context()),
			"client_ip", middleware.GetClientIP(r.Context()),
		)
	})
}

// redactPublicToken replaces the path segment right after "/public/" with a
// placeholder, so capability-URL tokens never reach the logs.
func redactPublicToken(path string) string {
	const marker = "/public/"
	i := strings.Index(path, marker)
	if i == -1 {
		return path
	}
	rest := path[i+len(marker):]
	if j := strings.Index(rest, "/"); j != -1 {
		return path[:i+len(marker)] + "{redacted}" + rest[j:]
	}
	return path[:i+len(marker)] + "{redacted}"
}

// requestSizeLimit wraps each request body with http.MaxBytesReader to enforce
// a maximum request body size.
func requestSizeLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) setupRoutes() {
	// Operational endpoints stay outside the rate limiter and the HTTP
	// metrics middleware so probes and scrapes never get throttled and never
	// pollute the route-labelled series.
	s.router.Get("/livez", s.handleLivez)
	s.router.Get("/readyz", s.handleReadyz)
	s.router.Method(http.MethodGet, "/metrics", s.metrics.handler())

	s.router.Group(func(r chi.Router) {
		// Metrics wrap the rate limiter, not the other way around: a 429
		// refused by the limiter must still be counted (status="429"), or an
		// abuse wave looks like silence on the dashboards.
		r.Use(s.metrics.middleware)
		// General rate limit: 60 req/min per IP.
		r.Use(httprate.Limit(60, time.Minute, httprate.WithKeyFuncs(httputil.ClientIPKey)))

		r.Route("/api/v1", func(r chi.Router) {
			// Regular request/response routes run under the per-request
			// timeout; streaming routes below must not — the timeout would
			// cancel a long-lived stream's context mid-flight.
			r.Group(func(r chi.Router) {
				r.Use(middleware.Timeout(s.requestTimeout))

				r.Get("/health", s.handleHealth)

				// Public routes.
				for _, reg := range s.registrars {
					reg.RegisterRoutes(r)
				}

				// Protected routes (require auth middleware).
				r.Group(func(r chi.Router) {
					if s.authMW != nil {
						r.Use(s.authMW)
					}
					for _, reg := range s.registrars {
						if pr, ok := reg.(ProtectedRouteRegistrar); ok {
							pr.RegisterProtectedRoutes(r)
						}
					}
				})
			})

			// Streaming routes (SSE): rate-limited and metered like the
			// rest, but no per-request timeout.
			for _, reg := range s.registrars {
				if sr, ok := reg.(StreamingRouteRegistrar); ok {
					sr.RegisterStreamingRoutes(r)
				}
			}
		})
	})
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	httpStatus := http.StatusOK

	if s.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := s.db.Ping(ctx); err != nil {
			slog.Error("health check: database ping failed", "error", err)
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
		}
	}

	writeStatus(w, httpStatus, status)
}

// handleLivez reports process liveness: the server is up and serving requests.
func (s *Server) handleLivez(w http.ResponseWriter, _ *http.Request) {
	writeStatus(w, http.StatusOK, "ok")
}

// handleReadyz reports readiness: the database must be reachable within 2s.
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeStatus(w, http.StatusServiceUnavailable, "unavailable")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		slog.Error("readiness check: database ping failed", "error", err)
		writeStatus(w, http.StatusServiceUnavailable, "unavailable")
		return
	}

	writeStatus(w, http.StatusOK, "ready")
}

func writeStatus(w http.ResponseWriter, httpStatus int, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": status}); err != nil {
		slog.Error("failed to write status response", "error", err)
	}
}
