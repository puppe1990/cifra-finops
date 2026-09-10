package store

import (
	"testing"

	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func TestReplaceCostLines_persistsCurrency(t *testing.T) {
	s := newTestStore(t)
	tid, err := s.CreateTenant("Demo", "demo")
	if err != nil {
		t.Fatal(err)
	}
	accID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, Provider: finops.ProviderHetzner, AWSAccountID: "hz:prod",
		Alias: "prod", Region: "fsn1", AuthMode: finops.AuthModeAPIToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceCostLines(accID, []models.CostLine{{
		Service: "Cloud Server", UsageType: "cx33", MonthlyCents: 999,
		Source: finops.SourceHetzner, Currency: "USD",
	}}); err != nil {
		t.Fatal(err)
	}
	lines, err := s.ListCostLines(accID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0].Currency != "USD" || lines[0].MonthlyCents != 999 {
		t.Fatalf("lines = %#v", lines)
	}
}

func TestReplaceResources_persistsCurrency(t *testing.T) {
	s := newTestStore(t)
	tid, err := s.CreateTenant("Demo", "demo-r")
	if err != nil {
		t.Fatal(err)
	}
	accID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, Provider: finops.ProviderHetzner, AWSAccountID: "hz:prod",
		Alias: "prod", Region: "fsn1", AuthMode: finops.AuthModeAPIToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceResources(accID, []models.CloudResource{{
		Kind: "hetzner_server", Name: "gestaobem-cx33", Region: "fsn1",
		State: "running", MonthlyCents: 999, Source: finops.SourceHetzner,
		ExternalID: "1", Currency: "USD",
	}}); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListResources(accID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Currency != "USD" {
		t.Fatalf("resources = %#v", got)
	}
}
