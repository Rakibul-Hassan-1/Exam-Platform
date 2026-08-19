// Package middleware implements cross-cutting HTTP concerns: JWT
// authentication, role-based authorization, CORS, and request logging.
package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	"examplatform/internal/auth"
	"examplatform/internal/httpx"
	"examplatform/internal/models"
)

// RequireAuth verifies the Authorization: Bearer <token> header and
// attaches the parsed claims to the request context. A student must
// not be able to access teacher-only APIs, and vice versa — role
// checks are layered on top via RequireRole.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				httpx.Error(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")

			claims, err := auth.ParseToken(secret, token)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := auth.WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole restricts a handler to one or more roles. Must be
// chained after RequireAuth.
func RequireRole(roles ...models.Role) func(http.Handler) http.Handler {
	allowed := make(map[models.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				httpx.Error(w, http.StatusUnauthorized, "not authenticated")
				return
			}
			if !allowed[claims.Role] {
				httpx.Error(w, http.StatusForbidden, "you do not have permission to perform this action")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS allows the configured frontend origin to call the API with
// credentials/headers needed for JWT bearer auth.
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Logger logs method, path, status-adjacent timing, and duration for every request.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}

// Chain composes middleware in the order given (first wraps outermost).
func Chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
