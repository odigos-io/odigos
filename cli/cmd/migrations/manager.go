package migrations

import (
	"fmt"

	"github.com/odigos-io/odigos/cli/cmd/migrations/runtime_details_migration"
	"github.com/odigos-io/odigos/cli/pkg/kube"

	"golang.org/x/mod/semver"
)

type MigrationManager struct {
	Migrations []Migration
}

func NewMigrationManager(client *kube.Client) *MigrationManager {
	return &MigrationManager{
		Migrations: []Migration{
			&runtime_details_migration.MigrateRuntimeDetails{Client: client},
			// Add more migrations here by referencing their structs
		},
	}
}
func (m *MigrationManager) Run(fromVersion, toVersion string) error {
	// Ensure versions are valid semantic versions
	if !semver.IsValid(fromVersion) {
		return fmt.Errorf("invalid from version: %s", fromVersion)
	}
	if !semver.IsValid(toVersion) {
		return fmt.Errorf("invalid to version: %s", toVersion)
	}

	for _, migration := range m.Migrations {
		cutVersion := migration.TriggerVersion()
		if !semver.IsValid(cutVersion) {
			return fmt.Errorf("invalid cut version for migration %s: %s", migration.Name(), cutVersion)
		}

		// Check if migration should be executed
		if semver.Compare(fromVersion, cutVersion) < 0 && semver.Compare(toVersion, cutVersion) >= 0 {
			fmt.Printf("Executing migration: %s - %s\n", migration.Name(), migration.Description())
			if err := migration.Execute(); err != nil {
				return fmt.Errorf("migration %s failed: %w", migration.Name(), err)
			}
			fmt.Printf("Migration %s completed successfully.\n", migration.Name())
		}
	}
	return nil
}
