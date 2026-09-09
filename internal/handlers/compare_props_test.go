package handlers

import (
	"testing"
	"time"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	appi18n "github.com/puppe1990/cifra-finops/internal/i18n"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func TestMonthDeltaBps(t *testing.T) {
	bps, ok := MonthDeltaBps(3215, 5228)
	if !ok || bps != (3215-5228)*10000/5228 {
		t.Fatalf("bps=%d ok=%v", bps, ok)
	}
	if _, ok := MonthDeltaBps(100, 0); ok {
		t.Fatal("prev zero should be skipped")
	}
}

func TestCompareMonthRows_newestFirstWithDelta(t *testing.T) {
	jul := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	aug := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	rows := compareMonthRows([]awsinv.MonthCost{
		{Query: "2026-07", Period: jul, Cents: 5228},
		{Query: "2026-08", Period: aug, Cents: 3215},
	}, appi18n.DefaultCatalog(), aug)
	if len(rows) != 2 {
		t.Fatalf("len=%d", len(rows))
	}
	if rows[0]["query"] != "2026-08" || rows[0]["usd"] != "US$ 32,15" {
		t.Fatalf("current=%v", rows[0])
	}
	if rows[0]["current"] != true {
		t.Fatal("august should be current")
	}
	want := int64((3215 - 5228) * 10000 / 5228)
	if rows[0]["deltaBps"] != want {
		t.Fatalf("deltaBps=%v want %d", rows[0]["deltaBps"], want)
	}
	if rows[1]["query"] != "2026-07" || rows[1]["deltaBps"] != nil {
		t.Fatalf("july=%v", rows[1])
	}
}

func TestCompareServiceHistory_seriesOldestFirst(t *testing.T) {
	jul := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	aug := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	rows := compareServiceHistory([]awsinv.MonthCost{
		{Query: "2026-07", Period: jul, Lines: []models.CostLine{
			{Service: "Amazon Lightsail", MonthlyCents: 3220},
			{Service: "Amazon S3", MonthlyCents: 100},
		}},
		{Query: "2026-08", Period: aug, Lines: []models.CostLine{
			{Service: "Amazon Lightsail", MonthlyCents: 1983},
			{Service: "Amazon S3", MonthlyCents: 200},
		}},
	})
	if len(rows) != 2 {
		t.Fatalf("len=%d", len(rows))
	}
	top := rows[0]
	if top["name"] != "Amazon Lightsail" || top["currentUSD"] != "US$ 19,83" {
		t.Fatalf("top=%v", top)
	}
	series, _ := top["months"].([]map[string]any)
	if len(series) != 2 || series[0]["query"] != "2026-07" || series[0]["cents"] != int64(3220) {
		t.Fatalf("series=%v", series)
	}
	if series[1]["query"] != "2026-08" || series[1]["usd"] != "US$ 19,83" {
		t.Fatalf("aug=%v", series[1])
	}
	want := int64((1983 - 3220) * 10000 / 3220)
	if top["deltaBps"] != want {
		t.Fatalf("deltaBps=%v want %d", top["deltaBps"], want)
	}
}

func TestCompareServiceHistory_dropsLeadingEmptyMonths(t *testing.T) {
	jun := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	jul := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	aug := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	rows := compareServiceHistory([]awsinv.MonthCost{
		{Query: "2026-06", Period: jun, Cents: 0},
		{Query: "2026-07", Period: jul, Cents: 3220, Lines: []models.CostLine{
			{Service: "Amazon Lightsail", MonthlyCents: 3220},
		}},
		{Query: "2026-08", Period: aug, Cents: 1983, Lines: []models.CostLine{
			{Service: "Amazon Lightsail", MonthlyCents: 1983},
		}},
	})
	series, _ := rows[0]["months"].([]map[string]any)
	if len(series) != 2 || series[0]["query"] != "2026-07" {
		t.Fatalf("series=%v", series)
	}
}

func TestCompareServiceRows_pairsCurrentAndPrevious(t *testing.T) {
	rows := compareServiceRows(
		[]models.CostLine{
			{Service: "Amazon Lightsail", MonthlyCents: 1983},
			{Service: "Amazon S3", MonthlyCents: 164},
		},
		[]models.CostLine{
			{Service: "Amazon Lightsail", MonthlyCents: 3220},
		},
	)
	if len(rows) != 2 {
		t.Fatalf("len=%d", len(rows))
	}
	if rows[0]["name"] != "Amazon Lightsail" {
		t.Fatalf("top=%v", rows[0])
	}
	if rows[0]["currentUSD"] != "US$ 19,83" || rows[0]["previousUSD"] != "US$ 32,20" {
		t.Fatalf("lightsail=%v", rows[0])
	}
	if rows[1]["name"] != "Amazon S3" || rows[1]["previousCents"] != int64(0) {
		t.Fatalf("s3=%v", rows[1])
	}
}
