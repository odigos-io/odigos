package distro

import "github.com/odigos-io/odigos/common"

const AgentPlaceholderDirectory = "{{ODIGOS_AGENTS_DIR}}"

type RuntimeEnvironment struct {
	// the runtime environment this distribution targets.
	// examples: nodejs, JVM, CPython, etc.
	// while java-script can run in both nodejs and browser, the distribution should specify where it is intended to run.
	Name string `yaml:"name"`

	// semconv range of the runtime versions supported by this distribution.
	SupportedVersions string `yaml:"supportedVersions,omitempty"`
}

type Framework struct {
	// the framework this distribution targets.
	FrameworkName string `yaml:"frameworkName"`

	// semconv range of the framework versions supported by this distribution.
	FrameworkVersion string `yaml:"frameworkVersion"`
}

// this struct describes environment variables that needs to be set in the application runtime
// to enable the distribution.
type EnvironmentVariable struct {

	// The name of the environment variable to set or patch.
	EnvName string `yaml:"envName"`

	// The value of the environment variable to set or patch.
	// One special value can be used in this text which is substituted by the actual value at runtime.
	// The special value is: `{{ODIGOS_AGENTS_DIR}}` which is replaced by `/var/odigos`, for k8s and with other values for other platforms.
	EnvValue string `yaml:"envValue"`

	// In case the environment variable needs to be appended to an existing value,
	// this field specifies the delimiter to use.
	// e.g. `:` for PYTHONPATH=/path/to/lib1:/path/to/lib2
	Delimiter string `yaml:"delimiter"`
}

type RuntimeAgent struct {
	// The name of a directory where odigos agent files can be found.
	// The special value {{ODIGOS_AGENTS_DIR}} is replaced with the actual value at this platform.
	// K8s will mount this directory from the node fs to the container. other platforms may have different ways to make the directory accessible.
	DirectoryNames []string `yaml:"directoryNames"`

	// This field indicates that the agent populates k8s resource attributes via environment variables.
	// It targets distros odigos uses without wrapper or customization.
	// For eBPF, the resource attributes are set in code.
	// For opamp distros, the resource attributes are set in the opamp server.
	// We will eventually remove this field once all distros upgrade to dynamic resource attributes.
	K8sAttrsViaEnvVars bool `yaml:"k8sAttrsViaEnvVars,omitempty"`
}

// OtelDistro (Short for OpenTelemetry Distribution) is a collection of OpenTelemetry components,
// including instrumentations, SDKs, and other components that are distributed together.
// Each distribution includes a unique name, and metadata about the ways it is implemented.
// The metadata includes the tiers of the distribution, the instrumentations, and the SDKs used.
// Multiple distributions can co-exist with the same properties but different names.
type OtelDistro struct {

	// a human-friendly name for this distribution, which can be displayed in the UI and documentation.
	// may include spaces and special characters.
	DisplayName string `yaml:"displayName"`

	// a unique name for this distribution, which helps to identify it.
	// should be a single word, lowercase, and may include hyphens (nodejs-community, dotnet-legacy-instrumentation).
	Name string `yaml:"name"`

	// the programming language this distribution targets.
	// each distribution must target a single language.
	Language common.ProgrammingLanguage `yaml:"language"`

	// List of distribution parameters that are required to be set by the user.
	// for example: libc type.
	RequireParameters []string `yaml:"requireParameters,omitempty"`

	// the runtime environments this distribution targets.
	// examples: nodejs, JVM, CPython, etc.
	// while java-script can run in both nodejs and browser, the distribution should specify where it is intended to run.
	RuntimeEnvironments []RuntimeEnvironment `yaml:"runtimeEnvironments"`

	// A list of frameworks this distribution targets (can be left empty)
	Frameworks []Framework `yaml:"frameworks"`

	// Free text description of the distribution, what it includes, it's use cases, etc.
	Description string `yaml:"description"`

	// Specifies the odigos tiers which includes this distribution
	Tiers []common.OdigosTier `yaml:"tiers"`

	// a list of environment variables that needs to be set in the application runtime
	// to enable the distribution.
	EnvironmentVariables []EnvironmentVariable `yaml:"environmentVariables,omitempty"`

	// Metadata and properties of the runtime agent that is used to enable the distribution.
	// Can be nil in case no runtime agent is required.
	RuntimeAgent *RuntimeAgent `yaml:"runtimeAgent,omitempty"`
}
