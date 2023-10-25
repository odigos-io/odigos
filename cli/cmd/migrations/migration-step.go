package migrations

import (
	"context"

	"github.com/keyval-dev/odigos/cli/pkg/kube"
)

type MigrationStep interface {

	// the version from which the migration should take place.
	// if the upgrade is from v0.1.2 to v0.1.3, then this should return v0.1.2
	SourceVersion() string

	// a name for the patcher which can be used for logging purposes.
	MigrationName() string

	Migrate(ctx context.Context, client *kube.Client, odigosNs string) error
	Rollback(ctx context.Context, client *kube.Client, odigosNs string) error
}
