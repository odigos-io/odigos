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

// Details is used to represent the details of a process.
// It may include different information depending on the platform (Kubernetes, VM, etc).
type Details interface {
	fmt.Stringer
}

// ConfigGroup is used to represent a group of instrumented processes which can be configured together.
// For example, a Kubernetes deployment with multiple replicas can be considered a ConfigGroup.
type ConfigGroup interface {
	comparable
}

// ProcessDetailsResolver is used to resolve the details of a process.
type ProcessDetailsResolver[details Details] interface {
	// Resolve will fetch the relevant details of a process.
	// Those details may be used for future calls when reporting the status of the instrumentation.
	// or for resolving the configuration group of the process.
	Resolve(context.Context, detector.ProcessEvent) (details, error)
}

// ConfigGroupResolver is used to resolve the configuration group of a process.
type ConfigGroupResolver[details Details, configGroup ConfigGroup] interface {
	// Resolve will classify the process into a configuration group.
	// The Otel Distribution is resolved in the time of calling this function, and may be used
	// to determine the configuration group.
	Resolve(context.Context, details, OtelDistribution) (configGroup, error)
}

// Reporter is used to report the status of the instrumentation.
// It is called at different stages of the instrumentation lifecycle.
type Reporter[details Details] interface {
	// OnInit is called when the instrumentation is initialized.
	// The error parameter will be nil if the instrumentation was initialized successfully.
	OnInit(ctx context.Context, pid int, err error, d details) error

	// OnLoad is called after an instrumentation is loaded successfully or failed to load.
	// The error parameter will be nil if the instrumentation was loaded successfully.
	OnLoad(ctx context.Context, pid int, err error, d details) error

	// OnRun is called after the instrumentation stops running.
	// An error may report a fatal error during the instrumentation run, or a closing error
	// which happened during the closing of the instrumentation.
	OnRun(ctx context.Context, pid int, err error, d details) error

	// OnExit is called when the instrumented process exits, and the instrumentation has already been stopped.
	// For a reported which persists the instrumentation state, this is the time to clean up the state.
	OnExit(ctx context.Context, pid int, e details) error
}

// DistributionMatcher is used to match a process to an Otel Distribution.
type DistributionMatcher[details Details] interface {
	// Distribution will match a process to an Otel Distribution.
	Distribution(context.Context, details) (OtelDistribution, error)
}

// SettingsGetter is used to fetch the initial settings of an instrumentation.
type SettingsGetter[details Details] interface {
	// GetSettings will fetch the initial settings of an instrumentation.
	Settings(context.Context, details, OtelDistribution) (Settings, error)
}

// Handler is used to classify, report and configure instrumentations.
type Handler[details Details, configGroup comparable] struct {
	DetailsResolver     ProcessDetailsResolver[details]
	ConfigGroupResolver ConfigGroupResolver[details, configGroup]
	Reporter            Reporter[details]
	DistributionMatcher DistributionMatcher[details]
	SettingsGetter	    SettingsGetter[details]
}