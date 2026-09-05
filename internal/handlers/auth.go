package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/httpx"
	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"
	"github.com/puppe1990/amarra-cais/pkg/cais/passwordreset"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"
	"github.com/puppe1990/amarra-cais/pkg/cais/validate"

	"github.com/puppe1990/aws-finops/internal/store"
)

type AuthHandler struct {
	renderer    *cais.Renderer
	store       store.Store
	site        meta.Site
	sessions    session.Store
	cfg         cais.Config
	catalog     *i18n.Catalog
	resetNotify passwordreset.Notifier
	views       *view.Renderer
}

func NewAuthHandler(_ *view.Renderer, s store.Store, site meta.Site, sessions session.Store, cfg cais.Config, catalog *i18n.Catalog, views *view.Renderer) *AuthHandler {
	return &AuthHandler{store: s, site: site, sessions: sessions, cfg: cfg, catalog: catalog, views: views}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if _, ok := session.UserID(r); ok {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	writePage(w, r, h.views, h.cfg, "auth", "login", publicProps(h.site, r, h.cfg.Locale))
}

func (h *AuthHandler) LoginPost(w http.ResponseWriter, r *http.Request) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	cat := requestCatalog(r, h.cfg.Locale)
	user, err := h.store.FindUserByEmail(email)
	if err != nil || !session.VerifyPassword(user.PasswordHash, password) {
		props := publicProps(h.site, r, h.cfg.Locale)
		props["Errors"] = map[string]string{"email": cat.T("auth.invalid_credentials")}
		writePage(w, r, h.views, h.cfg, "auth", "login", props, 422)
		return
	}

	if err := session.SignIn(w, h.sessions, r, user.ID, session.CookieOptionsFromConfig(h.cfg)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// cais flash cookie (gonertia SetFlash needs FlashDataProvider; scaffold uses cookies — #140).
	flash.Set(w, "notice", requestCatalog(r, h.cfg.Locale).T("auth.welcome"), h.cfg.CookieSecure())
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *AuthHandler) LogoutPost(w http.ResponseWriter, r *http.Request) {
	session.SignOut(w, h.sessions, r, session.CookieOptionsFromConfig(h.cfg))
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	if _, ok := session.UserID(r); ok {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	writePage(w, r, h.views, h.cfg, "auth", "signup", publicProps(h.site, r, h.cfg.Locale))
}

func (h *AuthHandler) SignUpPost(w http.ResponseWriter, r *http.Request) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirmation")

	var errs validate.FieldErrors
	if err := validate.Email(email); err != nil {
		errs.Add("email", requestCatalog(r, h.cfg.Locale).T("contact.email_invalid"))
	}
	if err := validate.MinLength(password, 8); err != nil {
		errs.Add("password", requestCatalog(r, h.cfg.Locale).T("auth.password_too_short"))
	}
	if password != confirm {
		errs.Add("password_confirmation", requestCatalog(r, h.cfg.Locale).T("auth.password_mismatch"))
	}
	if errs.Any() {
		ve := map[string]string{}
		for k, v := range errs {
			ve[k] = v
		}
		props := publicProps(h.site, r, h.cfg.Locale)
		props["Errors"] = ve
		writePage(w, r, h.views, h.cfg, "auth", "signup", props, 422)
		return
	}

	hash, err := session.HashPassword(password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userID, err := h.store.CreateUser(email, hash)
	if err != nil {
		if errors.Is(err, store.ErrEmailTaken) {
			props := publicProps(h.site, r, h.cfg.Locale)
			props["Errors"] = map[string]string{"email": requestCatalog(r, h.cfg.Locale).T("auth.email_taken")}
			writePage(w, r, h.views, h.cfg, "auth", "signup", props, 422)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := createPersonalTenant(h.store, userID, email); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := session.SignIn(w, h.sessions, r, userID, session.CookieOptionsFromConfig(h.cfg)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	flash.Set(w, "notice", requestCatalog(r, h.cfg.Locale).T("auth.welcome"), h.cfg.CookieSecure())
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if _, ok := session.UserID(r); ok {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	writePage(w, r, h.views, h.cfg, "auth", "forgot_password", publicProps(h.site, r, h.cfg.Locale))
}

func (h *AuthHandler) ForgotPasswordPost(w http.ResponseWriter, r *http.Request) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	var errs validate.FieldErrors
	if err := validate.Email(email); err != nil {
		errs.Add("email", requestCatalog(r, h.cfg.Locale).T("contact.email_invalid"))
	}
	if errs.Any() {
		ve := map[string]string{}
		for k, v := range errs {
			ve[k] = v
		}
		props := publicProps(h.site, r, h.cfg.Locale)
		props["Errors"] = ve
		writePage(w, r, h.views, h.cfg, "auth", "forgot_password", props, 422)
		return
	}

	if user, err := h.store.FindUserByEmail(email); err == nil {
		token, err := h.store.CreatePasswordResetToken(user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = h.resetNotifier().NotifyReset(user.Email, token)
	}

	flash.Set(w, "notice", requestCatalog(r, h.cfg.Locale).T("auth.reset_email_sent"), h.cfg.CookieSecure())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if _, ok := session.UserID(r); ok {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get("token"))
	props := publicProps(h.site, r, h.cfg.Locale)
	props["token"] = token
	if token == "" {
		writePage(w, r, h.views, h.cfg, "auth", "reset_password", props, 422)
		return
	}
	if _, ok := h.store.FindPasswordResetUserID(token); !ok {
		writePage(w, r, h.views, h.cfg, "auth", "reset_password", props, 422)
		return
	}
	writePage(w, r, h.views, h.cfg, "auth", "reset_password", props, 0)
}

func (h *AuthHandler) ResetPasswordPost(w http.ResponseWriter, r *http.Request) {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token := strings.TrimSpace(r.FormValue("token"))
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirmation")

	var errs validate.FieldErrors
	if token == "" {
		errs.Add("token", requestCatalog(r, h.cfg.Locale).T("auth.reset_invalid_token"))
	} else if _, ok := h.store.FindPasswordResetUserID(token); !ok {
		props := publicProps(h.site, r, h.cfg.Locale)
		props["token"] = token
		props["Errors"] = map[string]string{"token": requestCatalog(r, h.cfg.Locale).T("auth.reset_invalid_token")}
		writePage(w, r, h.views, h.cfg, "auth", "reset_password", props, 422)
		return
	}
	if err := validate.MinLength(password, 8); err != nil {
		errs.Add("password", requestCatalog(r, h.cfg.Locale).T("auth.password_too_short"))
	}
	if password != confirm {
		errs.Add("password_confirmation", requestCatalog(r, h.cfg.Locale).T("auth.password_mismatch"))
	}
	if errs.Any() {
		ve := map[string]string{}
		for k, v := range errs {
			ve[k] = v
		}
		writePage(w, r, h.views, h.cfg, "auth", "reset_password", map[string]any{"token": token}, 422)
		return
	}

	hash, err := session.HashPassword(password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.store.ResetPasswordWithToken(token, hash); err != nil {
		writePage(w, r, h.views, h.cfg, "auth", "reset_password", map[string]any{
			"site":  meta.ForRequest(h.site, r),
			"token": token,
		})
		return
	}

	flash.Set(w, "notice", requestCatalog(r, h.cfg.Locale).T("auth.reset_success"), h.cfg.CookieSecure())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) resetNotifier() passwordreset.Notifier {
	if h.resetNotify != nil {
		return h.resetNotify
	}
	return passwordreset.NotifierFromConfig(h.cfg, h.site.AppName)
}
