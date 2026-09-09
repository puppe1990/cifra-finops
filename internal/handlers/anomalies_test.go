package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/seed"
	"github.com/puppe1990/cifra-finops/internal/syncer"
)

type stubAnomalyCollector struct {
	stubRangeCollector
	anomalies []awsinv.CostAnomaly
}

func (s stubAnomalyCollector) CostAnomalies(_ context.Context, _ awsinv.Credentials, _, _ time.Time) ([]awsinv.CostAnomaly, error) {
	return s.anomalies, s.err
}

func TestAnomaliesHandler_InertiaComponent(t *testing.T) {
	h := NewAnomaliesHandler(setupTestStore(t), testSite(), cais.Config{}, setupTestViews(t))
	req := inertiaRequest(http.MethodGet, "/anomalies", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	assertInertiaComponent(t, rr, "Anomalies")
}

func TestAnomaliesHandler_mergesExplorerAndSpikes(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(finops.SeedAccountEnv, "111111111111")
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}

	col := stubAnomalyCollector{
		stubRangeCollector: stubRangeCollector{rangeLines: []models.CostLine{
			{Service: "Amazon Lightsail", MonthlyCents: 1000, Source: finops.SourceCE, PeriodStart: "2026-07-01"},
			{Service: "Amazon Lightsail", MonthlyCents: 2500, Source: finops.SourceCE, PeriodStart: "2026-08-01"},
		}},
		anomalies: []awsinv.CostAnomaly{{
			ID: "a1", Kind: "ce", Service: "Amazon S3", Query: "2026-08",
			Start: "2026-08-10", ImpactCents: 800,
		}},
	}
	h := NewAnomaliesHandler(s, testSite(), cais.Config{}, setupTestViews(t)).
		WithSyncer(syncer.New(s, col))
	h.now = func() time.Time { return time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC) }

	req := inertiaRequest(http.MethodGet, "/anomalies", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Anomalies")
}
