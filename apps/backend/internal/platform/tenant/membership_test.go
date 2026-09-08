package tenant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bowerbird/internal/platform/auth"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/http/api"
)

type stubMembershipChecker struct {
	err      error
	called   bool
	tenantID string
	userID   string
}

func (s *stubMembershipChecker) AssertMember(_ context.Context, tenantID, userID string) error {
	s.called = true
	s.tenantID = tenantID
	s.userID = userID
	return s.err
}

func TestRequireMembershipAllowsMember(t *testing.T) {
	checker := &stubMembershipChecker{}
	allowed := false
	handler := RequireMembership(checker, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	ctx := WithTenantID(req.Context(), "tenant-b")
	ctx = auth.WithClaims(ctx, &auth.CustomClaims{UserID: "user-a"})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if !allowed {
		t.Fatal("expected next handler to run")
	}
	if !checker.called || checker.tenantID != "tenant-b" || checker.userID != "user-a" {
		t.Fatalf("unexpected membership check: %+v", checker)
	}
}

func TestRequireMembershipRejectsNonMember(t *testing.T) {
	checker := &stubMembershipChecker{err: appErrors.New(appErrors.CodeForbidden, "tenant access denied")}
	handler := RequireMembership(checker, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not run")
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	ctx := WithTenantID(req.Context(), "tenant-b")
	ctx = auth.WithClaims(ctx, &auth.CustomClaims{UserID: "user-a"})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}

	var doc api.JSONAPIErrorDocument
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("decode error document: %v", err)
	}
	if len(doc.Errors) == 0 || doc.Errors[0].Code != appErrors.CodeForbidden {
		t.Fatalf("expected ERR_FORBIDDEN, got %+v", doc.Errors)
	}
}

func TestRequireMembershipSkipsWhenNoTenantHeader(t *testing.T) {
	checker := &stubMembershipChecker{err: appErrors.New(appErrors.CodeForbidden, "tenant access denied")}
	handler := RequireMembership(checker, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.CustomClaims{UserID: "user-a"}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if checker.called {
		t.Fatal("membership must not be checked without a tenant header")
	}
}
