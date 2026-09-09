package seed

import (
	"testing"

	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/store"
)

func TestEnsurePrimaryWorkspace_attachesSeededAccount(t *testing.T) {
	t.Setenv(finops.SeedAccountEnv, "111111111111")
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	uid, err := s.CreateUser("demo@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}

	tenant, err := s.FindTenantBySlug(finops.PrimaryTenantSlug)
	if err != nil {
		t.Fatal(err)
	}
	if tenant.Name != finops.PrimaryTenantName {
		t.Fatalf("tenant name = %q, want %q", tenant.Name, finops.PrimaryTenantName)
	}

	accounts, err := s.ListCloudAccounts(tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 {
		t.Fatalf("accounts = %d, want 1", len(accounts))
	}
	if accounts[0].AWSAccountID != "111111111111" {
		t.Fatalf("aws account = %q, want 111111111111", accounts[0].AWSAccountID)
	}
	if accounts[0].AuthMode != finops.AuthModeDefaultChain {
		t.Fatalf("auth mode = %q, want default_chain", accounts[0].AuthMode)
	}

	role, ok, err := s.MembershipRole(tenant.ID, uid)
	if err != nil || !ok || role != finops.RoleOwner {
		t.Fatalf("membership role = %q ok=%v err=%v, want owner", role, ok, err)
	}
}

func TestEnsurePrimaryWorkspace_isIdempotent(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	t.Setenv(finops.SeedAccountEnv, "111111111111")

	uid, err := s.CreateUser("demo@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}
	if err := EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
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
		t.Fatalf("accounts after reseed = %d, want 1", len(accounts))
	}
}

func TestEnsurePrimaryWorkspace_skipsAccountWhenEnvUnset(t *testing.T) {
	t.Setenv(finops.SeedAccountEnv, "")
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	uid, err := s.CreateUser("demo@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsurePrimaryWorkspace(s, uid); err != nil {
		t.Fatal(err)
	}
	tenant, err := s.FindTenantBySlug(finops.PrimaryTenantSlug)
	if err != nil {
		t.Fatal(err)
	}
	accounts, err := s.ListCloudAccounts(tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 0 {
		t.Fatalf("accounts without seed env = %d, want 0", len(accounts))
	}
}
