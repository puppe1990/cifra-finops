package handlers

import (
	"net/http"
	"strings"

	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/validate"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/crypto"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func (h *AccountsHandler) createHetzner(w http.ResponseWriter, r *http.Request, ws workspace) {
	alias := strings.TrimSpace(r.FormValue("alias"))
	token := strings.TrimSpace(r.FormValue("api_token"))
	cat := requestCatalog(r, h.cfg.Locale)
	var errs validate.FieldErrors
	if alias == "" {
		errs.Add("alias", cat.T("acc.err_alias"))
	}
	if !finops.ValidHetznerToken(token) {
		errs.Add("api_token", cat.T("acc.err_token"))
	}
	if errs.Any() {
		accounts, _ := h.store.ListCloudAccounts(ws.Tenant.ID)
		props := shellProps(h.site, r, h.store, ws)
		props["accounts"] = accountProps(accounts)
		props["policy"] = awsinv.FinOpsIAMPolicy
		writePage(w, r, h.views, h.cfg, "app", "accounts", props, 422)
		return
	}

	acc := models.CloudAccount{
		TenantID:     ws.Tenant.ID,
		Provider:     finops.ProviderHetzner,
		AWSAccountID: finops.HetznerProjectID(alias),
		Alias:        alias,
		Region:       finops.DefaultHetznerRegion,
		AuthMode:     finops.AuthModeAPIToken,
	}
	if len(h.appSecret) > 0 {
		ct, encErr := crypto.Encrypt(h.appSecret, token)
		if encErr != nil {
			http.Error(w, encErr.Error(), http.StatusInternalServerError)
			return
		}
		acc.SecretCipher = ct
	}
	if _, err := h.store.EnsureCloudAccount(acc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.store.UpdateCloudAccountAuth(acc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	flash.Set(w, "notice", cat.T("acc.linked_hetzner"), h.cfg.CookieSecure())
	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
}
