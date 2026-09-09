package awsinv

import (
	"time"

	"github.com/puppe1990/cifra-finops/internal/models"
)

type MonthCost struct {
	Query  string
	Period time.Time
	Cents  int64
	Lines  []models.CostLine
}

func FoldMonthlyLines(from, to time.Time, lines []models.CostLine) []MonthCost {
	from = time.Date(from.UTC().Year(), from.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	to = time.Date(to.UTC().Year(), to.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	var periods []time.Time
	for period := from; !period.After(to); period = period.AddDate(0, 1, 0) {
		periods = append(periods, period)
	}
	months := make([]MonthCost, len(periods))
	index := make(map[string]*MonthCost, len(periods))
	for i, period := range periods {
		query := period.Format(LedgerMonthLayout)
		months[i] = MonthCost{Query: query, Period: period}
		index[query] = &months[i]
	}
	for _, line := range lines {
		query := line.PeriodStart
		if len(query) >= 7 {
			query = query[:7]
		}
		bucket, ok := index[query]
		if !ok {
			continue
		}
		bucket.Cents += line.MonthlyCents
		bucket.Lines = append(bucket.Lines, line)
	}
	return months
}
