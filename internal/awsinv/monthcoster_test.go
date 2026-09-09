package awsinv

import (
	"context"
	"testing"
	"time"

	"github.com/puppe1990/cifra-finops/internal/models"
)

type stubMonthCoster struct{}

func (stubMonthCoster) Collect(context.Context, Credentials) (Inventory, error) {
	return Inventory{}, nil
}

func (stubMonthCoster) CostForMonth(context.Context, Credentials, time.Time) ([]models.CostLine, error) {
	start, end := MonthBounds(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	return []models.CostLine{{
		Service: "Amazon Lightsail", MonthlyCents: 1947,
		PeriodStart: start, PeriodEnd: end,
	}}, nil
}

func TestMonthCoster_julyBoundsOnLines(t *testing.T) {
	var c MonthCoster = stubMonthCoster{}
	lines, err := c.CostForMonth(context.Background(), Credentials{}, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || len(lines) != 1 {
		t.Fatalf("lines=%v err=%v", lines, err)
	}
	if lines[0].PeriodStart != "2026-07-01" || lines[0].PeriodEnd != "2026-08-01" {
		t.Fatalf("period = %s %s", lines[0].PeriodStart, lines[0].PeriodEnd)
	}
}
