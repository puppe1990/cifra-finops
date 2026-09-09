package handlers

import (
	"net/http"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	"github.com/puppe1990/cifra-finops/internal/store"
)

type ResourcesHandler struct {
	store store.Store
	site  meta.Site
	cfg   cais.Config
	views *view.Renderer
}

func NewResourcesHandler(s store.Store, site meta.Site, cfg cais.Config, views *view.Renderer) *ResourcesHandler {
	return &ResourcesHandler{store: s, site: site, cfg: cfg, views: views}
}

func (h *ResourcesHandler) List(w http.ResponseWriter, r *http.Request) {
	ws, err := loadWorkspace(h.store, r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	resources, err := h.store.ListResourcesForTenant(ws.Tenant.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	props := shellProps(h.site, r, h.store, ws)
	props["resources"] = resourceProps(resources, requestCatalog(r, "en"))
	writePage(w, r, h.views, h.cfg, "app", "resources", props, 0)
}
