package awsinv

import (
	"testing"

	"github.com/puppe1990/cifra-finops/internal/costest"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func TestMapLightsailInstance_usesBundlePrice(t *testing.T) {
	got := MapLightsailInstance("web-small", "small_3_0", "us-east-1", "running", costest.DefaultLightsailCatalog())
	if got.MonthlyCents != 1200 {
		t.Fatalf("cents = %d, want 1200", got.MonthlyCents)
	}
	if got.Kind != "lightsail_instance" {
		t.Fatalf("kind = %q", got.Kind)
	}
	if got.Source != finops.SourceEstimate {
		t.Fatalf("source = %q", got.Source)
	}
}

func TestMapStaticIP_idleCosts(t *testing.T) {
	got := MapStaticIP("orphan-ip", "us-east-1", "")
	if got.MonthlyCents != 300 {
		t.Fatalf("idle IP = %d, want 300", got.MonthlyCents)
	}
}

func TestEstimateLinesFromResources_groupsServices(t *testing.T) {
	lines := EstimateLinesFromResources([]resourceInput{
		{Service: "Amazon Lightsail", Cents: 1200},
		{Service: "Amazon Lightsail", Cents: 300},
		{Service: "Amazon Simple Storage Service", Cents: 2},
	})
	if len(lines) != 2 {
		t.Fatalf("lines = %#v", lines)
	}
	if lines[0].MonthlyCents != 1500 {
		t.Fatalf("lightsail total = %d", lines[0].MonthlyCents)
	}
}

func TestFindingsFromResources_groupsUnknownS3(t *testing.T) {
	got := FindingsFromResources([]models.CloudResource{
		{Kind: "s3_bucket", Name: "a", MonthlyCents: 0},
		{Kind: "s3_bucket", Name: "b", MonthlyCents: 0},
	}, false)
	if len(got) != 1 || got[0].Kind != finops.FindingUnknownS3Size {
		t.Fatalf("findings = %#v", got)
	}
}

func TestIsAccessDenied_detectsAWSMessage(t *testing.T) {
	if !IsAccessDenied(deniedError("not authorized to perform: ce:GetCostAndUsage")) {
		t.Fatal("expected access denied")
	}
	if IsAccessDenied(deniedError("timeout")) {
		t.Fatal("timeout should not look like access denied")
	}
}
