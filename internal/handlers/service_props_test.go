package handlers

import (
	"testing"

	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func TestServiceProps_nestsUsageTypesUnderService(t *testing.T) {
	got := serviceProps([]models.CostLine{
		{Service: "Amazon Lightsail", UsageType: "BoxUsage:small_3_0", MonthlyCents: 1900, Source: finops.SourceCE},
		{Service: "Amazon Lightsail", UsageType: "StaticIp", MonthlyCents: 47, Source: finops.SourceCE},
		{Service: "AWS Amplify", UsageType: "BuildMinutes", MonthlyCents: 721, Source: finops.SourceCE},
	})
	if len(got) != 2 {
		t.Fatalf("services = %d, want 2: %#v", len(got), got)
	}
	if got[0]["name"] != "Amazon Lightsail" || got[0]["cents"] != int64(1947) {
		t.Fatalf("lightsail = %#v", got[0])
	}
	details, ok := got[0]["details"].([]map[string]any)
	if !ok || len(details) != 2 {
		t.Fatalf("details = %#v", got[0]["details"])
	}
	if details[0]["name"] != "BoxUsage:small_3_0" || details[0]["cents"] != int64(1900) {
		t.Fatalf("first detail = %#v", details[0])
	}
	if got[1]["name"] != "AWS Amplify" {
		t.Fatalf("second = %#v", got[1])
	}
}

func TestServiceProps_omitsEmptyUsageDetails(t *testing.T) {
	got := serviceProps([]models.CostLine{
		{Service: "Amazon Lightsail", UsageType: "", MonthlyCents: 1200, Source: finops.SourceEstimate},
	})
	if len(got) != 1 {
		t.Fatalf("got = %#v", got)
	}
	if details, _ := got[0]["details"].([]map[string]any); len(details) != 0 {
		t.Fatalf("estimate should have no details: %#v", details)
	}
}
