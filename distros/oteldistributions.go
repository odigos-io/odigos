package distros

import "github.com/odigos-io/odigos/common"

type RuntimeEnvironment struct {
	// the runtime environment this distribution targets.
	// examples: nodejs, JVM, CPython, etc.
	// while java-script can run in both nodejs and browser, the distribution should specify where it is intended to run.
	RuntimeEnvironmentName string `json:"runtimeEnvironmentName"`

	// semconv range of the runtime versions supported by this distribution.
	RuntimeEnvironmentVersion string `json:"runtimeEnvironmentVersion"`
}

type Framework struct {
	// the framework this distribution targets.
	FrameworkName string `json:"frameworkName"`

	// semconv range of the framework versions supported by this distribution.
	FrameworkVersion string `json:"frameworkVersion"`
}

// this struct describes environment variables that needs to be set in the application runtime
// to enable the distribution.
type EnvironmentVariable struct {

	// The name of the environment variable to set or patch.
	EnvName string `json:"envName"`

	// The value of the environment variable to set or patch.
	// One special value can be used in this text which is substituted by the actual value at runtime.
	// The special value is: `{{ODIGOS_AGENTS_DIR}}` which is replaced by `/var/odigos`, for k8s and with other values for other platforms.
	EnvValue string `json:"envValue"`

	// In case the environment variable needs to be appended to an existing value,
	// this field specifies the delimiter to use.
	// e.g. `:` for PYTHONPATH=/path/to/lib1:/path/to/lib2
	Delimiter string `json:"delimiter"`
}

// OtelDistro (Short for OpenTelemetry Distribution) is a collection of OpenTelemetry components,
// including instrumentations, SDKs, and other components that are distributed together.
// Each distribution includes a unique name, and metadata about the ways it is implemented.
// The metadata includes the tiers of the distribution, the instrumentations, and the SDKs used.
// Multiple distributions can co-exist with the same properties but different names.
type OtelDistro struct {

	// a unique name for this distribution, which helps to identify it.
	// should be a single word, lowercase, and may include hyphens (nodejs-community, dotnet-legacy-instrumentation).
	Name string `json:"name"`

	// the programming language this distribution targets.
	// each distribution must target a single language.
	Language common.ProgrammingLanguage `json:"language"`

	// the runtime environments this distribution targets.
	// examples: nodejs, JVM, CPython, etc.
	// while java-script can run in both nodejs and browser, the distribution should specify where it is intended to run.
	RuntimeEnvironments []RuntimeEnvironment `json:"runtimeEnvironments"`

	// A list of frameworks this distribution targets (can be left empty)
	Frameworks []Framework `json:"frameworks"`

	// a human-friendly name for this distribution, which can be displayed in the UI and documentation.
	// may include spaces and special characters.
	DisplayName string `json:"displayName"`

	// Free text description of the distribution, what it includes, it's use cases, etc.
	Description string `json:"description"`

	// Specifies the odigos tiers which includes this distribution
	Tiers []common.OdigosTier `json:"tiers"`

	// a list of environment variables that needs to be set in the application runtime
	// to enable the distribution.
	EnvironmentVariables []EnvironmentVariable `json:"environmentVariables,omitempty"`
}
