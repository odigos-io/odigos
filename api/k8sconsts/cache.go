package k8sconsts

import "fmt"

const (
	// OdigosCacheServiceName is the ClusterIP Service for the shared cacheDb
	// (Redis). Deployed when a consuming feature such as URL templatization
	// live traffic learning is enabled.
	OdigosCacheServiceName = "odigos-cache"

	// OdigosCachePort is the cache listen port (Redis default).
	OdigosCachePort = 6379

	// UrlTemplatizationLiveTrafficLearningEnvVar enables the enterprise instrumentor
	// live-traffic learning runnable (path-example GC today; rule learning later).
	// Set by Helm when cardinalityControl.urlTemplatization.liveTrafficLearning.enabled
	// is set (enterprise on-prem token required). Presence of this env var registers
	// the job; the value is unused today (use "true").
	UrlTemplatizationLiveTrafficLearningEnvVar = "ODIGOS_URL_TEMPLATIZATION_LIVE_TRAFFIC_LEARNING"
)

// OdigosCacheEndpoint returns the in-cluster host:port for odigos-cache.
func OdigosCacheEndpoint(namespace string) string {
	return fmt.Sprintf("%s.%s:%d", OdigosCacheServiceName, namespace, OdigosCachePort)
}
