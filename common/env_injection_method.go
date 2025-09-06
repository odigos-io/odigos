package common

// +kubebuilder:validation:Enum=loader;pod-manifest;loader-fallback-to-pod-manifest
type EnvInjectionMethod string

const (
	// EnvInjectionMethodLoader will try and add the LD_PRELOAD env var to the pod manifest
	// which will trigger the odigos loader. If LD_PRELOAD is already set, it will not be added and the pod won't be instrumented.
	LoaderEnvInjectionMethod EnvInjectionMethod = "loader"
	// EnvInjectionMethodPodManifest will add the runtime specific agent loading env vars (e.g PYTHONPATH, NODE_OPTIONS) to the pod manifest
	// taking into account the user defined values and appending if necessary.
	PodManifestEnvInjectionMethod EnvInjectionMethod = "pod-manifest"
	// EnvInjectionMethodLoaderFallbackToPodManifest will try and add the LD_PRELOAD env var to the pod manifest
	// which will trigger the odigos loader. If LD_PRELOAD is set with a user defined value,
	// it will fallback to adding the runtime specific agent loading env vars (e.g PYTHONPATH, NODE_OPTIONS) to the pod manifest
	// and taking into account the user defined values and appending if necessary.
	LoaderFallbackToPodManifestInjectionMethod EnvInjectionMethod = "loader-fallback-to-pod-manifest"
)

// +kubebuilder:validation:Enum=loader;pod-manifest
type EnvInjectionDecision string

// The decision on the actual injection method to use for a specific pod.
// While the configuration allows one to choose options with fallbaks,
// this decision is based on the runtime inspection, user overrides, and the distro support,
// and reflects what odigos is actually plan to use.
const (
	EnvInjectionDecisionLoader     =  EnvInjectionDecision(LoaderEnvInjectionMethod)
	EnvInjectionDecisionPodManifest = EnvInjectionDecision(PodManifestEnvInjectionMethod)
)
