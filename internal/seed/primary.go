package seed

import (
	"fmt"

	"github.com/puppe1990/cifra-finops/internal/finops"
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/store"
)

func EnsurePrimaryWorkspace(s store.Store, ownerUserID int64) error {
	tenant, err := s.FindTenantBySlug(finops.PrimaryTenantSlug)
	if err != nil {
		id, createErr := s.CreateTenant(finops.PrimaryTenantName, finops.PrimaryTenantSlug)
		if createErr != nil {
			return fmt.Errorf("create primary tenant: %w", createErr)
		}
		tenant, err = s.FindTenantByID(id)
		if err != nil {
			return err
		}
	}
	if err := s.AddMember(tenant.ID, ownerUserID, finops.RoleOwner); err != nil {
		return err
	}
	if err := s.SetActiveTenant(ownerUserID, tenant.ID); err != nil {
		return err
	}
	budget := models.Budget{
		TenantID:    tenant.ID,
		Name:        "Mensal principal",
		AmountCents: 5000,
		Period:      "monthly",
	}
	if id := finops.SeedAWSAccountID(); finops.ValidAWSAccountID(id) {
		accID, err := s.EnsureCloudAccount(models.CloudAccount{
			TenantID:     tenant.ID,
			AWSAccountID: id,
			Alias:        finops.PrimaryAccountAlias,
			Region:       finops.DefaultRegion,
			AuthMode:     finops.AuthModeDefaultChain,
			IsPrimary:    true,
		})
		if err != nil {
			return err
		}
		budget.CloudAccountID = accID
	}
	return s.EnsureBudget(budget)
}
