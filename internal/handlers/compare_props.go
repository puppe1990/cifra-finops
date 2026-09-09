package handlers

import (
	"sort"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func MonthDeltaBps(curr, prev int64) (int64, bool) {
	if prev == 0 {
		return 0, false
	}
	return (curr - prev) * 10000 / prev, true
}

func compareMonthRows(months []awsinv.MonthCost, cat *i18n.Catalog, now time.Time) []map[string]any {
	now = time.Date(now.UTC().Year(), now.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	n := len(months)
	var max int64
	for _, m := range months {
		if m.Cents > max {
			max = m.Cents
		}
	}
	out := make([]map[string]any, n)
	for i := 0; i < n; i++ {
		m := months[i]
		pct := 4
		if max > 0 {
			pct = int(m.Cents * 100 / max)
			if pct < 4 && m.Cents > 0 {
				pct = 4
			}
		}
		row := map[string]any{
			"query":   m.Query,
			"label":   ledgerMonthLabel(cat, awsinv.LedgerMonth{Period: m.Period}),
			"cents":   m.Cents,
			"usd":     formatUSD(m.Cents),
			"pct":     pct,
			"current": m.Period.Equal(now),
		}
		if i > 0 {
			if bps, ok := MonthDeltaBps(m.Cents, months[i-1].Cents); ok {
				row["deltaBps"] = bps
				row["deltaUSD"] = formatUSD(m.Cents - months[i-1].Cents)
			}
		}
		out[i] = row
	}
	return out
}

func monthHasSpend(m awsinv.MonthCost) bool {
	if m.Cents > 0 {
		return true
	}
	for _, line := range m.Lines {
		if line.MonthlyCents > 0 {
			return true
		}
	}
	return false
}

func trimLeadingEmptyMonths(months []awsinv.MonthCost) []awsinv.MonthCost {
	i := 0
	for i < len(months) && !monthHasSpend(months[i]) {
		i++
	}
	return months[i:]
}

func compareServiceHistory(months []awsinv.MonthCost) []map[string]any {
	months = trimLeadingEmptyMonths(months)
	names := map[string]bool{}
	for _, m := range months {
		for _, line := range m.Lines {
			if line.Service != "" {
				names[line.Service] = true
			}
		}
	}
	n := len(months)
	list := make([]string, 0, len(names))
	for name := range names {
		list = append(list, name)
	}
	currIdx := n - 1
	sort.Slice(list, func(i, j int) bool {
		ci, cj := serviceMonthCents(months, currIdx, list[i]), serviceMonthCents(months, currIdx, list[j])
		if ci == cj {
			return list[i] < list[j]
		}
		return ci > cj
	})
	out := make([]map[string]any, 0, len(list))
	for _, name := range list {
		cells := make([]map[string]any, 0, n)
		var peak, total int64
		for i := range months {
			cents := serviceMonthCents(months, i, name)
			total += cents
			if cents > peak {
				peak = cents
			}
		}
		if peak == 0 {
			continue
		}
		for i, m := range months {
			cents := serviceMonthCents(months, i, name)
			pct := 0
			if peak > 0 {
				pct = int(cents * 100 / peak)
			}
			cells = append(cells, map[string]any{
				"query": m.Query,
				"cents": cents,
				"usd":   formatUSD(cents),
				"pct":   pct,
			})
		}
		current, previous := int64(0), int64(0)
		if n > 0 {
			current = serviceMonthCents(months, n-1, name)
		}
		if n > 1 {
			previous = serviceMonthCents(months, n-2, name)
		}
		row := map[string]any{
			"name":          name,
			"months":        cells,
			"totalUSD":      formatUSD(total),
			"currentCents":  current,
			"previousCents": previous,
			"currentUSD":    formatUSD(current),
			"previousUSD":   formatUSD(previous),
		}
		if bps, ok := MonthDeltaBps(current, previous); ok {
			row["deltaBps"] = bps
		}
		out = append(out, row)
	}
	return out
}

func serviceMonthCents(months []awsinv.MonthCost, i int, name string) int64 {
	if i < 0 || i >= len(months) {
		return 0
	}
	var sum int64
	for _, line := range months[i].Lines {
		if line.Service == name {
			sum += line.MonthlyCents
		}
	}
	return sum
}

func compareServiceRows(current, previous []models.CostLine) []map[string]any {
	curr := serviceSums(current)
	prev := serviceSums(previous)
	names := make([]string, 0, len(curr)+len(prev))
	seen := map[string]bool{}
	for name := range curr {
		names = append(names, name)
		seen[name] = true
	}
	for name := range prev {
		if !seen[name] {
			names = append(names, name)
		}
	}
	sort.Slice(names, func(i, j int) bool {
		if curr[names[i]] == curr[names[j]] {
			return names[i] < names[j]
		}
		return curr[names[i]] > curr[names[j]]
	})
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		c, p := curr[name], prev[name]
		if c == 0 && p == 0 {
			continue
		}
		row := map[string]any{
			"name":          name,
			"currentCents":  c,
			"previousCents": p,
			"currentUSD":    formatUSD(c),
			"previousUSD":   formatUSD(p),
		}
		if bps, ok := MonthDeltaBps(c, p); ok {
			row["deltaBps"] = bps
		}
		out = append(out, row)
	}
	return out
}

func serviceSums(lines []models.CostLine) map[string]int64 {
	out := map[string]int64{}
	for _, line := range lines {
		if line.Service == "" {
			continue
		}
		out[line.Service] += line.MonthlyCents
	}
	return out
}
