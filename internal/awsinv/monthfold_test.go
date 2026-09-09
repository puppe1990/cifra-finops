package awsinv

import (
	"testing"
	"time"

	"github.com/puppe1990/cifra-finops/internal/models"
)

func TestFoldMonthlyLines_fillsGapsAndSums(t *testing.T) {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	got := FoldMonthlyLines(from, to, []models.CostLine{
		{Service: "Amazon Lightsail", MonthlyCents: 1947, PeriodStart: "2026-07-01"},
		{Service: "Amazon S3", MonthlyCents: 100, PeriodStart: "2026-08-01"},
		{Service: "Tax", MonthlyCents: 50, PeriodStart: "2026-08-01"},
	})
	if len(got) != 3 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Query != "2026-06" || got[0].Cents != 0 {
		t.Fatalf("june=%#v", got[0])
	}
	if got[1].Query != "2026-07" || got[1].Cents != 1947 {
		t.Fatalf("july=%#v", got[1])
	}
	if got[2].Query != "2026-08" || got[2].Cents != 150 {
		t.Fatalf("aug=%#v", got[2])
	}
	if len(got[2].Lines) != 2 {
		t.Fatalf("aug lines=%d", len(got[2].Lines))
	}
}
