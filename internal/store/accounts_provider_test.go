package store

import (
	"testing"

	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func TestSQLiteStore_persistsHetznerProvider(t *testing.T) {
	s := newTestStore(t)
	tid, err := s.CreateTenant("Demo", "demo")
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID:     tid,
		Provider:     finops.ProviderHetzner,
		AWSAccountID: "hz:prod",
		Alias:        "prod",
		Region:       "fsn1",
		AuthMode:     finops.AuthModeAPIToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.FindCloudAccount(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != finops.ProviderHetzner {
		t.Fatalf("provider = %q", got.Provider)
	}
	if got.AWSAccountID != "hz:prod" || got.AuthMode != finops.AuthModeAPIToken {
		t.Fatalf("account = %#v", got)
	}
}

func TestSQLiteStore_defaultsProviderToAWS(t *testing.T) {
	s := newTestStore(t)
	tid, err := s.CreateTenant("Demo", "demo")
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID:     tid,
		AWSAccountID: "111111111111",
		Alias:        "principal",
		Region:       "us-east-1",
		AuthMode:     finops.AuthModeDefaultChain,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.FindCloudAccount(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != finops.ProviderAWS {
		t.Fatalf("provider = %q, want aws", got.Provider)
	}
}
