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

func TestAccountsHandler_Create_linksHetznerProject(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}

	key := crypto.DeriveKey("test-secret")
	h := NewAccountsHandler(s, testSite(), cais.Config{}, setupTestViews(t), key)
	body := `{"provider":"hetzner","alias":"prod","api_token":"tok-1234567890abcdef"}`
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
	found := false
	for _, acc := range accounts {
		if acc.Provider != finops.ProviderHetzner {
			continue
		}
		found = true
		if acc.AWSAccountID != "hz:prod" {
			t.Fatalf("external id = %q", acc.AWSAccountID)
		}
		if acc.AuthMode != finops.AuthModeAPIToken {
			t.Fatalf("auth = %q", acc.AuthMode)
		}
		if acc.SecretCipher == "" || acc.SecretCipher == "tok-1234567890abcdef" {
			t.Fatal("token should be encrypted")
		}
		plain, err := crypto.Decrypt(key, acc.SecretCipher)
		if err != nil || plain != "tok-1234567890abcdef" {
			t.Fatalf("decrypt = %q err=%v", plain, err)
		}
	}
	if !found {
		t.Fatalf("hetzner account missing: %#v", accounts)
	}
}

func TestAccountsHandler_List_includesHetznerTokenField(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err := seed.EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}
	h := NewAccountsHandler(s, testSite(), cais.Config{}, setupTestViews(t), nil)
	req := inertiaRequest(http.MethodGet, "/accounts", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.List(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `name="api_token"`) {
		t.Fatalf("missing hetzner token field: %s", body)
	}
	if !strings.Contains(body, `value="hetzner"`) {
		t.Fatalf("missing hetzner provider: %s", body)
	}
	assertPasswordEyeToggles(t, body, 2)
}
