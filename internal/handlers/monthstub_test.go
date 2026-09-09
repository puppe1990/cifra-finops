package handlers

import (
	"context"
	"time"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/models"
)

type stubMonthCollector struct {
	lines []models.CostLine
	err   error
}

func (s stubMonthCollector) Collect(_ context.Context, _ awsinv.Credentials) (awsinv.Inventory, error) {
	return awsinv.Inventory{}, nil
}

func (s stubMonthCollector) CostForMonth(_ context.Context, _ awsinv.Credentials, _ time.Time) ([]models.CostLine, error) {
	return s.lines, s.err
}

type stubForecastCollector struct {
	stubMonthCollector
	cents  int64
	period time.Time
	called bool
}

func (s *stubForecastCollector) ForecastForMonth(_ context.Context, _ awsinv.Credentials, period time.Time) (int64, error) {
	s.called = true
	s.period = period
	return s.cents, s.err
}

type stubRangeCollector struct {
	stubMonthCollector
	rangeLines []models.CostLine
}

func (s stubRangeCollector) CostByMonth(_ context.Context, _ awsinv.Credentials, _, _ time.Time) ([]models.CostLine, error) {
	return s.rangeLines, s.err
}
