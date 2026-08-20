package authmw

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/genkovich/task-tracker/api/internal/platform/apperr"
	"github.com/genkovich/task-tracker/api/internal/platform/httputil"
)

type Claims struct {
	UserID uuid.UUID
	Email  string
	Role   string
}

func (c *Claims) IsAdmin() bool {
	return c.Role == "admin"
}

type TokenValidator interface {
	Validate(token string) (*Claims, error)
}

type contextKey struct{}

func Middleware(v TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				httputil.WriteError(w, &apperr.Error{
					Code:       "auth.missing_token",
					Message:    "authorization token is required",
					StatusCode: http.StatusUnauthorized,
				})
				return
			}

			claims, err := v.Validate(token)
			if err != nil {
				httputil.WriteError(w, &apperr.Error{
					Code:       "auth.invalid_token",
					Message:    "invalid or expired token",
					StatusCode: http.StatusUnauthorized,
				})
				return
			}

			ctx := context.WithValue(r.Context(), contextKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AuthClaims(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(contextKey{}).(*Claims)
	return c, ok
}

// WithClaims injects claims into a context without going through Middleware.
//
// USE IN TESTS ONLY. It exists so handler-level unit tests can exercise routes
// that read claims via AuthClaims without signing real JWTs. Calling this from
// a request-handling path defeats the auth middleware and must never happen in
// production code.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, contextKey{}, claims)
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}
