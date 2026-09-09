package awsinv

import (
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func CEGroupLine(keys []string, cents int64, start, end string) models.CostLine {
	line := models.CostLine{
		MonthlyCents: cents,
		Source:       finops.SourceCE,
		PeriodStart:  start,
		PeriodEnd:    end,
	}
	if len(keys) > 0 {
		line.Service = keys[0]
	}
	if len(keys) > 1 && keys[1] != "NoUsageType" {
		line.UsageType = keys[1]
	}
	return line
}
