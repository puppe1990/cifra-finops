package db

import (
	"github.com/puppe1990/cifra-finops/internal/models"
	"github.com/puppe1990/cifra-finops/internal/seed"
	"github.com/puppe1990/cifra-finops/internal/store"
)

// RunSeeds populates demo data. Safe to run multiple times.
func RunSeeds(s store.Store) error {
	// cais:recurring-seeds
	// cais:seeds
	if _, err := s.InsertContact(models.Contact{
		Name:  "Demo",
		Email: "demo@example.com",
	}); err != nil {
		return err
	}
	user, err := s.FindUserByEmail("demo@example.com")
	if err != nil {
		return nil
	}
	return seed.EnsurePrimaryWorkspace(s, user.ID)
}
