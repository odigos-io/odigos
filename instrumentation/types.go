package instrumentation

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentation/detector"
)

// OtelDistribution is a customized version of an OpenTelemetry component.
// see https://opentelemetry.io/docs/concepts/distributions  and https://github.com/odigos-io/odigos/pull/1776#discussion_r1853367917 for more information.
// TODO: This should be moved to a common root package, since it will require a bigger refactor across multiple components,
// we use this local definition for now.
type OtelDistribution struct {
	Language common.ProgrammingLanguage
	OtelSdk  common.OtelSdk
}

// ProcessGroup represents a group of processes that are managed together by the hosting platform.
// It may include different information depending on the platform (Kubernetes, VM, etc).
//
// For example consider an app which is launched by a bash script, the script launches a python process.
// The process may create different child processes, and the bash script may launch multiple python processes.
// In this case, the process group may include the bash script, the python process, and the child processes.
type ProcessGroup interface {
	fmt.Stringer
}

// ConfigGroup is used to represent a group of instrumented processes which can be configured together.
// For example, a Kubernetes deployment with multiple replicas can be considered a ConfigGroup.
// A config group may include multiple process groups, but there is no assumption on a connection between them.
type ConfigGroup interface {
	comparable
}

// ProcessGroupResolver is used to resolve the process group of a process.
type ProcessGroupResolver[processGroup ProcessGroup] interface {
	// Resolve will classify the process into a process group.
	// Those process group details may be used for future calls when reporting the status of the instrumentation.
	// or for resolving the configuration group of the process.
	Resolve(context.Context, detector.ProcessEvent) (processGroup, error)
}

// ConfigGroupResolver is used to resolve the configuration group of a process.
type ConfigGroupResolver[processGroup ProcessGroup, configGroup ConfigGroup] interface {
	// Resolve will classify the process into a configuration group.
	// The Otel Distribution is resolved in the time of calling this function, and may be used
	// to determine the configuration group.
	Resolve(context.Context, processGroup, OtelDistribution) (configGroup, error)
}

// Reporter is used to report the status of the instrumentation.
// It is called at different stages of the instrumentation lifecycle.
type Reporter[processGroup ProcessGroup] interface {
	// OnInit is called when the instrumentation is initialized.
	// The error parameter will be nil if the instrumentation was initialized successfully.
	OnInit(ctx context.Context, pid int, err error, pg processGroup) error

	// OnLoad is called after an instrumentation is loaded successfully or failed to load.
	// The error parameter will be nil if the instrumentation was loaded successfully.
	OnLoad(ctx context.Context, pid int, err error, pg processGroup) error

	// OnRun is called after the instrumentation stops running.
	// An error may report a fatal error during the instrumentation run, or a closing error
	// which happened during the closing of the instrumentation.
	OnRun(ctx context.Context, pid int, err error, pg processGroup) error

	// OnExit is called when the instrumented process exits, and the instrumentation has already been stopped.
	// For a reported which persists the instrumentation state, this is the time to clean up the state.
	OnExit(ctx context.Context, pid int, pg processGroup) error
}

// DistributionMatcher is used to match a process to an Otel Distribution.
type DistributionMatcher[processGroup ProcessGroup] interface {
	// Distribution will match a process to an Otel Distribution.
	Distribution(context.Context, processGroup) (OtelDistribution, error)
}

// SettingsGetter is used to fetch the initial settings of an instrumentation.
type SettingsGetter[processGroup ProcessGroup] interface {
	// GetSettings will fetch the initial settings of an instrumentation.
	Settings(context.Context, processGroup, OtelDistribution) (Settings, error)
}

// Handler is used to classify, report and configure instrumentations.
type Handler[processGroup ProcessGroup, configGroup comparable] struct {
	ProcessGroupResolver ProcessGroupResolver[processGroup]
	ConfigGroupResolver  ConfigGroupResolver[processGroup, configGroup]
	Reporter             Reporter[processGroup]
	DistributionMatcher  DistributionMatcher[processGroup]
	SettingsGetter       SettingsGetter[processGroup]
}
