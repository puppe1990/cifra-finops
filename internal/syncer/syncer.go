package syncer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/puppe1990/cifra-finops/internal/awsinv"
	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/store"
)

type Syncer struct {
	store     store.Store
	collector awsinv.Collector
	decrypt   func(cipher string) (string, error)
}

func New(s store.Store, collector awsinv.Collector) *Syncer {
	return &Syncer{store: s, collector: collector}
}

func (s *Syncer) WithDecrypt(fn func(string) (string, error)) *Syncer {
	s.decrypt = fn
	return s
}

func (s *Syncer) SyncAccount(ctx context.Context, accountID int64) (run storeRun, err error) {
	acc, err := s.store.FindCloudAccount(accountID)
	if err != nil {
		return storeRun{}, err
	}
	runID, err := s.store.StartSyncRun(accountID)
	if err != nil {
		return storeRun{}, err
	}

	creds, err := s.credsFor(acc)
	if err != nil {
		_ = s.store.FinishSyncRun(runID, finops.SyncFailed, "", "", err.Error())
		return storeRun{Status: finops.SyncFailed, Error: err.Error()}, err
	}
	inv, err := s.collector.Collect(ctx, creds)
	if err != nil {
		_ = s.store.FinishSyncRun(runID, finops.SyncFailed, "", "", err.Error())
		return storeRun{Status: finops.SyncFailed, Error: err.Error()}, err
	}
	if err := s.store.ReplaceResources(accountID, inv.Resources); err != nil {
		return storeRun{}, err
	}
	if err := s.store.ReplaceCostLines(accountID, inv.Lines); err != nil {
		return storeRun{}, err
	}
	if err := s.store.ReplaceFindings(accountID, inv.Findings); err != nil {
		return storeRun{}, err
	}
	warning := strings.Join(inv.Warnings, "; ")
	if err := s.store.FinishSyncRun(runID, finops.SyncOK, inv.Source, warning, ""); err != nil {
		return storeRun{}, err
	}
	return storeRun{ID: runID, Status: finops.SyncOK, Source: inv.Source, Warning: warning}, nil
}

func (s *Syncer) credsFor(acc models.CloudAccount) (awsinv.Credentials, error) {
	creds := awsinv.Credentials{
		Mode:        acc.AuthMode,
		AccessKeyID: acc.AccessKeyID,
		Region:      acc.Region,
		AccountID:   acc.AWSAccountID,
	}
	if acc.AuthMode == finops.AuthModeAccessKeys && acc.SecretCipher != "" && s.decrypt != nil {
		secret, err := s.decrypt(acc.SecretCipher)
		if err != nil {
			return creds, err
		}
		creds.SecretAccessKey = secret
	}
	return creds, nil
}

func (s *Syncer) CostForMonth(ctx context.Context, tenantID int64, period time.Time) ([]models.CostLine, error) {
	mc, ok := s.collector.(awsinv.MonthCoster)
	if !ok {
		return nil, nil
	}
	accounts, err := s.store.ListCloudAccounts(tenantID)
	if err != nil {
		return nil, err
	}
	var all []models.CostLine
	for _, acc := range accounts {
		creds, err := s.credsFor(acc)
		if err != nil {
			return nil, fmt.Errorf("ce %s: %w", acc.AWSAccountID, err)
		}
		lines, err := mc.CostForMonth(ctx, creds, period)
		if err != nil {
			return nil, fmt.Errorf("ce %s: %w", acc.AWSAccountID, err)
		}
		all = append(all, lines...)
	}
	return all, nil
}

func (s *Syncer) CostAnomalies(ctx context.Context, tenantID int64, from, to time.Time) ([]awsinv.CostAnomaly, error) {
	af, ok := s.collector.(awsinv.AnomalyFinder)
	if !ok {
		return nil, nil
	}
	accounts, err := s.store.ListCloudAccounts(tenantID)
	if err != nil {
		return nil, err
	}
	var all []awsinv.CostAnomaly
	for _, acc := range accounts {
		creds, err := s.credsFor(acc)
		if err != nil {
			return nil, fmt.Errorf("anomalies %s: %w", acc.AWSAccountID, err)
		}
		items, err := af.CostAnomalies(ctx, creds, from, to)
		if err != nil {
			return nil, fmt.Errorf("anomalies %s: %w", acc.AWSAccountID, err)
		}
		all = append(all, items...)
	}
	return all, nil
}

func (s *Syncer) CostByMonth(ctx context.Context, tenantID int64, from, to time.Time) ([]models.CostLine, error) {
	rc, ok := s.collector.(awsinv.RangeCoster)
	if !ok {
		return nil, nil
	}
	accounts, err := s.store.ListCloudAccounts(tenantID)
	if err != nil {
		return nil, err
	}
	var all []models.CostLine
	for _, acc := range accounts {
		creds, err := s.credsFor(acc)
		if err != nil {
			return nil, fmt.Errorf("ce %s: %w", acc.AWSAccountID, err)
		}
		lines, err := rc.CostByMonth(ctx, creds, from, to)
		if err != nil {
			return nil, fmt.Errorf("ce %s: %w", acc.AWSAccountID, err)
		}
		all = append(all, lines...)
	}
	return all, nil
}

func (s *Syncer) ForecastForMonth(ctx context.Context, tenantID int64, period time.Time) (int64, error) {
	fc, ok := s.collector.(awsinv.CostForecaster)
	if !ok {
		return 0, nil
	}
	accounts, err := s.store.ListCloudAccounts(tenantID)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, acc := range accounts {
		creds, err := s.credsFor(acc)
		if err != nil {
			return 0, fmt.Errorf("forecast %s: %w", acc.AWSAccountID, err)
		}
		cents, err := fc.ForecastForMonth(ctx, creds, period)
		if err != nil {
			return 0, fmt.Errorf("forecast %s: %w", acc.AWSAccountID, err)
		}
		total += cents
	}
	return total, nil
}

func (s *Syncer) SyncTenant(ctx context.Context, tenantID int64) error {
	accounts, err := s.store.ListCloudAccounts(tenantID)
	if err != nil {
		return err
	}
	var first error
	for _, acc := range accounts {
		if _, err := s.SyncAccount(ctx, acc.ID); err != nil && first == nil {
			first = fmt.Errorf("sync %s: %w", acc.AWSAccountID, err)
		}
	}
	return first
}

type storeRun struct {
	ID      int64
	Status  string
	Source  string
	Warning string
	Error   string
}
