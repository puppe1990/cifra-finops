package syncer

import (
	"context"
	"testing"
	"time"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/store"
)

type stubMonthCollector struct {
	stubCollector
	lines []models.CostLine
}

func (s stubMonthCollector) CostForMonth(_ context.Context, _ awsinv.Credentials, _ time.Time) ([]models.CostLine, error) {
	return s.lines, nil
}

func TestSyncer_CostForMonth_doesNotWriteCostLines(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	tid, _ := s.CreateTenant("Demo", "demo")
	accID, _ := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, AWSAccountID: "111111111111", Alias: "p",
		Region: "us-east-1", AuthMode: finops.AuthModeDefaultChain,
	})
	_ = s.ReplaceCostLines(accID, []models.CostLine{{
		Service: "Stored", MonthlyCents: 100, Source: finops.SourceCE,
		PeriodStart: "2026-08-01", PeriodEnd: "2026-09-01",
	}})

	july := []models.CostLine{{
		Service: "Amazon Lightsail", MonthlyCents: 1947, Source: finops.SourceCE,
		PeriodStart: "2026-07-01", PeriodEnd: "2026-08-01",
	}}
	got, err := New(s, stubMonthCollector{lines: july}).CostForMonth(
		context.Background(), tid, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || len(got) != 1 || got[0].MonthlyCents != 1947 {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	stored, _ := s.ListCostLines(accID)
	if len(stored) != 1 || stored[0].Service != "Stored" {
		t.Fatalf("sqlite mutated: %#v", stored)
	}
}

type stubForecastCollector struct {
	stubCollector
	cents  int64
	period time.Time
}

func (s *stubForecastCollector) ForecastForMonth(_ context.Context, _ awsinv.Credentials, period time.Time) (int64, error) {
	s.period = period
	return s.cents, nil
}

func TestSyncer_ForecastForMonth_sumsAccounts(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	tid, _ := s.CreateTenant("Demo", "demo")
	_, _ = s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, AWSAccountID: "111111111111", Alias: "a",
		Region: "us-east-1", AuthMode: finops.AuthModeDefaultChain,
	})
	_, _ = s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, AWSAccountID: "222222222222", Alias: "b",
		Region: "us-east-1", AuthMode: finops.AuthModeDefaultChain,
	})

	col := &stubForecastCollector{cents: 1000}
	sept := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	got, err := New(s, col).ForecastForMonth(context.Background(), tid, sept)
	if err != nil || got != 2000 {
		t.Fatalf("got=%d err=%v", got, err)
	}
	if !col.period.Equal(sept) {
		t.Fatalf("period=%v", col.period)
	}
}

type stubRangeCollector struct {
	stubCollector
	lines []models.CostLine
}

func (s stubRangeCollector) CostByMonth(_ context.Context, _ awsinv.Credentials, _, _ time.Time) ([]models.CostLine, error) {
	return s.lines, nil
}

type stubAnomalyCollector struct {
	stubCollector
	anomalies []awsinv.CostAnomaly
}

func (s stubAnomalyCollector) CostAnomalies(_ context.Context, _ awsinv.Credentials, _, _ time.Time) ([]awsinv.CostAnomaly, error) {
	return s.anomalies, nil
}

func TestSyncer_CostAnomalies_sumsAccounts(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	tid, _ := s.CreateTenant("Demo", "demo")
	_, _ = s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, AWSAccountID: "111111111111", Alias: "a",
		Region: "us-east-1", AuthMode: finops.AuthModeDefaultChain,
	})
	got, err := New(s, stubAnomalyCollector{anomalies: []awsinv.CostAnomaly{{
		ID: "a1", Kind: "ce", Service: "Amazon S3", ImpactCents: 800,
	}}}).CostAnomalies(context.Background(), tid,
		time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC))
	if err != nil || len(got) != 1 || got[0].ID != "a1" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}

func TestSyncer_CostByMonth_sumsAccountsWithoutWrite(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	tid, _ := s.CreateTenant("Demo", "demo")
	accID, _ := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, AWSAccountID: "111111111111", Alias: "p",
		Region: "us-east-1", AuthMode: finops.AuthModeDefaultChain,
	})
	_ = s.ReplaceCostLines(accID, []models.CostLine{{
		Service: "Stored", MonthlyCents: 100, Source: finops.SourceCE,
	}})
	got, err := New(s, stubRangeCollector{lines: []models.CostLine{{
		Service: "Amazon Lightsail", MonthlyCents: 1947, Source: finops.SourceCE,
		PeriodStart: "2026-07-01", PeriodEnd: "2026-08-01",
	}}}).CostByMonth(context.Background(), tid,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || len(got) != 1 || got[0].MonthlyCents != 1947 {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	stored, _ := s.ListCostLines(accID)
	if stored[0].Service != "Stored" {
		t.Fatalf("sqlite mutated: %#v", stored)
	}
}

func TestSyncer_ForecastForMonth_withoutForecaster(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	tid, _ := s.CreateTenant("Demo", "demo")
	got, err := New(s, stubCollector{}).ForecastForMonth(
		context.Background(), tid, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || got != 0 {
		t.Fatalf("got=%d err=%v", got, err)
	}
}
