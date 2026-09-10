package awsinv

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/aws/aws-sdk-go-v2/service/lightsail"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/puppe1990/cifra-finops/internal/costest"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

type Live struct{}

func NewLive() *Live { return &Live{} }

func (l *Live) Collect(ctx context.Context, creds Credentials) (Inventory, error) {
	cfg, err := loadAWS(ctx, creds)
	if err != nil {
		return Inventory{}, err
	}

	inv := Inventory{Source: finops.SourceEstimate}
	catalog := costest.DefaultLightsailCatalog()

	if id, err := callerAccount(ctx, cfg); err != nil {
		if cerr := credentialCollectError(err); cerr != nil {
			return Inventory{}, cerr
		}
		inv.Warnings = append(inv.Warnings, "sts: "+err.Error())
	} else if creds.AccountID == "" {
		creds.AccountID = id
	}

	ceLines, ceErr := collectCostExplorer(ctx, cfg, time.Now().UTC())
	ceDenied := ceErr != nil && IsAccessDenied(ceErr)
	if ceErr != nil {
		inv.Warnings = append(inv.Warnings, "ce: "+ceErr.Error())
	} else if len(ceLines) > 0 {
		inv.Lines = ceLines
		inv.Source = finops.SourceCE
	}

	if live, err := lightsailCatalog(ctx, cfg); err != nil {
		inv.Warnings = append(inv.Warnings, "lightsail bundles: "+err.Error())
	} else if len(live) > 0 {
		catalog = live
	}

	resources, warns := collectInventory(ctx, cfg, catalog)
	inv.Resources = resources
	inv.Warnings = append(inv.Warnings, warns...)

	if len(inv.Lines) == 0 {
		inv.Lines = EstimateLinesFromResources(resourceInputs(resources))
	}
	inv.Findings = FindingsFromResources(resources, ceDenied || ceErr != nil)
	return inv, nil
}

func loadAWS(ctx context.Context, creds Credentials) (aws.Config, error) {
	region := creds.Region
	if region == "" {
		region = finops.DefaultRegion
	}
	opts := []func(*config.LoadOptions) error{config.WithRegion(region)}
	if creds.Mode == finops.AuthModeAccessKeys {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(creds.AccessKeyID, creds.SecretAccessKey, ""),
		))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("aws config: %w", err)
	}
	return cfg, nil
}

func callerAccount(ctx context.Context, cfg aws.Config) (string, error) {
	out, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.Account), nil
}

func (l *Live) CostForMonth(ctx context.Context, creds Credentials, period time.Time) ([]models.CostLine, error) {
	cfg, err := loadAWS(ctx, creds)
	if err != nil {
		return nil, err
	}
	return collectCostExplorer(ctx, cfg, period)
}

func (l *Live) ForecastForMonth(ctx context.Context, creds Credentials, period time.Time) (int64, error) {
	cfg, err := loadAWS(ctx, creds)
	if err != nil {
		return 0, err
	}
	return collectCostForecast(ctx, cfg, period, time.Now().UTC())
}

func (l *Live) CostByMonth(ctx context.Context, creds Credentials, from, to time.Time) ([]models.CostLine, error) {
	cfg, err := loadAWS(ctx, creds)
	if err != nil {
		return nil, err
	}
	return collectCostByMonth(ctx, cfg, from, to)
}

func (l *Live) CostAnomalies(ctx context.Context, creds Credentials, from, to time.Time) ([]CostAnomaly, error) {
	cfg, err := loadAWS(ctx, creds)
	if err != nil {
		return nil, err
	}
	return collectCostAnomalies(ctx, cfg, from, to)
}

var _ MonthCoster = (*Live)(nil)
var _ CostForecaster = (*Live)(nil)
var _ RangeCoster = (*Live)(nil)
var _ AnomalyFinder = (*Live)(nil)

func collectCostExplorer(ctx context.Context, cfg aws.Config, period time.Time) ([]models.CostLine, error) {
	start, end := MonthBounds(period)
	out, err := costexplorer.NewFromConfig(cfg).GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(start),
			End:   aws.String(end),
		},
		Granularity: cetypes.GranularityMonthly,
		Metrics:     []string{"UnblendedCost"},
		GroupBy: []cetypes.GroupDefinition{
			{Type: cetypes.GroupDefinitionTypeDimension, Key: aws.String("SERVICE")},
			{Type: cetypes.GroupDefinitionTypeDimension, Key: aws.String("USAGE_TYPE")},
		},
	})
	if err != nil {
		return nil, err
	}
	var lines []models.CostLine
	for _, result := range out.ResultsByTime {
		for _, group := range result.Groups {
			amount := group.Metrics["UnblendedCost"]
			cents := parseUSDCents(aws.ToString(amount.Amount))
			if cents == 0 && len(group.Keys) == 0 {
				continue
			}
			lines = append(lines, CEGroupLine(group.Keys, cents, start, end))
		}
	}
	return lines, nil
}

func collectCostForecast(ctx context.Context, cfg aws.Config, period, now time.Time) (int64, error) {
	monthStart, monthEnd := MonthBounds(period)
	out, err := costexplorer.NewFromConfig(cfg).GetCostForecast(ctx, &costexplorer.GetCostForecastInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(ForecastAPIStart(monthStart, now)),
			End:   aws.String(monthEnd),
		},
		Granularity: cetypes.GranularityMonthly,
		Metric:      cetypes.MetricUnblendedCost,
	})
	if err != nil {
		return 0, err
	}
	buckets := make([]ForecastBucket, 0, len(out.ForecastResultsByTime))
	for _, r := range out.ForecastResultsByTime {
		start := ""
		if r.TimePeriod != nil {
			start = aws.ToString(r.TimePeriod.Start)
		}
		buckets = append(buckets, ForecastBucket{Start: start, Amount: aws.ToString(r.MeanValue)})
	}
	total := ""
	if out.Total != nil {
		total = aws.ToString(out.Total.Amount)
	}
	return ForecastCentsForPeriod(monthStart, buckets, total), nil
}

func lightsailCatalog(ctx context.Context, cfg aws.Config) (map[string]int64, error) {
	out, err := lightsail.NewFromConfig(cfg).GetBundles(ctx, &lightsail.GetBundlesInput{})
	if err != nil {
		return nil, err
	}
	catalog := map[string]int64{}
	for _, b := range out.Bundles {
		id := aws.ToString(b.BundleId)
		if id == "" || b.Price == nil {
			continue
		}
		catalog[id] = int64(*b.Price * 100)
	}
	return catalog, nil
}

func collectInventory(ctx context.Context, cfg aws.Config, catalog map[string]int64) ([]models.CloudResource, []string) {
	var resources []models.CloudResource
	var warns []string

	ls := lightsail.NewFromConfig(cfg)
	if inst, err := ls.GetInstances(ctx, &lightsail.GetInstancesInput{}); err != nil {
		warns = append(warns, "lightsail instances: "+err.Error())
	} else {
		for _, inst := range inst.Instances {
			region := ""
			if inst.Location != nil {
				region = string(inst.Location.RegionName)
			}
			state := ""
			if inst.State != nil {
				state = aws.ToString(inst.State.Name)
			}
			resources = append(resources, MapLightsailInstance(
				aws.ToString(inst.Name),
				aws.ToString(inst.BundleId),
				region,
				state,
				catalog,
			))
		}
	}

	if ips, err := ls.GetStaticIps(ctx, &lightsail.GetStaticIpsInput{}); err != nil {
		warns = append(warns, "lightsail ips: "+err.Error())
	} else {
		for _, ip := range ips.StaticIps {
			region := ""
			if ip.Location != nil {
				region = string(ip.Location.RegionName)
			}
			resources = append(resources, MapStaticIP(
				aws.ToString(ip.Name),
				region,
				aws.ToString(ip.AttachedTo),
			))
		}
	}

	s3c := s3.NewFromConfig(cfg)
	buckets, err := s3c.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		warns = append(warns, "s3: "+err.Error())
		return resources, warns
	}
	for _, b := range buckets.Buckets {
		name := aws.ToString(b.Name)
		region := finops.DefaultRegion
		if loc, err := s3c.GetBucketLocation(ctx, &s3.GetBucketLocationInput{Bucket: b.Name}); err == nil {
			if loc.LocationConstraint != "" {
				region = string(loc.LocationConstraint)
			}
		}
		resources = append(resources, MapS3Bucket(name, region, 0))
	}
	return resources, warns
}

func resourceInputs(resources []models.CloudResource) []resourceInput {
	out := make([]resourceInput, 0, len(resources))
	for _, r := range resources {
		out = append(out, resourceInput{Service: serviceForKind(r.Kind), Cents: r.MonthlyCents})
	}
	return out
}

func serviceForKind(kind string) string {
	switch kind {
	case "s3_bucket":
		return "Amazon Simple Storage Service"
	default:
		return "Amazon Lightsail"
	}
}

func parseUSDCents(amount string) int64 {
	if amount == "" {
		return 0
	}
	var dollars float64
	if _, err := fmt.Sscanf(amount, "%f", &dollars); err != nil {
		return 0
	}
	return int64(dollars*100 + 0.5)
}
