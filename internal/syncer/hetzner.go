package syncer

import (
	"context"
	"fmt"
	"strings"

	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
)

func isAWSAccount(acc models.CloudAccount) bool {
	return acc.Provider != finops.ProviderHetzner
}

func (s *Syncer) finishHetzner(ctx context.Context, acc models.CloudAccount, runID int64) (storeRun, error) {
	if s.hetzner == nil {
		return s.failRun(runID, fmt.Errorf("hetzner collector is not configured"))
	}
	token, err := s.hetznerToken(acc)
	if err != nil {
		return s.failRun(runID, err)
	}
	inv, err := s.hetzner.Collect(ctx, token)
	if err != nil {
		return s.failRun(runID, err)
	}
	return s.persistInventory(acc.ID, runID, inv.Source, inv.Resources, inv.Lines, inv.Findings, inv.Warnings)
}

func (s *Syncer) hetznerToken(acc models.CloudAccount) (string, error) {
	if acc.SecretCipher == "" {
		return "", fmt.Errorf("hetzner api token is missing")
	}
	if s.decrypt == nil {
		return "", fmt.Errorf("cannot decrypt hetzner api token")
	}
	token, err := s.decrypt(acc.SecretCipher)
	if err != nil {
		return "", fmt.Errorf("decrypt hetzner api token: %w", err)
	}
	return token, nil
}

func (s *Syncer) persistInventory(accountID, runID int64, source string, resources []models.CloudResource, lines []models.CostLine, findings []models.Finding, warnings []string) (storeRun, error) {
	if err := s.store.ReplaceResources(accountID, resources); err != nil {
		return storeRun{}, err
	}
	if err := s.store.ReplaceCostLines(accountID, lines); err != nil {
		return storeRun{}, err
	}
	if err := s.store.ReplaceFindings(accountID, findings); err != nil {
		return storeRun{}, err
	}
	warning := strings.Join(warnings, "; ")
	if err := s.store.FinishSyncRun(runID, finops.SyncOK, source, warning, ""); err != nil {
		return storeRun{}, err
	}
	return storeRun{ID: runID, Status: finops.SyncOK, Source: source, Warning: warning}, nil
}

func (s *Syncer) failRun(runID int64, err error) (storeRun, error) {
	_ = s.store.FinishSyncRun(runID, finops.SyncFailed, "", "", err.Error())
	return storeRun{Status: finops.SyncFailed, Error: err.Error()}, err
}
