package handlers

import (
	"testing"
	"time"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/finops"
	appi18n "github.com/puppe1990/cifra-finops/internal/i18n"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func TestMonthSpend_keepsHetznerLinesWhenCEOverlay(t *testing.T) {
	s := setupTestStore(t)
	tid, err := s.CreateTenant("Demo", "demo")
	if err != nil {
		t.Fatal(err)
	}
	awsID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, Provider: finops.ProviderAWS, AWSAccountID: "111111111111",
		Alias: "aws", Region: "us-east-1", AuthMode: finops.AuthModeDefaultChain,
	})
	if err != nil {
		t.Fatal(err)
	}
	hzID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, Provider: finops.ProviderHetzner, AWSAccountID: "hz:prod",
		Alias: "prod", Region: "fsn1", AuthMode: finops.AuthModeAPIToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = awsID
	if err := s.ReplaceCostLines(hzID, []models.CostLine{{
		Service: "Cloud Server", MonthlyCents: 349, Source: finops.SourceHetzner,
	}}); err != nil {
		t.Fatal(err)
	}
	accounts, err := s.ListCloudAccounts(tid)
	if err != nil {
		t.Fatal(err)
	}
	overlay := []models.CostLine{{
		Service: "Amazon Lightsail", MonthlyCents: 1200, Source: finops.SourceCE,
	}}
	lm := awsinv.LedgerMonth{IsCurrent: true, Period: time.Now().UTC()}
	monthly, lines, _, err := monthSpend(s, accounts, nil, appi18n.DefaultCatalog(), lm, overlay)
	if err != nil {
		t.Fatal(err)
	}
	if monthly != 1549 {
		t.Fatalf("monthly = %d, want 1549", monthly)
	}
	if !hasCostService(lines, "Cloud Server", 349) || !hasCostService(lines, "Amazon Lightsail", 1200) {
		t.Fatalf("lines = %#v", lines)
	}
}

func TestBuildTenantView_formatsHetznerAsEuro(t *testing.T) {
	s := setupTestStore(t)
	tid, err := s.CreateTenant("Demo", "demo")
	if err != nil {
		t.Fatal(err)
	}
	hzID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, Provider: finops.ProviderHetzner, AWSAccountID: "hz:prod",
		Alias: "prod", Region: "fsn1", AuthMode: finops.AuthModeAPIToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceCostLines(hzID, []models.CostLine{{
		Service: "Cloud Server", MonthlyCents: 349, Source: finops.SourceHetzner,
	}}); err != nil {
		t.Fatal(err)
	}
	view, err := buildTenantView(s, tid, appi18n.DefaultCatalog(), awsinv.LedgerMonth{IsCurrent: true, Period: time.Now().UTC()}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if view.Summary["monthlyEUR"] != "€ 3,49" {
		t.Fatalf("monthlyEUR = %#v", view.Summary["monthlyEUR"])
	}
	if usd, _ := view.Summary["monthlyUSD"].(string); usd != "" {
		t.Fatalf("monthlyUSD should be empty for hetzner-only, got %q", usd)
	}
	if len(view.Services) != 1 {
		t.Fatalf("services = %#v", view.Services)
	}
	if view.Services[0]["usd"] != "€ 3,49" {
		t.Fatalf("service money = %#v", view.Services[0])
	}
}

func hasCostService(lines []models.CostLine, service string, cents int64) bool {
	for _, l := range lines {
		if l.Service == service && l.MonthlyCents == cents {
			return true
		}
	}
	return false
}
