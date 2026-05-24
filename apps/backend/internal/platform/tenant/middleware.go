package tenant

import (
	"net/http"
)

// Middleware extracts the X-Tenant-ID header and injects it into the request context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")

		// If the tenant is required for all routes, you could block it here.
		// For health checks and public webhooks, it might be optional.
		// We add it to the context if present.
		if tenantID != "" {
			ctx := WithTenant(r.Context(), tenantID)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}
