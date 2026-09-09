package handlers

import (
	"net/http"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"
	"github.com/puppe1990/amarra-cais/pkg/cais/validate"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/store"
)

type SettingsHandler struct {
	store store.Store
	site  meta.Site
	cfg   cais.Config
	views *view.Renderer
}

func NewSettingsHandler(s store.Store, site meta.Site, cfg cais.Config, views *view.Renderer) *SettingsHandler {
	return &SettingsHandler{store: s, site: site, cfg: cfg, views: views}
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	ws, err := loadWorkspace(h.store, r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	h.render(w, r, ws, nil, 0)
}

func (h *SettingsHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ws, err := loadWorkspace(h.store, r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := parseFormBody(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.store.FindUserByID(ws.User.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	current := r.FormValue("current_password")
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirmation")
	cat := requestCatalog(r, h.cfg.Locale)
	errs := map[string]string{}
	if !session.VerifyPassword(user.PasswordHash, current) {
		errs["current_password"] = cat.T("set.current_password_wrong")
	}
	if err := validate.MinLength(password, 8); err != nil {
		errs["password"] = cat.T("auth.password_too_short")
	}
	if password != confirm {
		errs["password_confirmation"] = cat.T("auth.password_mismatch")
	}
	if len(errs) > 0 {
		h.render(w, r, ws, map[string]any{"Errors": errs}, http.StatusUnprocessableEntity)
		return
	}

	hash, err := session.HashPassword(password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.store.UpdateUserPassword(ws.User.ID, hash); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	flash.Set(w, "notice", cat.T("set.password_updated"), h.cfg.CookieSecure())
	http.Redirect(w, r, "/settings?tab=password", http.StatusSeeOther)
}

func settingsTab(r *http.Request) string {
	tab := r.URL.Query().Get("tab")
	switch tab {
	case "sync", "cloudshell", "policy":
		return tab
	default:
		return "password"
	}
}

func (h *SettingsHandler) render(w http.ResponseWriter, r *http.Request, ws workspace, extra map[string]any, status int) {
	props := shellProps(h.site, r, h.store, ws)
	props["policy"] = awsinv.FinOpsIAMPolicy
	props["cloudShell"] = awsinv.CloudShellCommand()
	props["seededAccount"] = finops.SeedAWSAccountID()
	props["settingsTab"] = settingsTab(r)
	for k, v := range extra {
		props[k] = v
	}
	writePage(w, r, h.views, h.cfg, "app", "settings", props, status)
}
