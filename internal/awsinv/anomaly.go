package awsinv

import (
	"time"

	"github.com/puppe1990/cifra-finops/internal/models"
)

func AnomalyWindow(now time.Time) (time.Time, time.Time) {
	to := now.UTC()
	return to.AddDate(0, 0, -AnomalyLookbackDays), to
}

const SpikeBps int64 = 5000
const SpikeMinCents int64 = 100
const AnomalyLookbackDays = 90

type CostAnomaly struct {
	ID          string
	Kind        string
	Service     string
	Query       string
	Start       string
	End         string
	ImpactCents int64
	Score       float64
}

func isSpike(curr, prev int64) bool {
	if prev <= 0 {
		return false
	}
	delta := curr - prev
	if delta < SpikeMinCents {
		return false
	}
	return delta*10000/prev >= SpikeBps
}

func SpikeMonths(months []MonthCost) []CostAnomaly {
	var out []CostAnomaly
	for i := 1; i < len(months); i++ {
		curr, prev := months[i], months[i-1]
		if !isSpike(curr.Cents, prev.Cents) {
			continue
		}
		out = append(out, CostAnomaly{
			Kind:        "spike",
			Query:       curr.Query,
			Start:       curr.Period.Format("2006-01-02"),
			ImpactCents: curr.Cents - prev.Cents,
		})
	}
	return out
}

func SpikeServices(query string, current, previous []models.CostLine) []CostAnomaly {
	curr := map[string]int64{}
	prev := map[string]int64{}
	for _, line := range current {
		curr[line.Service] += line.MonthlyCents
	}
	for _, line := range previous {
		prev[line.Service] += line.MonthlyCents
	}
	var out []CostAnomaly
	for service, cents := range curr {
		if service == "" || !isSpike(cents, prev[service]) {
			continue
		}
		out = append(out, CostAnomaly{
			Kind:        "spike",
			Service:     service,
			Query:       query,
			ImpactCents: cents - prev[service],
		})
	}
	return out
}

func dateOnly(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func MapCostAnomaly(id, service, start, end string, impactUSD, score float64) CostAnomaly {
	start = dateOnly(start)
	end = dateOnly(end)
	query := start
	if len(query) >= 7 {
		query = query[:7]
	}
	return CostAnomaly{
		ID:          id,
		Kind:        "ce",
		Service:     service,
		Query:       query,
		Start:       start,
		End:         end,
		ImpactCents: int64(impactUSD*100 + 0.5),
		Score:       score,
	}
}
