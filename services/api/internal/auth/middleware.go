package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/Janith-Bhashitha/fileforge/services/api/internal/apikeys"
)

type contextKey string

const claimsContextKey contextKey = "claims"

func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":"missing or invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := VerifyToken(secret, tokenString)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// APIKeyOrJWTMiddleware is what makes FileForge a public API rather than
// just a backend for its own frontend: a request authenticates with either
// an X-API-Key header or the normal JWT, and every handler downstream
// reads the same Claims either way, never needing to know which happened.
func APIKeyOrJWTMiddleware(jwtSecret string, keys *apikeys.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// The wrapped JWT-only handler is built once per `next`, not per
		// request, the same as any other middleware constructor.
		jwtHandler := Middleware(jwtSecret)(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rawKey := r.Header.Get("X-API-Key"); rawKey != "" {
				key, err := keys.Verify(r.Context(), rawKey)
				if err != nil {
					http.Error(w, `{"error":"invalid or revoked api key"}`, http.StatusUnauthorized)
					return
				}
				// Email is left blank: an API key authenticates an owner,
				// not an interactive session, and nothing on the /api/v1
				// surface needs the email off the claims.
				claims := &Claims{UserID: key.OwnerID}
				ctx := context.WithValue(r.Context(), claimsContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			jwtHandler.ServeHTTP(w, r)
		})
	}
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok
}
