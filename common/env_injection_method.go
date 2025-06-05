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
