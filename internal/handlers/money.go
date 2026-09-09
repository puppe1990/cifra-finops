package handlers

import (
	"fmt"

	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"

	"github.com/puppe1990/cifra-finops/internal/finops"
	appi18n "github.com/puppe1990/cifra-finops/internal/i18n"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func formatUSD(cents int64) string {
	return formatMoney(cents, "USD")
}

func formatEUR(cents int64) string {
	return formatMoney(cents, "EUR")
}

func formatMoney(cents int64, currency string) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	switch currency {
	case "EUR":
		return fmt.Sprintf("%s€ %d,%02d", sign, cents/100, cents%100)
	default:
		return fmt.Sprintf("%sUS$ %d,%02d", sign, cents/100, cents%100)
	}
}

func formatCost(cents int64, source string) string {
	if source == finops.SourceHetzner {
		return formatEUR(cents)
	}
	return formatUSD(cents)
}

func catalogOrEN(cat *i18n.Catalog) *i18n.Catalog {
	if cat != nil {
		return cat
	}
	return appi18n.DefaultCatalog()
}

func resourceKindLabel(kind string, cat *i18n.Catalog) string {
	cat = catalogOrEN(cat)
	key := "kind." + kind
	if label := cat.T(key); label != key {
		return label
	}
	return kind
}

func translateFinding(cat *i18n.Catalog, f models.Finding) (string, string) {
	cat = catalogOrEN(cat)
	switch f.Kind {
	case finops.FindingCEDenied:
		return cat.T("finding.ce_denied.title"), cat.T("finding.ce_denied.detail")
	case finops.FindingUnattachedIP:
		return cat.T("finding.unattached_ip.title", f.Title), cat.T("finding.unattached_ip.detail")
	case finops.FindingStoppedBill:
		return cat.T("finding.stopped_bill.title", f.Title), cat.T("finding.stopped_bill.detail")
	case finops.FindingUnknownS3Size:
		return cat.T("finding.unknown_s3.title", f.Title), cat.T("finding.unknown_s3.detail")
	case finops.FindingUnattachedVolume:
		return cat.T("finding.unattached_volume.title", f.Title), cat.T("finding.unattached_volume.detail")
	default:
		return f.Title, f.Detail
	}
}
