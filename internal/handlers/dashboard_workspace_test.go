package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	"github.com/puppe1990/aws-finops/internal/finops"
	"github.com/puppe1990/aws-finops/internal/seed"
)

func TestDashboardHandler_showsSeededAccount(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(finops.SeedAccountEnv, "111111111111")
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}

	h := NewDashboardHandler(setupTestRenderer(t), s, testSite(), cais.Config{}, setupTestViews(t))
	req := inertiaRequest(http.MethodGet, "/dashboard", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Dashboard")
	if !strings.Contains(rr.Body.String(), "111111111111") {
		t.Fatalf("seeded account missing: %s", rr.Body.String())
	}
}
