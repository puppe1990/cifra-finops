package handlers

import (
	"sort"

	"github.com/puppe1990/cifra-finops/internal/models"
)

func serviceProps(lines []models.CostLine) []map[string]any {
	type bucket struct {
		cents   int64
		source  string
		details map[string]int64
	}
	byService := map[string]*bucket{}
	var order []string
	for _, line := range lines {
		if line.Service == "" && line.MonthlyCents == 0 {
			continue
		}
		b, ok := byService[line.Service]
		if !ok {
			b = &bucket{details: map[string]int64{}, source: line.Source}
			byService[line.Service] = b
			order = append(order, line.Service)
		}
		b.cents += line.MonthlyCents
		if line.UsageType != "" && line.MonthlyCents > 0 {
			b.details[line.UsageType] += line.MonthlyCents
		}
	}
	sort.Slice(order, func(i, j int) bool {
		if byService[order[i]].cents == byService[order[j]].cents {
			return order[i] < order[j]
		}
		return byService[order[i]].cents > byService[order[j]].cents
	})
	out := make([]map[string]any, 0, len(order))
	for _, name := range order {
		b := byService[name]
		details := usageDetails(b.details, b.source)
		out = append(out, map[string]any{
			"name": name, "cents": b.cents, "usd": formatCost(b.cents, b.source), "details": details,
		})
	}
	return out
}

func usageDetails(sums map[string]int64, source string) []map[string]any {
	names := make([]string, 0, len(sums))
	for n := range sums {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if sums[names[i]] == sums[names[j]] {
			return names[i] < names[j]
		}
		return sums[names[i]] > sums[names[j]]
	})
	out := make([]map[string]any, 0, len(names))
	for _, n := range names {
		out = append(out, map[string]any{
			"name": n, "cents": sums[n], "usd": formatCost(sums[n], source),
		})
	}
	return out
}
