package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	"github.com/puppe1990/aws-finops/internal/finops"
	"github.com/puppe1990/aws-finops/internal/models"
	"github.com/puppe1990/aws-finops/internal/seed"
	"github.com/puppe1990/aws-finops/internal/syncer"
)

func TestCompareHandler_InertiaComponent(t *testing.T) {
	h := NewCompareHandler(setupTestStore(t), testSite(), cais.Config{}, setupTestViews(t))
	req := inertiaRequest(http.MethodGet, "/compare", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	assertInertiaComponent(t, rr, "Compare")
}

func TestCompareHandler_monthlyRowsFromCostExplorer(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(finops.SeedAccountEnv, "111111111111")
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}

	col := stubRangeCollector{rangeLines: []models.CostLine{
		{Service: "Amazon Lightsail", MonthlyCents: 3220, Source: finops.SourceCE, PeriodStart: "2026-07-01", PeriodEnd: "2026-08-01"},
		{Service: "Amazon Lightsail", MonthlyCents: 1983, Source: finops.SourceCE, PeriodStart: "2026-08-01", PeriodEnd: "2026-09-01"},
	}}
	h := NewCompareHandler(s, testSite(), cais.Config{}, setupTestViews(t)).
		WithSyncer(syncer.New(s, col))
	h.now = func() time.Time { return time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC) }

	req := inertiaRequest(http.MethodGet, "/compare", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Compare")
	if !strings.Contains(rr.Body.String(), "US$ 19,83") {
		t.Fatalf("compare usd missing: %s", rr.Body.String())
	}
}
