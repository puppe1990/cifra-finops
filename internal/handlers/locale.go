package handlers

import (
	"net/http"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"

	"github.com/puppe1990/cifra-finops/internal/locale"
)

type LocaleHandler struct {
	cfg   cais.Config
	views *view.Renderer
}

func NewLocaleHandler(cfg cais.Config, views *view.Renderer) *LocaleHandler {
	return &LocaleHandler{cfg: cfg, views: views}
}

func (h *LocaleHandler) Post(w http.ResponseWriter, r *http.Request) {
	if err := parseFormBody(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	locale.SetCookie(w, r.FormValue("locale"), h.cfg.CookieSecure())
	http.Redirect(w, r, locale.SafeBack(r), http.StatusSeeOther)
}
