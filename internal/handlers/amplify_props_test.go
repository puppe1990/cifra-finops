package handlers

import (
	"testing"

	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func TestAmplifyProps_groupsByDimensionSortedDesc(t *testing.T) {
	got := amplifyProps([]models.CostLine{
		{Service: finops.AmplifyService, UsageType: "USE1-BuildDuration", MonthlyCents: 900, Source: finops.SourceCE},
		{Service: finops.AmplifyService, UsageType: "EU-BuildDuration", MonthlyCents: 100, Source: finops.SourceCE},
		{Service: finops.AmplifyService, UsageType: "USE1-DataStorage", MonthlyCents: 300, Source: finops.SourceCE},
		{Service: "Amazon Lightsail", UsageType: "StaticIp", MonthlyCents: 5000, Source: finops.SourceCE},
	})
	if len(got) != 2 {
		t.Fatalf("dimensions = %d, want 2: %#v", len(got), got)
	}
	if got[0]["name"] != "BuildDuration" || got[0]["cents"] != int64(1000) || got[0]["usd"] != "US$ 10,00" {
		t.Fatalf("first = %#v", got[0])
	}
	if got[1]["name"] != "DataStorage" || got[1]["cents"] != int64(300) {
		t.Fatalf("second = %#v", got[1])
	}
}

func TestAmplifyProps_equalCentsSortByNameAscending(t *testing.T) {
	got := amplifyProps([]models.CostLine{
		{Service: finops.AmplifyService, UsageType: "USE1-DataStorage", MonthlyCents: 250, Source: finops.SourceCE},
		{Service: finops.AmplifyService, UsageType: "USE1-BuildDuration", MonthlyCents: 250, Source: finops.SourceCE},
	})
	if len(got) != 2 {
		t.Fatalf("dimensions = %d, want 2: %#v", len(got), got)
	}
	if got[0]["name"] != "BuildDuration" || got[1]["name"] != "DataStorage" {
		t.Fatalf("tie order = %#v, want BuildDuration before DataStorage", got)
	}
}

func TestAmplifyProps_emptyWhenNoAmplifyLines(t *testing.T) {
	got := amplifyProps([]models.CostLine{
		{Service: "Amazon Lightsail", MonthlyCents: 1200, Source: finops.SourceEstimate},
	})
	if len(got) != 0 {
		t.Fatalf("got = %#v, want empty", got)
	}
}
