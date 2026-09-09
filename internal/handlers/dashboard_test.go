package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/seed"
	"github.com/puppe1990/cifra-finops/internal/syncer"
)

func TestDashboardHandler_InertiaComponent(t *testing.T) {
	h := NewDashboardHandler(setupTestRenderer(t), setupTestStore(t), testSite(), cais.Config{}, setupTestViews(t))

	req := inertiaRequest(http.MethodGet, "/dashboard", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	assertInertiaComponent(t, rr, "Dashboard")
}

func TestDashboardHandler_sidebarUsesMultiCloudEyebrow(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}
	h := NewDashboardHandler(setupTestRenderer(t), s, testSite(), cais.Config{}, setupTestViews(t))

	req := inertiaRequest(http.MethodGet, "/dashboard", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	body := rr.Body.String()
	if strings.Contains(body, "FinOps AWS") {
		t.Error("sidebar still says FinOps AWS")
	}
	if !strings.Contains(body, "multi cloud FinOps") {
		t.Errorf("missing multi cloud eyebrow: %s", body)
	}
}

func TestDashboardHandler_pinsSidebarWhileMainScrolls(t *testing.T) {
	h := NewDashboardHandler(setupTestRenderer(t), setupTestStore(t), testSite(), cais.Config{}, setupTestViews(t))

	req := inertiaRequest(http.MethodGet, "/dashboard", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	body := rr.Body.String()
	for _, class := range []string{"md:sticky", "md:top-0", "md:h-screen", "md:self-start"} {
		if !strings.Contains(body, class) {
			t.Errorf("sidebar missing %s", class)
		}
	}
}

func TestDashboardHandler_includesFlashProp(t *testing.T) {
	h := NewDashboardHandler(setupTestRenderer(t), setupTestStore(t), testSite(), cais.Config{}, setupTestViews(t))

	req := inertiaRequest(http.MethodGet, "/dashboard", nil)
	req = flash.WithMessage(req, flash.Message{Kind: "notice", Message: "Welcome back!"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Welcome back!") {
		t.Errorf("flash missing: %s", rr.Body.String())
	}
}

func TestDashboardHandler_pastMonthUsesOverlayNotSQLite(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(finops.SeedAccountEnv, "111111111111")
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}
	tenant, _ := s.FindTenantBySlug(finops.PrimaryTenantSlug)
	accounts, _ := s.ListCloudAccounts(tenant.ID)
	_ = s.ReplaceCostLines(accounts[0].ID, []models.CostLine{{
		Service: "Stored August", MonthlyCents: 999, Source: finops.SourceCE,
		PeriodStart: "2026-08-01", PeriodEnd: "2026-09-01",
	}})
	_ = s.ReplaceFindings(accounts[0].ID, []models.Finding{{
		Kind: finops.FindingCEDenied, Severity: "warning",
	}})

	col := stubMonthCollector{lines: []models.CostLine{{
		Service: "Amazon Lightsail", MonthlyCents: 1947, Source: finops.SourceCE,
		PeriodStart: "2026-07-01", PeriodEnd: "2026-08-01",
	}}}

	h := NewDashboardHandler(setupTestRenderer(t), s, testSite(), cais.Config{}, setupTestViews(t)).
		WithSyncer(syncer.New(s, col))
	h.now = func() time.Time { return time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC) }

	req := inertiaRequest(http.MethodGet, "/dashboard?month=2026-07", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `data-month="2026-07"`) {
		t.Fatal("month")
	}
	if !strings.Contains(rr.Body.String(), "US$ 19,47") {
		t.Fatalf("usd missing: %s", rr.Body.String())
	}
	stored, _ := s.ListCostLines(accounts[0].ID)
	if stored[0].Service != "Stored August" {
		t.Fatalf("sqlite=%#v", stored)
	}
}

func TestDashboardHandler_currentMonthUsesCEOverlay(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(finops.SeedAccountEnv, "111111111111")
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}
	tenant, _ := s.FindTenantBySlug(finops.PrimaryTenantSlug)
	accounts, _ := s.ListCloudAccounts(tenant.ID)
	_ = s.ReplaceFindings(accounts[0].ID, []models.Finding{{
		Kind: finops.FindingCEDenied, Severity: "warning",
	}})

	col := stubMonthCollector{lines: []models.CostLine{{
		Service: "Amazon Lightsail", MonthlyCents: 636, Source: finops.SourceCE,
		PeriodStart: "2026-08-01", PeriodEnd: "2026-09-01",
	}}}
	h := NewDashboardHandler(setupTestRenderer(t), s, testSite(), cais.Config{}, setupTestViews(t)).
		WithSyncer(syncer.New(s, col))
	h.now = func() time.Time { return time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC) }

	req := inertiaRequest(http.MethodGet, "/dashboard", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	body := rr.Body.String()
	if !strings.Contains(body, "US$ 6,36") {
		t.Fatalf("current month missing CE overlay: %s", body)
	}
	if strings.Contains(body, "Nothing synced yet") {
		t.Fatal("empty state despite CE overlay")
	}
}

func TestDashboardHandler_currentMonthIncludesNextMonthForecast(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(finops.SeedAccountEnv, "111111111111")
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}

	col := &stubForecastCollector{cents: 3850}
	h := NewDashboardHandler(setupTestRenderer(t), s, testSite(), cais.Config{}, setupTestViews(t)).
		WithSyncer(syncer.New(s, col))
	h.now = func() time.Time { return time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC) }

	req := inertiaRequest(http.MethodGet, "/dashboard", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "US$ 38,50") {
		t.Fatalf("forecastUSD missing: %s", rr.Body.String())
	}
	if !col.called || !col.period.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("period=%v called=%v", col.period, col.called)
	}
}

func TestDashboardHandler_pastMonthOmitsForecast(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(finops.SeedAccountEnv, "111111111111")
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}

	col := &stubForecastCollector{
		stubMonthCollector: stubMonthCollector{lines: []models.CostLine{{
			Service: "Amazon Lightsail", MonthlyCents: 1947, Source: finops.SourceCE,
			PeriodStart: "2026-07-01", PeriodEnd: "2026-08-01",
		}}},
		cents: 3850,
	}
	h := NewDashboardHandler(setupTestRenderer(t), s, testSite(), cais.Config{}, setupTestViews(t)).
		WithSyncer(syncer.New(s, col))
	h.now = func() time.Time { return time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC) }

	req := inertiaRequest(http.MethodGet, "/dashboard?month=2026-07", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	if col.called {
		t.Fatal("forecast called on past month")
	}
	summary := assertInertiaProp(t, rr, "summary").(map[string]any)
	if summary["forecastUSD"] != nil && summary["forecastUSD"] != "" {
		t.Fatalf("forecastUSD=%v", summary["forecastUSD"])
	}
}

func TestDashboardHandler_zeroForecastOmitsProp(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(finops.SeedAccountEnv, "111111111111")
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}

	col := &stubForecastCollector{cents: 0}
	h := NewDashboardHandler(setupTestRenderer(t), s, testSite(), cais.Config{}, setupTestViews(t)).
		WithSyncer(syncer.New(s, col))
	h.now = func() time.Time { return time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC) }

	req := inertiaRequest(http.MethodGet, "/dashboard", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	summary := assertInertiaProp(t, rr, "summary").(map[string]any)
	if summary["forecastUSD"] != nil && summary["forecastUSD"] != "" {
		t.Fatalf("forecastUSD=%v", summary["forecastUSD"])
	}
}
