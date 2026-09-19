package handlers

import (
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

// amplifyProps ranks Amplify usage dimensions from most to least expensive.
func amplifyProps(lines []models.CostLine) []map[string]any {
	sums := map[string]int64{}
	source, currency := "", ""
	for _, line := range lines {
		if line.Service != finops.AmplifyService {
			continue
		}
		if source == "" {
			source, currency = line.Source, line.Currency
		}
		sums[finops.AmplifyDimension(line.UsageType)] += line.MonthlyCents
	}
	return usageDetails(sums, source, currency)
}
