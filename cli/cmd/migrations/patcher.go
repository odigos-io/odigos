package migrations

import "context"

type Patcher interface {
	PatcherName() string
	Patch(ctx context.Context) error
	UnPatch(ctx context.Context) error
}
