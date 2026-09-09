package handlers

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/store"
	"github.com/puppe1990/cifra-finops/internal/syncer"
)

type AnomaliesHandler struct {
	store  store.Store
	site   meta.Site
	cfg    cais.Config
	views  *view.Renderer
	syncer *syncer.Syncer
	now    func() time.Time
}

func NewAnomaliesHandler(s store.Store, site meta.Site, cfg cais.Config, views *view.Renderer) *AnomaliesHandler {
	return &AnomaliesHandler{store: s, site: site, cfg: cfg, views: views}
}

func (h *AnomaliesHandler) WithSyncer(s *syncer.Syncer) *AnomaliesHandler {
	h.syncer = s
	return h
}

func (h *AnomaliesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := loadWorkspace(h.store, r)
	props := map[string]any{
		"site":      meta.ForRequest(h.site, r),
		"anomalies": []any{},
		"ceDenied":  false,
		"flash":     map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = map[string]string{msg.Kind: msg.Message}
	}
	if err != nil {
		writePage(w, r, h.views, h.cfg, "app", "anomalies", props, 0)
		return
	}

	now := time.Now().UTC()
	if h.now != nil {
		now = h.now()
	}

	items := collectWorkspaceAnomalies(r.Context(), h.syncer, ws.Tenant.ID, now)
	for k, v := range shellProps(h.site, r, h.store, ws) {
		props[k] = v
	}
	props["anomalies"] = anomalyProps(items)
	props["ceDenied"] = items.denied
	writePage(w, r, h.views, h.cfg, "app", "anomalies", props, 0)
}

type anomalyBag struct {
	items  []awsinv.CostAnomaly
	denied bool
}

func collectWorkspaceAnomalies(ctx context.Context, syn *syncer.Syncer, tenantID int64, now time.Time) anomalyBag {
	if syn == nil {
		return anomalyBag{}
	}
	var bag anomalyBag
	from, to := awsinv.AnomalyWindow(now)
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	ceItems, err := syn.CostAnomalies(ctx, tenantID, from, to)
	if err != nil {
		if awsinv.IsAccessDenied(err) {
			bag.denied = true
			return bag
		}
	} else {
		bag.items = append(bag.items, ceItems...)
	}

	window := awsinv.LookbackMonths(now)
	lines, err := syn.CostByMonth(ctx, tenantID, window[0], window[len(window)-1])
	if err != nil {
		if awsinv.IsAccessDenied(err) {
			bag.denied = true
		}
		return bag
	}
	buckets := awsinv.FoldMonthlyLines(window[0], window[len(window)-1], lines)
	bag.items = append(bag.items, awsinv.SpikeMonths(buckets)...)
	if n := len(buckets); n >= 2 {
		bag.items = append(bag.items, awsinv.SpikeServices(buckets[n-1].Query, buckets[n-1].Lines, buckets[n-2].Lines)...)
	}
	return bag
}

func anomalyProps(bag anomalyBag) []map[string]any {
	items := append([]awsinv.CostAnomaly(nil), bag.items...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].ImpactCents == items[j].ImpactCents {
			return items[i].Query > items[j].Query
		}
		return items[i].ImpactCents > items[j].ImpactCents
	})
	out := make([]map[string]any, 0, len(items))
	for _, a := range items {
		out = append(out, map[string]any{
			"id":      a.ID,
			"kind":    a.Kind,
			"service": a.Service,
			"query":   a.Query,
			"start":   a.Start,
			"end":     a.End,
			"usd":     formatUSD(a.ImpactCents),
			"score":   a.Score,
		})
	}
	return out
}
