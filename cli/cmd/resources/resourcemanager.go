package resources

import (
	"context"

	"github.com/keyval-dev/odigos/cli/cmd/migrations"
)

type MigrationStep struct {
	// The version of the ** source ** of the migration step.
	// For example - if the migration step is from version v1.0.0 to v1.1.0, the source version is v1.0.0.
	SourceVersion string

	Patchers []migrations.Patcher
}

type ResourceManager interface {

	// This function is being called to install the resource from scratch.
	// It should create all the required resources in the cluster, and return an error if the installation failed.
	// This function will only be invoked with `install`, thus it can assume that the resource is not installed in the cluster yet.
	// It is, however, preferable to make this function idempotent, so it can be invoked multiple times without causing any harm.
	InstallFromScratch(ctx context.Context) error

	// This function is being called to apply a migration step from a source version to the next version.
	// It should invoke any patches to the resource being managed, and return an error if the migration step failed.
	// You can read about patch options
	// [here](https://erosb.github.io/post/json-patch-vs-merge-patch/). and [here](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/)
	// Such examples are:
	// 1. CRDs migration - patching the SDR with the new SDR version.
	// 2. New objects, as of a specific version - like config maps, secrets, deployments, etc.
	// 3. Semantic changes to existing objects - like adding a new field spec or change object values.
	// 4. Any other change that is required to be applied to the cluster as part of the migration.
	//
	// This function can be a noop if no migration step is required for the current version.
	// The function should return an error if the migration step failed.
	// The function should return nil if the migration step was applied successfully.
	// ApplyMigrationStep(ctx context.Context, sourceVersion string) error

	// This function is being called to rollback a migration step to a source version from the next version.
	// Every step that is being applied in ApplyMigrationStep should have a rollback step here.
	// It should invoke any patches to the resource being managed, and return an error if the rollback step failed.
	// This function must be idempotent, it cannot assume if the migration step was applied successfully or not,
	// and should thus be able to set the resource to the desired state regardless of the current state.
	// RollbackMigrationStep(ctx context.Context, sourceVersion string) error

	GetMigrationSteps() []MigrationStep

	// This function is being called to explicitly set the Odigos version in the cluster.
	// It allows us to skip bumping the version for each migration step, and instead
	// set the version to the target version once all migrations are complete.
	// If the resource being managed is not an odigos deployment, this function should be a noop.
	// Any other object fields that needs to be updated with values which are not odigos version,
	// should be patched as a migration step.
	PatchOdigosVersionToTarget(ctx context.Context, newOdigosVersion string) error
}
