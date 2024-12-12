package instrumentation

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/instrumentation/detector"
)

// OtelDistribution is a customized version of an OpenTelemetry component.
// see https://opentelemetry.io/docs/concepts/distributions  and https://github.com/odigos-io/odigos/pull/1776#discussion_r1853367917 for more information.
// TODO: This should be moved to a common root package, since it will require a bigger refactor across multiple components,
// we use this local definition for now.
type OtelDistribution struct {
	Language common.ProgrammingLanguage
	OtelSdk  common.OtelSdk
}

type Details interface {
	fmt.Stringer
}

type ConfigGroup interface {
	comparable
}

type ProcessDetailsResolver[details Details] interface {
	Resolve(context.Context, detector.ProcessEvent) (details, error)
}

type ConfigGroupResolver[details Details, configGroup ConfigGroup] interface {
	Resolve(context.Context, details, OtelDistribution) (configGroup, error)
}

type Reporter[details Details] interface {
	OnInit(ctx context.Context, pid int, err error, d details) error
	OnLoad(ctx context.Context, pid int, err error, d details) error
	OnRun(ctx context.Context, pid int, err error, d details) error
	OnExit(ctx context.Context, pid int, e details) error
}

type DistributionMatcher[details Details] interface {
	Distribution(context.Context, details) (OtelDistribution, error)
}

type SettingsGetter[details Details] interface {
	Settings(context.Context, details, OtelDistribution) (Settings, error)
}

type Handler[details Details, configGroup comparable] struct {
	DetailsResolver     ProcessDetailsResolver[details]
	ConfigGroupResolver ConfigGroupResolver[details, configGroup]
	Reporter            Reporter[details]
	DistributionMatcher DistributionMatcher[details]
	SettingsGetter	    SettingsGetter[details]
}
