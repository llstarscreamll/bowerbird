package auth

import (
	"context"
	"net/http"
	"strings"

	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
)

type contextKey string

const userContextKey = contextKey("user_claims")

func Middleware(tokenGen *TokenGenerator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeUnauthorized(w, r)
				return
			}

			scheme, token, ok := strings.Cut(authHeader, " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
				writeUnauthorized(w, r)
				return
			}

			claims, err := tokenGen.ValidateAccessToken(token)
			if err != nil {
				writeUnauthorized(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) (*CustomClaims, bool) {
	claims, ok := ctx.Value(userContextKey).(*CustomClaims)
	return claims, ok
}

func writeUnauthorized(w http.ResponseWriter, r *http.Request) {
	api.RespondWithError(w, r, appErrors.New(appErrors.CodeUnauthorized, "unauthorized"), false)
}
