package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/syncer"
)

type recordingCollector struct {
	stubMonthCollector
	collects int
}

func (r *recordingCollector) Collect(_ context.Context, _ awsinv.Credentials) (awsinv.Inventory, error) {
	r.collects++
	return awsinv.Inventory{
		Source: finops.SourceCE,
		Lines: []models.CostLine{{
			Service: "Amazon Lightsail", MonthlyCents: 1947, Source: finops.SourceCE,
		}},
	}, nil
}

func TestDashboardHandler_syncsAWSWhenHetznerAlreadyHasInventory(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	tid, err := s.CreateTenant("Workspace", "ws-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddMember(tid, uid, finops.RoleOwner); err != nil {
		t.Fatal(err)
	}
	if err := s.SetActiveTenant(uid, tid); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, AWSAccountID: "840298254452", Alias: "principal",
		Region: "us-east-1", AuthMode: finops.AuthModeAccessKeys, IsPrimary: true,
	}); err != nil {
		t.Fatal(err)
	}
	hzID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, Provider: finops.ProviderHetzner, AWSAccountID: "hz:prod",
		Alias: "gestaobem", Region: "fsn1", AuthMode: finops.AuthModeAPIToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceResources(hzID, []models.CloudResource{{
		Kind: "hetzner_server", Name: "web", Region: "fsn1", State: "running",
		MonthlyCents: 500, Source: finops.SourceHetzner, ExternalID: "web",
	}}); err != nil {
		t.Fatal(err)
	}

	col := &recordingCollector{}
	h := NewDashboardHandler(setupTestRenderer(t), s, testSite(), cais.Config{}, setupTestViews(t)).
		WithSyncer(syncer.New(s, col))
	req := inertiaRequest(http.MethodGet, "/dashboard", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if col.collects == 0 {
		t.Fatal("AWS collect skipped because Hetzner already had resources")
	}
}

func TestLastSyncProps_prefersFailedRun(t *testing.T) {
	s := setupTestStore(t)
	tid, err := s.CreateTenant("Demo", "demo")
	if err != nil {
		t.Fatal(err)
	}
	awsID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, AWSAccountID: "840298254452", Alias: "principal",
		Region: "us-east-1", AuthMode: finops.AuthModeDefaultChain, IsPrimary: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	hzID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, Provider: finops.ProviderHetzner, AWSAccountID: "hz:prod",
		Alias: "gestaobem", Region: "fsn1", AuthMode: finops.AuthModeAPIToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	runAWS, err := s.StartSyncRun(awsID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FinishSyncRun(runAWS, finops.SyncFailed, "", "", "no EC2 IMDS role found"); err != nil {
		t.Fatal(err)
	}
	runHZ, err := s.StartSyncRun(hzID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FinishSyncRun(runHZ, finops.SyncOK, finops.SourceHetzner, "", ""); err != nil {
		t.Fatal(err)
	}
	accounts, err := s.ListCloudAccounts(tid)
	if err != nil {
		t.Fatal(err)
	}
	got := lastSyncProps(s, accounts)
	if got["status"] != finops.SyncFailed {
		t.Fatalf("status=%v want failed (AWS error hidden behind Hetzner ok)", got["status"])
	}
	if !strings.Contains(got["error"].(string), "IMDS") {
		t.Fatalf("error=%v", got["error"])
	}
}

func TestLastSyncProps_prefersAWSWarningOverLaterOK(t *testing.T) {
	s := setupTestStore(t)
	tid, err := s.CreateTenant("Demo", "demo-w")
	if err != nil {
		t.Fatal(err)
	}
	awsID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, AWSAccountID: "840298254452", Alias: "principal",
		Region: "us-east-1", AuthMode: finops.AuthModeDefaultChain, IsPrimary: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	hzID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, Provider: finops.ProviderHetzner, AWSAccountID: "hz:prod",
		Alias: "gestaobem", Region: "fsn1", AuthMode: finops.AuthModeAPIToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	runAWS, err := s.StartSyncRun(awsID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FinishSyncRun(runAWS, finops.SyncOK, finops.SourceEstimate, "sts: no EC2 IMDS role found", ""); err != nil {
		t.Fatal(err)
	}
	runHZ, err := s.StartSyncRun(hzID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FinishSyncRun(runHZ, finops.SyncOK, finops.SourceHetzner, "", ""); err != nil {
		t.Fatal(err)
	}
	accounts, err := s.ListCloudAccounts(tid)
	if err != nil {
		t.Fatal(err)
	}
	got := lastSyncProps(s, accounts)
	if !strings.Contains(got["warning"].(string), "IMDS") {
		t.Fatalf("warning=%v (Hetzner ok hid the AWS credential warning)", got["warning"])
	}
}

func TestDashboardHandler_rendersFailedAWSSync(t *testing.T) {
	s := setupTestStore(t)
	uid, err := s.CreateUser("ops@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	tid, err := s.CreateTenant("Workspace", "ws-2")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddMember(tid, uid, finops.RoleOwner); err != nil {
		t.Fatal(err)
	}
	if err := s.SetActiveTenant(uid, tid); err != nil {
		t.Fatal(err)
	}
	awsID, err := s.CreateCloudAccount(models.CloudAccount{
		TenantID: tid, AWSAccountID: "840298254452", Alias: "principal",
		Region: "us-east-1", AuthMode: finops.AuthModeDefaultChain, IsPrimary: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	runID, err := s.StartSyncRun(awsID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FinishSyncRun(runID, finops.SyncFailed, "", "", "no EC2 IMDS role found"); err != nil {
		t.Fatal(err)
	}

	h := NewDashboardHandler(setupTestRenderer(t), s, testSite(), cais.Config{}, setupTestViews(t))
	req := inertiaRequest(http.MethodGet, "/dashboard", nil)
	req = session.WithUserID(req, uid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if !strings.Contains(rr.Body.String(), `data-sync-error`) {
		t.Fatalf("failed AWS sync not rendered: %s", rr.Body.String())
	}
}
