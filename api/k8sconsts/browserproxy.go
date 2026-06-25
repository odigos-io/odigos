package k8sconsts

// Contract shared between the instrumentor pod webhook (which injects the sidecar) and the
// odigos-browser-proxy binary (which runs as the sidecar). These MUST stay in sync; the
// browser-proxy module imports these constants directly.

const (
	// BrowserProxyContainerName is the name of the sidecar container injected in front of a
	// browser-instrumented web server.
	BrowserProxyContainerName = "odigos-browser-proxy"

	// BrowserProxyInitContainerName is the name of the init container that sets up the iptables
	// rules redirecting the application's inbound traffic to the sidecar.
	BrowserProxyInitContainerName = "odigos-browser-proxy-init"

	// BrowserProxyRunAsUser is the UID the sidecar runs as. The iptables redirect excludes traffic
	// owned by this UID so the sidecar can reach the application on loopback without looping.
	BrowserProxyRunAsUser int64 = 1337

	// BrowserProxyListenPort is the port the sidecar listens on. Inbound traffic to the application
	// port is redirected here by the init container.
	BrowserProxyListenPort = 15001

	// Default location (under the agents directory) of the browser SDK bundle the sidecar serves.
	BrowserProxyDefaultAgentDir  = OdigosAgentsDirectory + "/browser"
	BrowserProxyDefaultAgentFile = "agent.js"

	// Same-origin URL path prefix the sidecar reserves for itself. The application is assumed not to
	// serve anything under this prefix.
	BrowserProxyPathPrefix = "/__odigos/"
	// Path (under the prefix) where the sidecar serves the browser SDK bundle.
	BrowserProxyAgentJsPath = "/__odigos/agent.js"
	// Path (under the prefix) where the sidecar receives OTLP/HTTP traces from the browser and
	// forwards them to the node-local collector.
	BrowserProxyTracesPath = "/__odigos/v1/traces"

	// Image name (without prefix/tag) of the browser-proxy sidecar.
	OdigosBrowserProxyImage = "odigos-browser-proxy"
	// Environment variable the instrumentor reads to override the browser-proxy image (set at install/upgrade).
	OdigosBrowserProxyEnvVarName = "ODIGOS_BROWSER_PROXY_IMAGE"
)

// Environment variables passed by the webhook to the sidecar / init container.
const (
	// Full http(s) URL of the application the sidecar forwards browser requests to (e.g. http://127.0.0.1:8080).
	BrowserProxyUpstreamEnvVar = "ODIGOS_BROWSER_PROXY_UPSTREAM"
	// Address the sidecar listens on (e.g. ":15001").
	BrowserProxyListenAddrEnvVar = "ODIGOS_BROWSER_PROXY_LISTEN_ADDR"
	// Directory the sidecar serves the browser SDK bundle from (e.g. /var/odigos/browser).
	BrowserProxyAgentDirEnvVar = "ODIGOS_BROWSER_PROXY_AGENT_DIR"
	// File name (within the agent dir) of the browser SDK bundle (e.g. agent.js).
	BrowserProxyAgentFileEnvVar = "ODIGOS_BROWSER_PROXY_AGENT_FILE"
	// Base OTLP/HTTP endpoint of the node-local collector the sidecar forwards browser telemetry to
	// (e.g. http://10.0.0.1:4318). The sidecar appends the OTLP signal path (e.g. /v1/traces).
	BrowserProxyOtlpHttpEndpointEnvVar = "ODIGOS_BROWSER_PROXY_OTLP_HTTP_ENDPOINT"
	// service.name reported by the browser SDK.
	BrowserProxyServiceNameEnvVar = "ODIGOS_BROWSER_PROXY_SERVICE_NAME"
	// OTEL_RESOURCE_ATTRIBUTES-style (key1=val1,key2=val2) resource attributes forwarded to the browser config.
	BrowserProxyResourceAttributesEnvVar = "ODIGOS_BROWSER_PROXY_RESOURCE_ATTRIBUTES"
	// Comma-separated list of URLs/regexes the browser SDK may attach trace-context headers to.
	BrowserProxyPropagateCorsUrlsEnvVar = "ODIGOS_BROWSER_PROXY_PROPAGATE_CORS_URLS"

	// Init-container-only: the application's inbound port that should be redirected to the sidecar.
	BrowserProxyAppPortEnvVar = "ODIGOS_BROWSER_PROXY_APP_PORT"
	// Init-container-only: the UID the sidecar runs as (excluded from the iptables redirect).
	BrowserProxyUidEnvVar = "ODIGOS_BROWSER_PROXY_UID"
)
