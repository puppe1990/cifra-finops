package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/store"
	"github.com/puppe1990/cifra-finops/internal/syncer"
)

type CompareHandler struct {
	store  store.Store
	site   meta.Site
	cfg    cais.Config
	views  *view.Renderer
	syncer *syncer.Syncer
	now    func() time.Time
}

func NewCompareHandler(s store.Store, site meta.Site, cfg cais.Config, views *view.Renderer) *CompareHandler {
	return &CompareHandler{store: s, site: site, cfg: cfg, views: views}
}

func (h *CompareHandler) WithSyncer(s *syncer.Syncer) *CompareHandler {
	h.syncer = s
	return h
}

func (h *CompareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := loadWorkspace(h.store, r)
	props := map[string]any{
		"site":     meta.ForRequest(h.site, r),
		"months":   []any{},
		"services": []any{},
		"ceDenied": false,
		"flash":    map[string]string{},
	}
	if msg, ok := flash.MessageFromRequest(r); ok {
		props["flash"] = map[string]string{msg.Kind: msg.Message}
	}
	if err != nil {
		writePage(w, r, h.views, h.cfg, "app", "compare", props, 0)
		return
	}

	now := time.Now().UTC()
	if h.now != nil {
		now = h.now()
	}
	window := awsinv.LookbackMonths(now)
	from, to := window[0], window[len(window)-1]
	cat := requestCatalog(r, h.cfg.Locale)

	var lines []models.CostLine
	ceDenied := false
	if h.syncer != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		var overlayErr error
		lines, overlayErr = h.syncer.CostByMonth(ctx, ws.Tenant.ID, from, to)
		cancel()
		if overlayErr != nil {
			if awsinv.IsAccessDenied(overlayErr) {
				ceDenied = true
			}
			lines = nil
		}
	}

	buckets := awsinv.FoldMonthlyLines(from, to, lines)

	for k, v := range shellProps(h.site, r, h.store, ws) {
		props[k] = v
	}
	months := compareMonthRows(buckets, cat, now)
	props["months"] = months
	if len(months) > 0 {
		props["current"] = months[0]
	}
	props["services"] = compareServiceHistory(buckets)
	props["ceDenied"] = ceDenied
	writePage(w, r, h.views, h.cfg, "app", "compare", props, 0)
}
