package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	"github.com/puppe1990/aws-finops/internal/seed"
)

func TestSettingsHandler_includesPolicyAndCloudShell(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}

	h := NewSettingsHandler(s, testSite(), cais.Config{}, setupTestViews(t))
	req := inertiaRequest(http.MethodGet, "/settings", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Settings")
	if !strings.Contains(rr.Body.String(), "CifraFinOpsRead") && !strings.Contains(rr.Body.String(), "2012-10-17") {
		t.Errorf("policy missing from HTML")
	}
	if !strings.Contains(rr.Body.String(), "aws") {
		t.Errorf("cloudShell missing from HTML: %s", rr.Body.String())
	}
}
