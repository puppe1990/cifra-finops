package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/seed"
)

func TestDashboardHandler_ledgerTabsFilterByCloud(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}
	tenant, err := s.FindTenantBySlug(finops.PrimaryTenantSlug)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tenant.ID, Provider: finops.ProviderAWS, AWSAccountID: "111111111111",
		Alias: "aws", Region: "us-east-1", AuthMode: finops.AuthModeDefaultChain,
	}); err != nil {
		t.Fatal(err)
	}
	hzID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tenant.ID, Provider: finops.ProviderHetzner, AWSAccountID: "hz:prod",
		Alias: "prod", Region: "fsn1", AuthMode: finops.AuthModeAPIToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	accounts, err := s.ListCloudAccounts(tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, acc := range accounts {
		if acc.Provider == finops.ProviderHetzner {
			if err := s.ReplaceCostLines(acc.ID, []models.CostLine{{
				Service: "Cloud Server", MonthlyCents: 349, Source: finops.SourceHetzner,
			}}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := s.ReplaceCostLines(acc.ID, []models.CostLine{{
			Service: "Amazon Lightsail", MonthlyCents: 1200, Source: finops.SourceCE,
		}}); err != nil {
			t.Fatal(err)
		}
	}
	_ = hzID

	h := NewDashboardHandler(setupTestRenderer(t), s, testSite(), cais.Config{}, setupTestViews(t))

	all := getDashboard(t, h, uid, "/dashboard")
	if !strings.Contains(all, `role="tablist"`) {
		t.Fatalf("missing tablist: %s", all)
	}
	if !strings.Contains(all, ">All<") || !strings.Contains(all, ">AWS<") || !strings.Contains(all, ">Hetzner<") {
		t.Fatalf("missing cloud tabs: %s", all)
	}
	if !strings.Contains(all, `href="/dashboard?cloud=aws"`) || !strings.Contains(all, `href="/dashboard?cloud=hetzner"`) {
		t.Fatalf("missing tab hrefs: %s", all)
	}
	if !strings.Contains(all, "Amazon Lightsail") || !strings.Contains(all, "Cloud Server") {
		t.Fatalf("geral should list both clouds: %s", all)
	}
	if !strings.Contains(all, "US$ 12,00") || !strings.Contains(all, "€ 3,49") {
		t.Fatalf("geral should show both currencies: %s", all)
	}

	hz := getDashboard(t, h, uid, "/dashboard?cloud=hetzner")
	if !strings.Contains(hz, "Cloud Server") {
		t.Fatalf("hetzner tab missing server: %s", hz)
	}
	if strings.Contains(hz, "Amazon Lightsail") {
		t.Fatal("hetzner tab still shows AWS")
	}
	if !strings.Contains(hz, "€ 3,49") {
		t.Fatalf("hetzner tab missing EUR: %s", hz)
	}
	if strings.Contains(hz, "US$ 12,00") {
		t.Fatal("hetzner tab still shows AWS USD")
	}
	if !strings.Contains(hz, "cloud=hetzner") {
		t.Fatalf("month links dropped cloud filter: %s", hz)
	}

	aws := getDashboard(t, h, uid, "/dashboard?cloud=aws")
	if !strings.Contains(aws, "Amazon Lightsail") {
		t.Fatalf("aws tab missing lightsail: %s", aws)
	}
	if strings.Contains(aws, "Cloud Server") {
		t.Fatal("aws tab still shows Hetzner")
	}
}

func getDashboard(t *testing.T, h *DashboardHandler, uid int64, path string) string {
	t.Helper()
	req := inertiaRequest(http.MethodGet, path, nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("%s status=%d body=%s", path, rr.Code, rr.Body.String())
	}
	return rr.Body.String()
}
