package store

import (
	"testing"

	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func TestStore_ListResources_isolatedByTenant(t *testing.T) {
	s := newTestStore(t)

	alpha, err := s.CreateTenant("Alpha", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := s.CreateTenant("Beta", "beta")
	if err != nil {
		t.Fatal(err)
	}

	accA, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID:     alpha,
		AWSAccountID: "111111111111",
		Alias:        "alpha-prod",
		Region:       "us-east-1",
		AuthMode:     finops.AuthModeDefaultChain,
		IsPrimary:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	accB, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID:     beta,
		AWSAccountID: "222222222222",
		Alias:        "beta-prod",
		Region:       "us-east-1",
		AuthMode:     finops.AuthModeDefaultChain,
		IsPrimary:    true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.ReplaceResources(accA, []models.CloudResource{{
		Kind: "lightsail_instance", Name: "only-alpha", Region: "us-east-1",
		MonthlyCents: 1200, Source: finops.SourceEstimate,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceResources(accB, []models.CloudResource{{
		Kind: "lightsail_instance", Name: "only-beta", Region: "us-east-1",
		MonthlyCents: 700, Source: finops.SourceEstimate,
	}}); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListResourcesForTenant(alpha)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "only-alpha" {
		t.Fatalf("alpha resources = %#v, want only-alpha", got)
	}
}

func TestStore_EnsureCloudAccount_updatesExisting(t *testing.T) {
	s := newTestStore(t)
	tid, err := s.CreateTenant("Demo", "demo")
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.EnsureCloudAccount(models.CloudAccount{
		TenantID:     tid,
		AWSAccountID: "111111111111",
		Alias:        "principal",
		Region:       "us-east-1",
		AuthMode:     finops.AuthModeDefaultChain,
		IsPrimary:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.EnsureCloudAccount(models.CloudAccount{
		TenantID:     tid,
		AWSAccountID: "111111111111",
		Alias:        "principal",
		Region:       "us-east-1",
		AuthMode:     finops.AuthModeDefaultChain,
		IsPrimary:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("EnsureCloudAccount ids %d vs %d, want same", first, second)
	}
	accounts, err := s.ListCloudAccounts(tid)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 {
		t.Fatalf("accounts = %d, want 1", len(accounts))
	}
}

func TestStore_UpdateCloudAccountAuth_switchesToAccessKeys(t *testing.T) {
	s := newTestStore(t)
	tid, err := s.CreateTenant("Demo", "demo")
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.EnsureCloudAccount(models.CloudAccount{
		TenantID:     tid,
		AWSAccountID: "111111111111",
		Alias:        "principal",
		Region:       "us-east-1",
		AuthMode:     finops.AuthModeDefaultChain,
		IsPrimary:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateCloudAccountAuth(models.CloudAccount{
		TenantID:     tid,
		AWSAccountID: "111111111111",
		Alias:        "principal",
		Region:       "us-east-1",
		AuthMode:     finops.AuthModeAccessKeys,
		AccessKeyID:  "AKIATEST",
		SecretCipher: "cipher",
	}); err != nil {
		t.Fatal(err)
	}
	acc, err := s.FindCloudAccount(id)
	if err != nil {
		t.Fatal(err)
	}
	if acc.AuthMode != finops.AuthModeAccessKeys {
		t.Fatalf("auth_mode = %q, want access_keys", acc.AuthMode)
	}
	if acc.AccessKeyID != "AKIATEST" || acc.SecretCipher != "cipher" {
		t.Fatalf("keys = %q / %q", acc.AccessKeyID, acc.SecretCipher)
	}
}
