package handlers

import (
	"fmt"
	"net/http"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	"github.com/puppe1990/aws-finops/internal/finops"
)

type HomeHandler struct {
	site    meta.Site
	catalog *i18n.Catalog
	cfg     cais.Config
	views   *view.Renderer
}

func NewHomeHandler(_ *view.Renderer, site meta.Site, catalog *i18n.Catalog, cfg cais.Config, views *view.Renderer) *HomeHandler {
	return &HomeHandler{site: site, catalog: catalog, cfg: cfg, views: views}
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	site := meta.ForRequest(h.site, r)
	cat := requestCatalog(r, h.cfg.Locale)
	props := publicProps(h.site, r, h.cfg.Locale)
	props["title"] = cat.T("home.title")
	labels, _ := props["labels"].(map[string]string)
	if labels == nil {
		labels = map[string]string{}
	}
	labels["heading"] = cat.T("home.rails_heading")
	labels["subtitle"] = fmt.Sprintf(cat.T("home.rails_subtitle"), site.AppName)
	labels["stack"] = cat.T("home.stack")
	labels["contact"] = cat.T("home.contact_link")
	labels["login"] = cat.T("auth.login_submit")
	labels["dashboard"] = cat.T("dashboard.title")
	labels["account"] = finops.SeedAWSAccountID()
	if labels["account"] != "" {
		labels["eyebrow"] = fmt.Sprintf(cat.T("home.eyebrow_seeded"), labels["account"])
	} else {
		labels["eyebrow"] = cat.T("home.eyebrow_empty")
	}
	props["labels"] = labels
	writePage(w, r, h.views, h.cfg, "public", "home", props, 0)
}
