package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/store"
)

type BudgetsHandler struct {
	store store.Store
	site  meta.Site
	cfg   cais.Config
	views *view.Renderer
}

func NewBudgetsHandler(s store.Store, site meta.Site, cfg cais.Config, views *view.Renderer) *BudgetsHandler {
	return &BudgetsHandler{store: s, site: site, cfg: cfg, views: views}
}

func (h *BudgetsHandler) List(w http.ResponseWriter, r *http.Request) {
	ws, err := loadWorkspace(h.store, r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	view, err := buildTenantView(h.store, ws.Tenant.ID, requestCatalog(r, h.cfg.Locale), awsinv.LedgerMonth{IsCurrent: true}, nil, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	props := shellProps(h.site, r, h.store, ws)
	props["budgets"] = view.Budgets
	props["spentUSD"] = view.Summary["monthlyUSD"]
	writePage(w, r, h.views, h.cfg, "app", "budgets", props, 0)
}

func (h *BudgetsHandler) Create(w http.ResponseWriter, r *http.Request) {
	ws, err := loadWorkspace(h.store, r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := parseFormBody(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	amount, _ := strconv.ParseFloat(strings.ReplaceAll(r.FormValue("amount_usd"), ",", "."), 64)
	if name == "" || amount <= 0 {
		flash.Set(w, "alert", requestCatalog(r, h.cfg.Locale).T("bud.need_fields"), h.cfg.CookieSecure())
		http.Redirect(w, r, "/budgets", http.StatusSeeOther)
		return
	}
	if _, err := h.store.CreateBudget(models.Budget{
		TenantID:    ws.Tenant.ID,
		Name:        name,
		AmountCents: int64(amount * 100),
		Period:      "monthly",
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	flash.Set(w, "notice", requestCatalog(r, h.cfg.Locale).T("bud.created"), h.cfg.CookieSecure())
	http.Redirect(w, r, "/budgets", http.StatusSeeOther)
}
