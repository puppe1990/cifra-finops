package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	"github.com/puppe1990/cifra-finops/internal/crypto"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/seed"
)

func TestAccountsHandler_List_secretKeyHasEyeToggle(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(finops.SeedAccountEnv, "111111111111")
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}
	h := NewAccountsHandler(s, testSite(), cais.Config{}, setupTestViews(t), nil)
	req := inertiaRequest(http.MethodGet, "/accounts", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if rr.Code == http.StatusSeeOther {
		t.Fatal("redirected; workspace missing for list")
	}
	assertPasswordEyeToggles(t, rr.Body.String(), 2)
	body := rr.Body.String()
	if !strings.Contains(body, `value="access_keys" selected`) {
		t.Fatal("access keys should be the default auth mode on a VPS")
	}
	if strings.Contains(body, `id="aws-keys" hidden`) {
		t.Fatal("access key fields must be visible when access_keys is the default")
	}
}

func TestAccountsHandler_Create_updatesExistingToAccessKeys(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(finops.SeedAccountEnv, "111111111111")
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}

	key := crypto.DeriveKey("test-secret")
	h := NewAccountsHandler(s, testSite(), cais.Config{}, setupTestViews(t), key)
	body := `{"aws_account_id":"111111111111","alias":"principal","region":"us-east-1","auth_mode":"access_keys","access_key_id":"AKIATEST","secret_access_key":"secret"}`
	req := inertiaRequest(http.MethodPost, "/accounts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	tenant, err := s.FindTenantBySlug(finops.PrimaryTenantSlug)
	if err != nil {
		t.Fatal(err)
	}
	accounts, err := s.ListCloudAccounts(tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 {
		t.Fatalf("accounts = %#v", accounts)
	}
	acc := accounts[0]
	if acc.AuthMode != finops.AuthModeAccessKeys {
		t.Fatalf("auth_mode = %q, want access_keys", acc.AuthMode)
	}
	if acc.AccessKeyID != "AKIATEST" {
		t.Fatalf("access_key_id = %q", acc.AccessKeyID)
	}
	if acc.SecretCipher == "" || acc.SecretCipher == "secret" {
		t.Fatal("secret should be encrypted")
	}
	got, err := crypto.Decrypt(key, acc.SecretCipher)
	if err != nil || got != "secret" {
		t.Fatalf("decrypt = %q err=%v", got, err)
	}
}
