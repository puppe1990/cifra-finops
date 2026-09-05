package handlers

import (
	"net/http"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	"github.com/puppe1990/aws-finops/internal/awsinv"
	"github.com/puppe1990/aws-finops/internal/finops"
	"github.com/puppe1990/aws-finops/internal/store"
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
	props := shellProps(h.site, r, h.store, ws)
	props["policy"] = awsinv.FinOpsIAMPolicy
	props["cloudShell"] = awsinv.CloudShellCommand()
	props["seededAccount"] = finops.SeedAWSAccountID()
	writePage(w, r, h.views, h.cfg, "app", "settings", props, 0)
}
