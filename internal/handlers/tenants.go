package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	"github.com/puppe1990/aws-finops/internal/finops"
	"github.com/puppe1990/aws-finops/internal/store"
)

type TenantsHandler struct {
	store store.Store
	site  meta.Site
	cfg   cais.Config
	views *view.Renderer
}

func NewTenantsHandler(s store.Store, site meta.Site, cfg cais.Config, views *view.Renderer) *TenantsHandler {
	return &TenantsHandler{store: s, site: site, cfg: cfg, views: views}
}

func (h *TenantsHandler) List(w http.ResponseWriter, r *http.Request) {
	ws, err := loadWorkspace(h.store, r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	props := shellProps(h.site, r, h.store, ws)
	writePage(w, r, h.views, h.cfg, "app", "tenants", props, 0)
}

func (h *TenantsHandler) Create(w http.ResponseWriter, r *http.Request) {
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
	slug := slugify(r.FormValue("slug"), name)
	if name == "" || slug == "" {
		flash.Set(w, "alert", requestCatalog(r, h.cfg.Locale).T("ten.need_name"), h.cfg.CookieSecure())
		http.Redirect(w, r, "/tenants", http.StatusSeeOther)
		return
	}
	id, err := h.store.CreateTenant(name, slug)
	if err != nil {
		flash.Set(w, "alert", requestCatalog(r, h.cfg.Locale).T("ten.slug_taken"), h.cfg.CookieSecure())
		http.Redirect(w, r, "/tenants", http.StatusSeeOther)
		return
	}
	if err := h.store.AddMember(id, ws.User.ID, finops.RoleOwner); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = h.store.SetActiveTenant(ws.User.ID, id)
	flash.Set(w, "notice", requestCatalog(r, h.cfg.Locale).T("ten.created"), h.cfg.CookieSecure())
	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
}

func (h *TenantsHandler) Switch(w http.ResponseWriter, r *http.Request) {
	ws, err := loadWorkspace(h.store, r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := parseFormBody(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, _ := strconv.ParseInt(r.FormValue("tenant_id"), 10, 64)
	if _, ok, err := h.store.MembershipRole(id, ws.User.ID); err != nil || !ok {
		flash.Set(w, "alert", requestCatalog(r, h.cfg.Locale).T("ten.not_member"), h.cfg.CookieSecure())
		http.Redirect(w, r, "/tenants", http.StatusSeeOther)
		return
	}
	if err := h.store.SetActiveTenant(ws.User.ID, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	flash.Set(w, "notice", requestCatalog(r, h.cfg.Locale).T("ten.switched"), h.cfg.CookieSecure())
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func slugify(raw, fallback string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		s = strings.ToLower(fallback)
	}
	re := regexp.MustCompile(`[^a-z0-9]+`)
	return strings.Trim(re.ReplaceAllString(s, "-"), "-")
}
