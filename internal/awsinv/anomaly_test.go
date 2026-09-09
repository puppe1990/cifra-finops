package awsinv

import (
	"testing"
	"time"

	"github.com/puppe1990/cifra-finops/internal/models"
)

func TestSpikeMonths_flagsFiftyPercentJump(t *testing.T) {
	jul := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	aug := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	got := SpikeMonths([]MonthCost{
		{Query: "2026-07", Period: jul, Cents: 1000},
		{Query: "2026-08", Period: aug, Cents: 2000},
	})
	if len(got) != 1 || got[0].Query != "2026-08" || got[0].Kind != "spike" {
		t.Fatalf("got=%#v", got)
	}
	if got[0].ImpactCents != 1000 {
		t.Fatalf("impact=%d", got[0].ImpactCents)
	}
}

func TestSpikeMonths_ignoresSmallMoves(t *testing.T) {
	jul := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	aug := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	got := SpikeMonths([]MonthCost{
		{Query: "2026-07", Period: jul, Cents: 1000},
		{Query: "2026-08", Period: aug, Cents: 1100},
	})
	if len(got) != 0 {
		t.Fatalf("got=%#v", got)
	}
}

func TestSpikeServices_flagsServiceJump(t *testing.T) {
	got := SpikeServices("2026-08",
		[]models.CostLine{{Service: "Amazon Lightsail", MonthlyCents: 4000}},
		[]models.CostLine{{Service: "Amazon Lightsail", MonthlyCents: 2000}},
	)
	if len(got) != 1 || got[0].Service != "Amazon Lightsail" || got[0].Query != "2026-08" {
		t.Fatalf("got=%#v", got)
	}
}

func TestMapCostAnomaly_fromExplorerFields(t *testing.T) {
	got := MapCostAnomaly("abc", "Amazon S3", "2026-08-10", "2026-08-12", 12.5, 80)
	if got.ID != "abc" || got.Kind != "ce" || got.Service != "Amazon S3" {
		t.Fatalf("got=%#v", got)
	}
	if got.Query != "2026-08" || got.ImpactCents != 1250 {
		t.Fatalf("query/impact=%#v", got)
	}
}

func TestMapCostAnomaly_trimsTimestamp(t *testing.T) {
	got := MapCostAnomaly("abc", "Amazon S3", "2026-08-10T00:00:00Z", "2026-08-12T00:00:00Z", 1, 0)
	if got.Start != "2026-08-10" || got.End != "2026-08-12" || got.Query != "2026-08" {
		t.Fatalf("got=%#v", got)
	}
}
