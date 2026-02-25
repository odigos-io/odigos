package instrumentation

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentation/detector"
)

var (
	// ErrProcessLanguageNotMatchesDistribution is returned when the detected programming language of a process
	// does not match the language of the distribution determined for instrumentation.
	// This can happen in scenarios where a process spawns child processes of different languages.
	// For example, a bash script that launches a python process - The distribution will be python but the bash process will not match.
	ErrProcessLanguageNotMatchesDistribution = fmt.Errorf("the detected programming language of the process does not match the language of the distribution determined for instrumentation")
)

// ProcessDetails is used to convert the common process details reported by the detector to details relevant to hosting platform.
//
// ProcessDetails can contain details that associates a process to a group of processes that are managed together by the hosting platform.
// It may include different information depending on the platform (Kubernetes, VM, etc).
//
// Another category of information that may be included relates to language and runtime information which can be used to
// determine the OTel distribution to use.
type ProcessDetails[processGroup ProcessGroup, configGroup ConfigGroup] interface {
	fmt.Stringer
	// ConfigGroup will classify the process into a configuration group.
	ConfigGroup(context.Context) (configGroup, error)
	// ProcessGroup will classify the process into a process group.
	ProcessGroup(context.Context) (processGroup, error)
	// Distribution will match a process to an Otel Distribution.
	Distribution(context.Context) (*distro.OtelDistro, error)
}

// ConfigGroup is used to represent a group of instrumented processes which can be configured together.
// i.e when an instrumentation config changes - all the processes within the same config group can be configured to the new configuration.
//
// For example, a Kubernetes deployment with multiple replicas can be considered a ConfigGroup.
// Additional data such as OTel distribution or programming language may be used to make the ConfigGroup more specific than a ProcessGroup.
// A config group may include multiple process groups, but there is no assumption on a connection between them.
type ConfigGroup interface {
	comparable
}

// ProcessGroup represents a group of processes that are part of the same workload (e.g Deployment replica, systemd service)
// A ProcessGroup can be considered for a batch operation of instrumenting or un-instrumenting all the processes in the group.
//
// For example consider an app which is launched by a bash script, the script launches a python process.
// The process may create different child processes, and the bash script may launch multiple python processes.
// In this case, the process group may include the bash script, the python process, and the child processes.
type ProcessGroup interface {
	comparable
}

// ProcessDetailsResolver is used to resolve the required details based on the detector event for a new process
type ProcessDetailsResolver[processGroup ProcessGroup, configGroup ConfigGroup, processDetails ProcessDetails[processGroup, configGroup]] interface {
	// Resolve will classify the process into a process group.
	// Those process group details may be used for future calls when reporting the status of the instrumentation.
	// or for resolving the configuration group of the process.
	Resolve(context.Context, detector.ProcessEvent) (processDetails, error)
}

// Reporter is used to report the status of the instrumentation.
// It is called at different stages of the instrumentation lifecycle.
type Reporter[processGroup ProcessGroup, configGroup ConfigGroup, processDetails ProcessDetails[processGroup, configGroup]] interface {
	// OnInit is called when the instrumentation is initialized.
	// The error parameter will be nil if the instrumentation was initialized successfully.
	OnInit(ctx context.Context, pid int, err error, pg processDetails) error

	// OnLoad is called after an instrumentation is loaded successfully or failed to load.
	// The error parameter will be nil if the instrumentation was loaded successfully.
	OnLoad(ctx context.Context, pid int, err error, pg processDetails, status Status) error

	// OnRun is called after the instrumentation stops running.
	// An error may report a fatal error during the instrumentation run, or a closing error
	// which happened during the closing of the instrumentation.
	OnRun(ctx context.Context, pid int, err error, pg processDetails) error

	// OnExit is called when the instrumented process exits, and the instrumentation has already been stopped.
	// For a reported which persists the instrumentation state, this is the time to clean up the state.
	OnExit(ctx context.Context, pid int, pg processDetails) error
}

// StatusReporter is an optional extension of Reporter for instrumentations that
// support asynchronous status updates (e.g., lazy-loaded libraries).
// Implementations of Reporter can optionally implement this interface to receive
// status updates that occur after the initial Load phase.
type StatusReporter[processGroup ProcessGroup, configGroup ConfigGroup, processDetails ProcessDetails[processGroup, configGroup]] interface {
	// OnStatusUpdate is called when the instrumentation status changes after the initial load.
	// For example, when lazily-loaded libraries become active.
	OnStatusUpdate(ctx context.Context, pid int, pg processDetails, status Status) error
}

// SettingsGetter is used to fetch the initial settings of an instrumentation.
type SettingsGetter[processGroup ProcessGroup, configGroup ConfigGroup, processDetails ProcessDetails[processGroup, configGroup]] interface {
	// GetSettings will fetch the initial settings of an instrumentation.
	Settings(context.Context, logr.Logger, processDetails, *distro.OtelDistro) (Settings, error)
}

// Handler is used to classify, report and configure instrumentations.
type Handler[processGroup ProcessGroup, configGroup ConfigGroup, processDetails ProcessDetails[processGroup, configGroup]] struct {
	ProcessDetailsResolver ProcessDetailsResolver[processGroup, configGroup, processDetails]
	Reporter               Reporter[processGroup, configGroup, processDetails]
	SettingsGetter         SettingsGetter[processGroup, configGroup, processDetails]
}
