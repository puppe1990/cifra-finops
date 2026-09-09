package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"
	"github.com/puppe1990/amarra-cais/pkg/cais/validate"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/crypto"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/store"
	"github.com/puppe1990/cifra-finops/internal/syncer"
)

type AccountsHandler struct {
	store     store.Store
	site      meta.Site
	cfg       cais.Config
	views     *view.Renderer
	syncer    *syncer.Syncer
	appSecret []byte
}

func NewAccountsHandler(s store.Store, site meta.Site, cfg cais.Config, views *view.Renderer, appSecret []byte) *AccountsHandler {
	return &AccountsHandler{store: s, site: site, cfg: cfg, views: views, appSecret: appSecret}
}

func (h *AccountsHandler) WithSyncer(s *syncer.Syncer) *AccountsHandler {
	h.syncer = s
	return h
}

func (h *AccountsHandler) List(w http.ResponseWriter, r *http.Request) {
	ws, err := loadWorkspace(h.store, r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	accounts, err := h.store.ListCloudAccounts(ws.Tenant.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	props := shellProps(h.site, r, h.store, ws)
	props["accounts"] = accountProps(accounts)
	props["policy"] = awsinv.FinOpsIAMPolicy
	writePage(w, r, h.views, h.cfg, "app", "accounts", props, 0)
}

func (h *AccountsHandler) Create(w http.ResponseWriter, r *http.Request) {
	ws, err := loadWorkspace(h.store, r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := parseFormBody(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	awsID := strings.TrimSpace(r.FormValue("aws_account_id"))
	alias := strings.TrimSpace(r.FormValue("alias"))
	region := strings.TrimSpace(r.FormValue("region"))
	mode := strings.TrimSpace(r.FormValue("auth_mode"))
	if region == "" {
		region = finops.DefaultRegion
	}
	if mode == "" {
		mode = finops.AuthModeDefaultChain
	}
	var errs validate.FieldErrors
	if len(awsID) != 12 {
		errs.Add("aws_account_id", requestCatalog(r, h.cfg.Locale).T("acc.err_id"))
	}
	if alias == "" {
		errs.Add("alias", requestCatalog(r, h.cfg.Locale).T("acc.err_alias"))
	}
	if mode == finops.AuthModeAccessKeys && r.FormValue("access_key_id") == "" {
		errs.Add("access_key_id", requestCatalog(r, h.cfg.Locale).T("acc.err_key"))
	}
	if errs.Any() {
		ve := map[string]string{}
		for k, v := range errs {
			ve[k] = v
		}
		accounts, _ := h.store.ListCloudAccounts(ws.Tenant.ID)
		props := shellProps(h.site, r, h.store, ws)
		props["accounts"] = accountProps(accounts)
		props["policy"] = awsinv.FinOpsIAMPolicy
		writePage(w, r, h.views, h.cfg, "app", "accounts", props, 422)
		return
	}
	acc := models.CloudAccount{
		TenantID:     ws.Tenant.ID,
		AWSAccountID: awsID,
		Alias:        alias,
		Region:       region,
		AuthMode:     mode,
	}
	if mode == finops.AuthModeAccessKeys {
		acc.AccessKeyID = strings.TrimSpace(r.FormValue("access_key_id"))
		secret := r.FormValue("secret_access_key")
		if secret != "" && len(h.appSecret) > 0 {
			ct, encErr := crypto.Encrypt(h.appSecret, secret)
			if encErr != nil {
				http.Error(w, encErr.Error(), http.StatusInternalServerError)
				return
			}
			acc.SecretCipher = ct
		}
	}
	if _, err := h.store.EnsureCloudAccount(acc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if acc.AuthMode == finops.AuthModeAccessKeys {
		if err := h.store.UpdateCloudAccountAuth(acc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	flash.Set(w, "notice", requestCatalog(r, h.cfg.Locale).T("acc.linked"), h.cfg.CookieSecure())
	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
}

func (h *AccountsHandler) Sync(w http.ResponseWriter, r *http.Request) {
	ws, err := loadWorkspace(h.store, r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if h.syncer == nil {
		flash.Set(w, "alert", requestCatalog(r, h.cfg.Locale).T("acc.sync_none"), h.cfg.CookieSecure())
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
	defer cancel()
	if err := h.syncer.SyncTenant(ctx, ws.Tenant.ID); err != nil {
		flash.Set(w, "alert", requestCatalog(r, h.cfg.Locale).T("acc.sync_fail", err.Error()), h.cfg.CookieSecure())
	} else {
		flash.Set(w, "notice", requestCatalog(r, h.cfg.Locale).T("acc.sync_ok"), h.cfg.CookieSecure())
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
