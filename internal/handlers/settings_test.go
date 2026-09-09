package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	"github.com/puppe1990/cifra-finops/internal/seed"
	"github.com/puppe1990/cifra-finops/internal/store"
)

func TestSettingsHandler_Get_includesPasswordForm(t *testing.T) {
	s := setupTestStore(t)
	hash, err := session.HashPassword("oldpass12")
	if err != nil {
		t.Fatal(err)
	}
	uid, err := s.CreateUser("ops@example.com", hash)
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}

	h := NewSettingsHandler(s, testSite(), cais.Config{}, setupTestViews(t))
	req := inertiaRequest(http.MethodGet, "/settings", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `action="/settings/password"`) {
		t.Fatal("missing password form")
	}
	assertPasswordEyeToggles(t, body, 3)
}

func TestSettingsHandler_Get_marksSettingsNavActive(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}
	h := NewSettingsHandler(s, testSite(), cais.Config{}, setupTestViews(t))
	req := inertiaRequest(http.MethodGet, "/settings", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	body := rr.Body.String()
	settings := navLinkClass(body, "/settings")
	ledger := navLinkClass(body, "/dashboard")
	if !strings.Contains(settings, "bg-copper-500") {
		t.Errorf("settings nav not copper: %s", settings)
	}
	if strings.Contains(ledger, "bg-copper-500") {
		t.Errorf("ledger nav still copper on settings: %s", ledger)
	}
}

func settingsChangePassword(t *testing.T, current, next, confirm string) (*httptest.ResponseRecorder, store.Store, int64) {
	t.Helper()
	s := setupTestStore(t)
	hash, err := session.HashPassword("oldpass12")
	if err != nil {
		t.Fatal(err)
	}
	uid, err := s.CreateUser("ops@example.com", hash)
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}
	h := NewSettingsHandler(s, testSite(), cais.Config{}, setupTestViews(t))
	form := url.Values{
		"current_password":      {current},
		"password":              {next},
		"password_confirmation": {confirm},
	}
	req := httptest.NewRequest(http.MethodPost, "/settings/password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ChangePassword(rr, req)
	return rr, s, uid
}

func TestSettingsHandler_ChangePassword_success(t *testing.T) {
	rr, s, uid := settingsChangePassword(t, "oldpass12", "newpass12", "newpass12")
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Location") != "/settings?tab=password" {
		t.Errorf("Location = %q", rr.Header().Get("Location"))
	}
	u, err := s.FindUserByID(uid)
	if err != nil {
		t.Fatal(err)
	}
	if !session.VerifyPassword(u.PasswordHash, "newpass12") {
		t.Fatal("password was not updated")
	}
}

func TestSettingsHandler_ChangePassword_wrongCurrent(t *testing.T) {
	rr, _, _ := settingsChangePassword(t, "nope!!!!", "newpass12", "newpass12")
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="settings-tab-password"`) {
		t.Error("password panel missing after 422")
	}
	if strings.Contains(body, `id="settings-tab-password" hidden`) {
		t.Error("password panel hidden after 422")
	}
}

func TestSettingsHandler_includesPolicyAndCloudShell(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}

	h := NewSettingsHandler(s, testSite(), cais.Config{}, setupTestViews(t))
	req := inertiaRequest(http.MethodGet, "/settings", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	assertInertiaComponent(t, rr, "Settings")
	if !strings.Contains(rr.Body.String(), "CifraFinOpsRead") && !strings.Contains(rr.Body.String(), "2012-10-17") {
		t.Errorf("policy missing from HTML")
	}
	if !strings.Contains(rr.Body.String(), "aws") {
		t.Errorf("cloudShell missing from HTML: %s", rr.Body.String())
	}
}

func settingsGET(t *testing.T, path string) string {
	t.Helper()
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}
	h := NewSettingsHandler(s, testSite(), cais.Config{}, setupTestViews(t))
	req := inertiaRequest(http.MethodGet, path, nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.Get(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	return rr.Body.String()
}

func TestSettingsHandler_Get_tabsDefaultPassword(t *testing.T) {
	body := settingsGET(t, "/settings")
	for _, tab := range []string{"password", "sync", "cloudshell", "policy"} {
		if !strings.Contains(body, `href="/settings?tab=`+tab+`"`) {
			t.Errorf("missing tab link %s", tab)
		}
	}
	if !strings.Contains(body, `id="settings-tab-password"`) {
		t.Fatal("missing password panel")
	}
	if strings.Contains(body, `id="settings-tab-password" hidden`) {
		t.Error("password panel hidden on default")
	}
	for _, tab := range []string{"sync", "cloudshell", "policy"} {
		if !strings.Contains(body, `id="settings-tab-`+tab+`" hidden`) {
			t.Errorf("%s panel should be hidden", tab)
		}
	}
}

func TestSettingsHandler_Get_tabPolicyHidesPassword(t *testing.T) {
	body := settingsGET(t, "/settings?tab=policy")
	if !strings.Contains(body, `id="settings-tab-password" hidden`) {
		t.Error("password should be hidden")
	}
	if !strings.Contains(body, `id="settings-tab-policy"`) {
		t.Fatal("missing policy panel")
	}
	if strings.Contains(body, `id="settings-tab-policy" hidden`) {
		t.Error("policy panel hidden")
	}
}

func TestSettingsHandler_Get_cloudshellHasCopyButton(t *testing.T) {
	body := settingsGET(t, "/settings?tab=cloudshell")
	if !strings.Contains(body, `id="settings-tab-cloudshell"`) {
		t.Fatal("missing cloudshell panel")
	}
	if !strings.Contains(body, ">Copy</button>") {
		t.Fatal("cloudshell tab missing Copy button")
	}
	if !strings.Contains(body, `amarra-hook="clipboard"`) {
		t.Fatal("copy button missing clipboard hook")
	}
}

func TestSettingsHandler_Get_unknownTabFallsBackToPassword(t *testing.T) {
	body := settingsGET(t, "/settings?tab=nope")
	if !strings.Contains(body, `id="settings-tab-password"`) {
		t.Fatal("missing password panel")
	}
	if strings.Contains(body, `id="settings-tab-password" hidden`) {
		t.Error("password panel hidden on unknown tab")
	}
}
