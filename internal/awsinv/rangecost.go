package awsinv

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"

	"github.com/puppe1990/cifra-finops/internal/models"
)

func collectCostByMonth(ctx context.Context, cfg aws.Config, from, to time.Time) ([]models.CostLine, error) {
	start, _ := MonthBounds(from)
	_, end := MonthBounds(to)
	out, err := costexplorer.NewFromConfig(cfg).GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(start),
			End:   aws.String(end),
		},
		Granularity: cetypes.GranularityMonthly,
		Metrics:     []string{"UnblendedCost"},
		GroupBy: []cetypes.GroupDefinition{
			{Type: cetypes.GroupDefinitionTypeDimension, Key: aws.String("SERVICE")},
		},
	})
	if err != nil {
		return nil, err
	}
	var lines []models.CostLine
	for _, result := range out.ResultsByTime {
		pStart, pEnd := start, end
		if result.TimePeriod != nil {
			pStart = aws.ToString(result.TimePeriod.Start)
			pEnd = aws.ToString(result.TimePeriod.End)
		}
		for _, group := range result.Groups {
			amount := group.Metrics["UnblendedCost"]
			cents := parseUSDCents(aws.ToString(amount.Amount))
			if cents == 0 && len(group.Keys) == 0 {
				continue
			}
			lines = append(lines, CEGroupLine(group.Keys, cents, pStart, pEnd))
		}
	}
	return lines, nil
}
