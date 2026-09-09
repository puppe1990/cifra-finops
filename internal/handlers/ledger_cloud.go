package handlers

import (
	"net/url"

	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"

	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func parseLedgerCloud(raw string) string {
	switch raw {
	case finops.ProviderAWS, finops.ProviderHetzner:
		return raw
	default:
		return ""
	}
}

func accountProvider(acc models.CloudAccount) string {
	if acc.Provider == "" {
		return finops.ProviderAWS
	}
	return acc.Provider
}

func filterAccountsByCloud(accounts []models.CloudAccount, cloud string) []models.CloudAccount {
	if cloud == "" {
		return accounts
	}
	var out []models.CloudAccount
	for _, acc := range accounts {
		if accountProvider(acc) == cloud {
			out = append(out, acc)
		}
	}
	return out
}

func accountIDSet(accounts []models.CloudAccount) map[int64]struct{} {
	ids := make(map[int64]struct{}, len(accounts))
	for _, acc := range accounts {
		ids[acc.ID] = struct{}{}
	}
	return ids
}

func filterResourcesByAccounts(resources []models.CloudResource, ids map[int64]struct{}) []models.CloudResource {
	if ids == nil {
		return resources
	}
	var out []models.CloudResource
	for _, r := range resources {
		if _, ok := ids[r.CloudAccountID]; ok {
			out = append(out, r)
		}
	}
	return out
}

func filterFindingsByAccounts(findings []models.Finding, ids map[int64]struct{}) []models.Finding {
	if ids == nil {
		return findings
	}
	var out []models.Finding
	for _, f := range findings {
		if _, ok := ids[f.CloudAccountID]; ok {
			out = append(out, f)
		}
	}
	return out
}

func ledgerHref(cloud, month string) string {
	q := url.Values{}
	if cloud != "" {
		q.Set("cloud", cloud)
	}
	if month != "" {
		q.Set("month", month)
	}
	path := "/dashboard"
	if enc := q.Encode(); enc != "" {
		return path + "?" + enc
	}
	return path
}

func ledgerCloudTabs(accounts []models.CloudAccount, cloud, month string, cat *i18n.Catalog) []map[string]any {
	tabs := []map[string]any{
		cloudTab("all", cat.T("dash.cloud_all"), ledgerHref("", month), cloud == ""),
	}
	have := map[string]bool{}
	for _, acc := range accounts {
		have[accountProvider(acc)] = true
	}
	for _, p := range []string{finops.ProviderAWS, finops.ProviderHetzner} {
		if !have[p] {
			continue
		}
		tabs = append(tabs, cloudTab(p, cat.T("dash.cloud_"+p), ledgerHref(p, month), cloud == p))
	}
	return tabs
}

func cloudTab(id, label, href string, active bool) map[string]any {
	return map[string]any{
		"id": id, "label": label, "href": href, "active": active,
	}
}
