package awsinv

import (
	"fmt"
	"strings"
	"time"

	"github.com/puppe1990/cifra-finops/internal/costest"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

type resourceInput struct {
	Service string
	Cents   int64
}

func MapLightsailInstance(name, bundleID, region, state string, catalog map[string]int64) models.CloudResource {
	return models.CloudResource{
		Kind:         "lightsail_instance",
		Name:         name,
		Region:       region,
		State:        state,
		MonthlyCents: costest.LightsailMonthlyCents(bundleID, catalog),
		Source:       finops.SourceEstimate,
		ExternalID:   name,
		MetaJSON:     `{"bundle":"` + bundleID + `"}`,
	}
}

func MapStaticIP(name, region, attachedTo string) models.CloudResource {
	return models.CloudResource{
		Kind:         "lightsail_static_ip",
		Name:         name,
		Region:       region,
		State:        ipState(attachedTo),
		MonthlyCents: costest.StaticIPMonthlyCents(attachedTo != ""),
		Source:       finops.SourceEstimate,
		ExternalID:   name,
		MetaJSON:     `{"attached_to":"` + attachedTo + `"}`,
	}
}

func MapS3Bucket(name, region string, bytes int64) models.CloudResource {
	return models.CloudResource{
		Kind:         "s3_bucket",
		Name:         name,
		Region:       region,
		State:        "active",
		MonthlyCents: costest.S3StandardMonthlyCents(bytes),
		Source:       finops.SourceEstimate,
		ExternalID:   name,
	}
}

func EstimateLinesFromResources(items []resourceInput) []models.CostLine {
	start, end := monthBounds(time.Now().UTC())
	grouped := costest.GroupByService(toCostest(items))
	out := make([]models.CostLine, 0, len(grouped))
	for _, g := range grouped {
		out = append(out, models.CostLine{
			Service:      g.Service,
			MonthlyCents: g.MonthlyCents,
			Source:       finops.SourceEstimate,
			PeriodStart:  start,
			PeriodEnd:    end,
		})
	}
	return out
}

func FindingsFromResources(resources []models.CloudResource, ceDenied bool) []models.Finding {
	var out []models.Finding
	if ceDenied {
		out = append(out, models.Finding{
			Kind:     finops.FindingCEDenied,
			Severity: "warning",
		})
	}
	unknownS3 := 0
	for _, r := range resources {
		if r.Kind == "lightsail_static_ip" && r.MonthlyCents > 0 {
			out = append(out, models.Finding{
				Kind:     finops.FindingUnattachedIP,
				Severity: "warning",
				Title:    r.Name,
			})
		}
		if r.Kind == "lightsail_instance" && r.State == "stopped" && r.MonthlyCents > 0 {
			out = append(out, models.Finding{
				Kind:     finops.FindingStoppedBill,
				Severity: "info",
				Title:    r.Name,
			})
		}
		if r.Kind == "s3_bucket" && r.MonthlyCents == 0 {
			unknownS3++
		}
	}
	if unknownS3 > 0 {
		out = append(out, models.Finding{
			Kind:     finops.FindingUnknownS3Size,
			Severity: "info",
			Title:    fmt.Sprintf("%d", unknownS3),
		})
	}
	return out
}

func IsAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not authorized") ||
		strings.Contains(msg, "accessdenied") ||
		strings.Contains(msg, "access denied")
}

func toCostest(items []resourceInput) []costest.Line {
	out := make([]costest.Line, 0, len(items))
	for _, item := range items {
		out = append(out, costest.Line{Service: item.Service, MonthlyCents: item.Cents})
	}
	return out
}

func ipState(attachedTo string) string {
	if attachedTo == "" {
		return "unattached"
	}
	return "attached"
}

func monthBounds(now time.Time) (string, string) {
	return MonthBounds(now)
}

func deniedError(msg string) error {
	return &simpleError{msg: msg}
}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }
