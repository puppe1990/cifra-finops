package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/costest"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/store"
	"github.com/puppe1990/cifra-finops/internal/syncer"
)

type DashboardHandler struct {
	store  store.Store
	site   meta.Site
	cfg    cais.Config
	views  *view.Renderer
	syncer *syncer.Syncer
	now    func() time.Time
}

func NewDashboardHandler(_ *view.Renderer, s store.Store, site meta.Site, cfg cais.Config, views *view.Renderer) *DashboardHandler {
	return &DashboardHandler{store: s, site: site, cfg: cfg, views: views}
}

func (h *DashboardHandler) WithSyncer(s *syncer.Syncer) *DashboardHandler {
	h.syncer = s
	return h
}

func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := loadWorkspace(h.store, r)
	props := map[string]any{
		"site":          meta.ForRequest(h.site, r),
		"totalContacts": int64(0),
		"env":           h.cfg.Env,
		"summary":       map[string]any{},
		"services":      []any{},
		"resources":     []any{},
		"findings":      []any{},
		"budgets":       []any{},
		"accounts":      []any{},
		"lastSync":      nil,
		"flash":         map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = map[string]string{msg.Kind: msg.Message}
	}
	if err != nil {
		writePage(w, r, h.views, h.cfg, "app", "dashboard", props, 0)
		return
	}

	now := time.Now().UTC()
	if h.now != nil {
		now = h.now()
	}
	lm := awsinv.ParseLedgerMonth(r.URL.Query().Get("month"), now)
	cat := requestCatalog(r, h.cfg.Locale)

	if h.syncer != nil && lm.IsCurrent {
		resources, _ := h.store.ListResourcesForTenant(ws.Tenant.ID)
		if len(resources) == 0 {
			ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
			_ = h.syncer.SyncTenant(ctx, ws.Tenant.ID)
			cancel()
		}
	}

	var overlay []models.CostLine
	ceDenied := false
	if h.syncer != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		var overlayErr error
		overlay, overlayErr = h.syncer.CostForMonth(ctx, ws.Tenant.ID, lm.Period)
		cancel()
		if overlayErr != nil {
			if awsinv.IsAccessDenied(overlayErr) {
				ceDenied = true
			}
			overlay = nil
		}
	}

	for k, v := range shellProps(h.site, r, h.store, ws) {
		props[k] = v
	}
	view, err := buildTenantView(h.store, ws.Tenant.ID, cat, lm, overlay, ceDenied)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if lm.IsCurrent && h.syncer != nil {
		attachForecast(r.Context(), h.syncer, ws.Tenant.ID, now, cat, view.Summary)
	}
	props["summary"] = view.Summary
	props["services"] = withSpendPct(view.Services)
	props["resources"] = view.Resources
	props["findings"] = view.Findings
	props["budgets"] = view.Budgets
	props["accounts"] = view.Accounts
	props["lastSync"] = view.LastSync
	props["month"] = lm.Query
	props["monthLabel"] = ledgerMonthLabel(cat, lm)
	props["prevMonth"] = lm.Prev
	props["nextMonth"] = lm.Next
	props["isCurrent"] = lm.IsCurrent
	writePage(w, r, h.views, h.cfg, "app", "dashboard", props, 0)
}

type tenantView struct {
	Summary   map[string]any
	Services  []map[string]any
	Resources []map[string]any
	Findings  []map[string]any
	Budgets   []map[string]any
	Accounts  []map[string]any
	LastSync  map[string]any
}

func buildTenantView(s store.Store, tenantID int64, cat *i18n.Catalog, lm awsinv.LedgerMonth, overlay []models.CostLine, ceDenied bool) (tenantView, error) {
	resources, err := s.ListResourcesForTenant(tenantID)
	if err != nil {
		return tenantView{}, err
	}
	accounts, err := s.ListCloudAccounts(tenantID)
	if err != nil {
		return tenantView{}, err
	}
	findings, err := s.ListFindingsForTenant(tenantID)
	if err != nil {
		return tenantView{}, err
	}
	budgets, err := s.ListBudgets(tenantID)
	if err != nil {
		return tenantView{}, err
	}

	monthly, costLines, source, err := monthSpend(s, accounts, resources, cat, lm, overlay)
	if err != nil {
		return tenantView{}, err
	}
	mtd := monthly
	if lm.IsCurrent && source != finops.SourceCE {
		mtd = costest.MonthToDateCents(monthly, time.Now())
	}

	shownFindings := findings
	denied := hasFinding(findings, finops.FindingCEDenied) || ceDenied
	if source == finops.SourceCE {
		denied = false
		var keep []models.Finding
		for _, f := range shownFindings {
			if f.Kind != finops.FindingCEDenied {
				keep = append(keep, f)
			}
		}
		shownFindings = keep
	}
	if !lm.IsCurrent {
		shownFindings = nil
		if source != finops.SourceCE {
			denied = ceDenied
		}
	}

	return tenantView{
		Summary: map[string]any{
			"monthlyCents":  monthly,
			"monthlyUSD":    formatUSD(monthly),
			"mtdCents":      mtd,
			"mtdUSD":        formatUSD(mtd),
			"source":        source,
			"accountCount":  len(accounts),
			"resourceCount": len(resources),
			"ceDenied":      denied,
		},
		Services:  serviceProps(costLines),
		Resources: resourceProps(resources, cat),
		Findings:  findingProps(shownFindings, cat),
		Budgets:   budgetProps(budgets, monthly),
		Accounts:  accountProps(accounts),
		LastSync:  lastSyncProps(s, accounts),
	}, nil
}

func monthSpend(s store.Store, accounts []models.CloudAccount, resources []models.CloudResource, cat *i18n.Catalog, lm awsinv.LedgerMonth, overlay []models.CostLine) (int64, []models.CostLine, string, error) {
	if !lm.IsCurrent || len(overlay) > 0 {
		monthly, lines, source := sumCostLines(overlay)
		return monthly, lines, source, nil
	}
	var monthly int64
	var lines []models.CostLine
	source := finops.SourceEstimate
	for _, acc := range accounts {
		costLines, err := s.ListCostLines(acc.ID)
		if err != nil {
			return 0, nil, "", err
		}
		m, l, src := sumCostLines(costLines)
		monthly += m
		lines = append(lines, l...)
		if src == finops.SourceCE {
			source = finops.SourceCE
		}
	}
	if monthly == 0 {
		for _, r := range resources {
			monthly += r.MonthlyCents
			lines = append(lines, models.CostLine{
				Service: resourceKindLabel(r.Kind, cat), MonthlyCents: r.MonthlyCents, Source: r.Source,
			})
		}
	}
	return monthly, lines, source, nil
}

func sumCostLines(costLines []models.CostLine) (monthly int64, lines []models.CostLine, source string) {
	source = finops.SourceEstimate
	for _, line := range costLines {
		monthly += line.MonthlyCents
		lines = append(lines, line)
		if line.Source == finops.SourceCE {
			source = finops.SourceCE
		}
	}
	return monthly, lines, source
}

func lastSyncProps(s store.Store, accounts []models.CloudAccount) map[string]any {
	var lastSync map[string]any
	for _, acc := range accounts {
		if run, err := s.LastSyncRun(acc.ID); err == nil && run.ID != 0 {
			lastSync = map[string]any{
				"status":  run.Status,
				"source":  run.Source,
				"warning": run.Warning,
				"error":   run.Error,
				"at":      run.StartedAt.Format(time.RFC3339),
			}
		}
	}
	return lastSync
}

func attachForecast(ctx context.Context, syn *syncer.Syncer, tenantID int64, now time.Time, cat *i18n.Catalog, summary map[string]any) {
	period := awsinv.NextMonth(now)
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	cents, err := syn.ForecastForMonth(ctx, tenantID, period)
	if err != nil || cents <= 0 {
		return
	}
	summary["forecastUSD"] = formatUSD(cents)
	summary["forecastLabel"] = ledgerMonthLabel(cat, awsinv.LedgerMonth{Period: period})
}

func ledgerMonthLabel(cat *i18n.Catalog, lm awsinv.LedgerMonth) string {
	name := cat.T(awsinv.MonthLabelKey(lm.Period))
	label := cat.T("dash.month_fmt")
	label = strings.Replace(label, "%s", name, 1)
	label = strings.Replace(label, "%d", strconv.Itoa(lm.Period.Year()), 1)
	return label
}

func resourceProps(resources []models.CloudResource, cat *i18n.Catalog) []map[string]any {
	out := make([]map[string]any, 0, len(resources))
	for _, r := range resources {
		out = append(out, map[string]any{
			"id":     r.ID,
			"kind":   r.Kind,
			"label":  resourceKindLabel(r.Kind, cat),
			"name":   r.Name,
			"region": r.Region,
			"state":  r.State,
			"usd":    formatUSD(r.MonthlyCents),
			"cents":  r.MonthlyCents,
			"source": r.Source,
		})
	}
	return out
}

func findingProps(findings []models.Finding, cat *i18n.Catalog) []map[string]any {
	out := make([]map[string]any, 0, len(findings))
	for _, f := range findings {
		title, detail := translateFinding(cat, f)
		out = append(out, map[string]any{
			"kind": f.Kind, "severity": f.Severity, "title": title, "detail": detail,
		})
	}
	return out
}

func budgetProps(budgets []models.Budget, spent int64) []map[string]any {
	out := make([]map[string]any, 0, len(budgets))
	for _, b := range budgets {
		out = append(out, map[string]any{
			"id":      b.ID,
			"name":    b.Name,
			"amount":  formatUSD(b.AmountCents),
			"spent":   formatUSD(spent),
			"burnBps": costest.BudgetBurnBps(spent, b.AmountCents),
			"period":  b.Period,
			"over":    spent > b.AmountCents && b.AmountCents > 0,
		})
	}
	return out
}

func accountProps(accounts []models.CloudAccount) []map[string]any {
	out := make([]map[string]any, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, map[string]any{
			"id":           a.ID,
			"awsAccountId": a.AWSAccountID,
			"alias":        a.Alias,
			"region":       a.Region,
			"authMode":     a.AuthMode,
			"primary":      a.IsPrimary,
		})
	}
	return out
}

func withSpendPct(services []map[string]any) []map[string]any {
	var max int64
	out := make([]map[string]any, 0, len(services))
	for _, s := range services {
		cents, _ := s["cents"].(int64)
		if cents <= 0 {
			continue
		}
		if cents > max {
			max = cents
		}
		out = append(out, s)
	}
	for _, s := range out {
		cents, _ := s["cents"].(int64)
		if max == 0 {
			s["pct"] = 0
			continue
		}
		s["pct"] = int(cents * 100 / max)
	}
	return out
}

func hasFinding(findings []models.Finding, kind string) bool {
	for _, f := range findings {
		if f.Kind == kind {
			return true
		}
	}
	return false
}
