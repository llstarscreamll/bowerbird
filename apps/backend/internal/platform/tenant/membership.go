package tenant

import (
	"context"
	"net/http"

	"github.com/bowerbird/internal/platform/auth"
	"github.com/bowerbird/internal/platform/http/api"
)

// MembershipChecker verifies that the authenticated user belongs to a tenant.
type MembershipChecker interface {
	AssertMember(ctx context.Context, tenantID, userID string) error
}

// RequireMembership rejects authenticated requests whose X-Tenant-ID is not a
// tenant the caller belongs to. Requests without a tenant header or without
// claims (workers, public routes) pass through.
func RequireMembership(checker MembershipChecker, isDev bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, err := TenantIDFromContext(r.Context())
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			if err := checker.AssertMember(r.Context(), tenantID, claims.UserID); err != nil {
				api.RespondWithError(w, r, err, isDev)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
