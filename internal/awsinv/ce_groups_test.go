package awsinv

import (
	"testing"

	"github.com/puppe1990/cifra-finops/internal/finops"
)

func TestCEGroupLine_splitsServiceAndUsageType(t *testing.T) {
	got := CEGroupLine([]string{"Amazon Lightsail", "BoxUsage:small_3_0"}, 1947, "2026-07-01", "2026-08-01")
	if got.Service != "Amazon Lightsail" {
		t.Fatalf("service = %q", got.Service)
	}
	if got.UsageType != "BoxUsage:small_3_0" {
		t.Fatalf("usage = %q", got.UsageType)
	}
	if got.MonthlyCents != 1947 || got.Source != finops.SourceCE {
		t.Fatalf("line = %#v", got)
	}
	if got.PeriodStart != "2026-07-01" || got.PeriodEnd != "2026-08-01" {
		t.Fatalf("period = %s %s", got.PeriodStart, got.PeriodEnd)
	}
}

func TestCEGroupLine_serviceOnly(t *testing.T) {
	got := CEGroupLine([]string{"Tax"}, 391, "2026-08-01", "2026-09-01")
	if got.Service != "Tax" || got.UsageType != "" {
		t.Fatalf("line = %#v", got)
	}
}

func TestCEGroupLine_dropsNoUsageType(t *testing.T) {
	got := CEGroupLine([]string{"Tax", "NoUsageType"}, 391, "2026-08-01", "2026-09-01")
	if got.UsageType != "" {
		t.Fatalf("usage = %q", got.UsageType)
	}
}
