package syncer

import (
	"context"
	"testing"
	"time"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/hetznerinv"
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/store"
)

type stubHetzner struct {
	inv   hetznerinv.Inventory
	err   error
	token string
}

func (s *stubHetzner) Collect(_ context.Context, token string) (hetznerinv.Inventory, error) {
	s.token = token
	return s.inv, s.err
}

type trackingAWS struct {
	stubCollector
	calls int
}

func (s *trackingAWS) Collect(ctx context.Context, creds awsinv.Credentials) (awsinv.Inventory, error) {
	s.calls++
	return s.stubCollector.Collect(ctx, creds)
}

func TestSyncer_syncsHetznerAccountWithToken(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	tid, err := s.CreateTenant("Demo", "demo")
	if err != nil {
		t.Fatal(err)
	}
	accID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID:     tid,
		Provider:     finops.ProviderHetzner,
		AWSAccountID: "hz:prod",
		Alias:        "prod",
		Region:       "fsn1",
		AuthMode:     finops.AuthModeAPIToken,
		SecretCipher: "cipher",
	})
	if err != nil {
		t.Fatal(err)
	}

	hz := &stubHetzner{inv: hetznerinv.Inventory{
		Source: finops.SourceHetzner,
		Resources: []models.CloudResource{{
			Kind: "hetzner_server", Name: "web-1", Region: "fsn1",
			MonthlyCents: 349, Source: finops.SourceHetzner,
		}},
		Lines: []models.CostLine{{
			Service: "Cloud Server", MonthlyCents: 349, Source: finops.SourceHetzner,
		}},
	}}
	aws := &trackingAWS{}
	syn := New(s, aws).WithHetzner(hz).WithDecrypt(func(cipher string) (string, error) {
		if cipher != "cipher" {
			t.Fatalf("cipher = %q", cipher)
		}
		return "tok-123", nil
	})
	run, err := syn.SyncAccount(context.Background(), accID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != finops.SyncOK {
		t.Fatalf("status = %q", run.Status)
	}
	if hz.token != "tok-123" {
		t.Fatalf("token = %q", hz.token)
	}
	if aws.calls != 0 {
		t.Fatalf("aws collector called %d times", aws.calls)
	}
	lines, err := s.ListCostLines(accID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0].Service != "Cloud Server" || lines[0].MonthlyCents != 349 {
		t.Fatalf("lines = %#v", lines)
	}
}

type trackingMonth struct {
	stubCollector
	accountIDs []string
}

func (s *trackingMonth) CostForMonth(_ context.Context, creds awsinv.Credentials, _ time.Time) ([]models.CostLine, error) {
	s.accountIDs = append(s.accountIDs, creds.AccountID)
	return nil, nil
}

func TestSyncer_CostForMonth_skipsHetznerAccounts(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	tid, err := s.CreateTenant("Demo", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, Provider: finops.ProviderAWS, AWSAccountID: "111111111111",
		Alias: "aws", Region: "us-east-1", AuthMode: finops.AuthModeDefaultChain,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, Provider: finops.ProviderHetzner, AWSAccountID: "hz:prod",
		Alias: "prod", Region: "fsn1", AuthMode: finops.AuthModeAPIToken, SecretCipher: "cipher",
	}); err != nil {
		t.Fatal(err)
	}
	col := &trackingMonth{}
	_, err = New(s, col).CostForMonth(context.Background(), tid, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(col.accountIDs) != 1 || col.accountIDs[0] != "111111111111" {
		t.Fatalf("ce accounts = %#v", col.accountIDs)
	}
}
