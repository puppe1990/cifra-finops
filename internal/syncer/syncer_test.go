package syncer

import (
	"context"
	"testing"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/store"
)

type stubCollector struct {
	inv awsinv.Inventory
	err error
}

func (s stubCollector) Collect(_ context.Context, _ awsinv.Credentials) (awsinv.Inventory, error) {
	return s.inv, s.err
}

func TestSyncer_persistsInventoryForAccount(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	tid, err := s.CreateTenant("Demo", "demo")
	if err != nil {
		t.Fatal(err)
	}
	accID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID:     tid,
		AWSAccountID: "111111111111",
		Alias:        "principal",
		Region:       "us-east-1",
		AuthMode:     finops.AuthModeDefaultChain,
		IsPrimary:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	col := stubCollector{inv: awsinv.Inventory{
		Source: finops.SourceEstimate,
		Resources: []models.CloudResource{{
			Kind: "lightsail_instance", Name: "web-small", Region: "us-east-1",
			State: "running", MonthlyCents: 1200, Source: finops.SourceEstimate,
			ExternalID: "web-small",
		}},
		Lines: []models.CostLine{{
			Service: "Amazon Lightsail", MonthlyCents: 1200, Source: finops.SourceEstimate,
		}},
		Findings: []models.Finding{{
			Kind: finops.FindingCEDenied, Severity: "warning",
			Title: "Cost Explorer sem permissão", Detail: "ce:GetCostAndUsage denied",
		}},
		Warnings: []string{"ce:GetCostAndUsage denied"},
	}}

	run, err := New(s, col).SyncAccount(context.Background(), accID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != finops.SyncOK {
		t.Fatalf("status = %q, want ok", run.Status)
	}

	resources, err := s.ListResourcesForTenant(tid)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 || resources[0].Name != "web-small" {
		t.Fatalf("resources = %#v", resources)
	}
	lines, err := s.ListCostLines(accID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0].MonthlyCents != 1200 {
		t.Fatalf("lines = %#v", lines)
	}
	findings, err := s.ListFindings(accID)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Kind != finops.FindingCEDenied {
		t.Fatalf("findings = %#v", findings)
	}
}
