package tools

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/odigos-io/odigos/api/k8sconsts"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	destinationProbeTimeout      = 3 * time.Second
	maxDestinationListResults    = 200
	maxDestinationGatewayMatches = 20
	destinationErrorLogTail      = 500
)

// endpointKeyCandidates is the priority-ordered list of keys to look up in
// Destination.Spec.Data when guessing an endpoint URL/host. Odigos has many
// destination types; the keys overlap. First non-empty hit wins.
var endpointKeyCandidates = []string{
	"OTLP_GRPC_ENDPOINT",
	"OTLP_HTTP_ENDPOINT",
	"OTLP_ENDPOINT",
	"endpoint",
	"ENDPOINT",
	"url",
	"URL",
	"host",
	"HOST",
}

// destinationErrorLogPattern matches gateway log lines we treat as
// destination-related errors. Case-insensitive and broad on purpose - the
// destination name filter is the load-bearing constraint.
var destinationErrorLogPattern = regexp.MustCompile(`(?i)(exporter|destination|otlp).*?(error|failed|refused|429|401|403|timeout)`)

// RegisterDestinationTools wires the destination MCP tools onto the server.
func RegisterDestinationTools(server *mcpserver.MCPServer, clients *Clients) {
	manager := &destinationManager{
		clients:   clients,
		namespace: OdigosNamespace(),
	}
	manager.register(server)
}

type destinationManager struct {
	clients   *Clients
	namespace string
}

func (m *destinationManager) register(server *mcpserver.MCPServer) {
	server.AddTool(mcp.NewTool(
		"list_destinations",
		mcp.WithDescription("List Destination CRs across all namespaces. Capped at 200 items."),
	), m.listDestinations)

	server.AddTool(mcp.NewTool(
		"get_destination",
		mcp.WithDescription("Fetch a Destination CR. Includes spec.data and the name of spec.secretRef but never the secret values."),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Destination namespace (usually odigos-system).")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Destination CR name (this is `metadata.name`, distinct from `spec.destinationName`).")),
	), m.getDestination)

	server.AddTool(mcp.NewTool(
		"inspect_destination_secret",
		mcp.WithDescription("Return per-key length and `looks_empty` flag for the destination's referenced Secret. Raw values are NEVER returned."),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Destination namespace.")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Destination CR name.")),
	), m.inspectDestinationSecret)

	server.AddTool(mcp.NewTool(
		"get_destination_config_in_gateway",
		mcp.WithDescription("Find this destination's exporter block(s) in the gateway ConfigMap and the pipelines that reference them."),
		mcp.WithString("destination_name", mcp.Required(), mcp.Description("Destination CR name (matches the suffix odigos uses in exporter keys, e.g. `otlp/<name>`).")),
	), m.getDestinationConfigInGateway)

	server.AddTool(mcp.NewTool(
		"get_gateway_export_errors",
		mcp.WithDescription("Tail the gateway pod logs and return only lines that look like exporter/destination errors mentioning this destination name."),
		mcp.WithString("destination_name", mcp.Required(), mcp.Description("Destination CR name (used as a literal substring in the log filter).")),
		mcp.WithNumber("tail", mcp.Description("Trailing log lines to scan (default 500, max 2000).")),
	), m.getGatewayExportErrors)

	server.AddTool(mcp.NewTool(
		"probe_destination_endpoint",
		mcp.WithDescription("Resolve and TCP/TLS-probe the destination endpoint from the MCP pod. No auth headers sent. For TLS schemes the result distinguishes `tls_handshake_ok` (TCP+TLS protocol round-trip works) from `tls_verified` (server cert chain is trusted by the MCP container) so the agent can tell a self-signed enterprise destination from a genuinely unreachable one."),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Destination namespace.")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Destination CR name.")),
	), m.probeDestinationEndpoint)
}

// ---- handlers ----

func (m *destinationManager) listDestinations(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	list, err := m.clients.Odigos.OdigosV1alpha1().Destinations("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return ToolError("list Destinations: %v", err)
	}
	items := list.Items
	truncated := false
	if len(items) > maxDestinationListResults {
		items = items[:maxDestinationListResults]
		truncated = true
	}
	return WriteJSON(map[string]any{
		"items":     items,
		"count":     len(items),
		"total":     len(list.Items),
		"truncated": truncated,
	})
}

func (m *destinationManager) getDestination(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, err := request.RequireString("namespace")
	if err != nil {
		return ToolError("namespace required: %v", err)
	}
	name, err := request.RequireString("name")
	if err != nil {
		return ToolError("name required: %v", err)
	}
	destination, err := m.clients.Odigos.OdigosV1alpha1().Destinations(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return WriteJSON(map[string]any{"found": false, "namespace": namespace, "name": name})
		}
		return ToolError("get Destination %s/%s: %v", namespace, name, err)
	}
	return WriteJSON(map[string]any{"found": true, "destination": destination})
}

func (m *destinationManager) inspectDestinationSecret(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, err := request.RequireString("namespace")
	if err != nil {
		return ToolError("namespace required: %v", err)
	}
	name, err := request.RequireString("name")
	if err != nil {
		return ToolError("name required: %v", err)
	}
	destination, err := m.clients.Odigos.OdigosV1alpha1().Destinations(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return WriteJSON(map[string]any{"found": false, "name": name})
		}
		return ToolError("get Destination %s/%s: %v", namespace, name, err)
	}
	if destination.Spec.SecretRef == nil || destination.Spec.SecretRef.Name == "" {
		return WriteJSON(map[string]any{
			"found":          true,
			"has_secret_ref": false,
			"name":           name,
			"reason":         "Destination has no secretRef",
		})
	}
	secret, err := m.clients.Core.CoreV1().Secrets(namespace).Get(ctx, destination.Spec.SecretRef.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return WriteJSON(map[string]any{
				"found":          true,
				"has_secret_ref": true,
				"secret_missing": true,
				"secret_name":    destination.Spec.SecretRef.Name,
			})
		}
		return ToolError("get Secret %s/%s: %v", namespace, destination.Spec.SecretRef.Name, err)
	}
	keys := summarizeSecretKeys(secret.Data)
	return WriteJSON(map[string]any{
		"found":          true,
		"has_secret_ref": true,
		"secret_name":    secret.Name,
		"namespace":      secret.Namespace,
		"keys":           keys,
	})
}

func (m *destinationManager) getDestinationConfigInGateway(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	destinationName, err := request.RequireString("destination_name")
	if err != nil {
		return ToolError("destination_name required: %v", err)
	}
	configMapName := k8sconsts.OdigosClusterCollectorConfigMapName
	configKey := k8sconsts.OdigosClusterCollectorConfigMapKey
	configMap, err := m.clients.Core.CoreV1().ConfigMaps(m.namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return WriteJSON(map[string]any{
				"found":          false,
				"configmap_name": configMapName,
				"reason":         "gateway ConfigMap missing",
			})
		}
		return ToolError("get ConfigMap %s/%s: %v", m.namespace, configMapName, err)
	}
	raw, ok := configMap.Data[configKey]
	if !ok {
		return WriteJSON(map[string]any{
			"found":          true,
			"configmap_name": configMapName,
			"missing_key":    configKey,
			"available_keys": keysOf(configMap.Data),
		})
	}
	parsed, parseErr := parseCollectorYAML(raw)
	if parseErr != nil {
		return ToolError("parse gateway config: %v", parseErr)
	}
	exporters := findExportersForDestination(parsed, destinationName)
	pipelineRefs := findPipelinesUsingExporters(parsed, keysOfAnyMap(exporters))
	return WriteJSON(map[string]any{
		"destination_name": destinationName,
		"configmap_name":   configMapName,
		"matched":          len(exporters) > 0,
		"exporters":        exporters,
		"pipelines":        pipelineRefs,
	})
}

func (m *destinationManager) getGatewayExportErrors(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	destinationName, err := request.RequireString("destination_name")
	if err != nil {
		return ToolError("destination_name required: %v", err)
	}
	tail := ClampInt(request.GetInt("tail", destinationErrorLogTail), 1, maxLogTailLines)

	gatewayPods, err := m.clients.Core.CoreV1().Pods(m.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: collectorRoleLabelSelector(string(k8sconsts.CollectorsRoleClusterGateway)),
	})
	if err != nil {
		return ToolError("list gateway pods: %v", err)
	}
	if len(gatewayPods.Items) == 0 {
		return WriteJSON(map[string]any{"found": false, "reason": "no gateway pods"})
	}
	var chosen *corev1.Pod
	for index := range gatewayPods.Items {
		if isPodReady(&gatewayPods.Items[index]) {
			chosen = &gatewayPods.Items[index]
			break
		}
	}
	if chosen == nil {
		chosen = &gatewayPods.Items[0]
	}

	raw, err := m.clients.Core.CoreV1().Pods(m.namespace).
		GetLogs(chosen.Name, &corev1.PodLogOptions{TailLines: ptrInt64(int64(tail)), Timestamps: true}).DoRaw(ctx)
	if err != nil {
		return ToolError("get logs for %s/%s: %v", m.namespace, chosen.Name, err)
	}
	filtered := filterExportErrorLines(string(raw), destinationName)
	return WriteJSON(map[string]any{
		"found":            true,
		"destination_name": destinationName,
		"pod":              chosen.Name,
		"namespace":        m.namespace,
		"tail":             tail,
		"errors":           filtered,
		"error_count":      countLines(filtered),
	})
}

func (m *destinationManager) probeDestinationEndpoint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, err := request.RequireString("namespace")
	if err != nil {
		return ToolError("namespace required: %v", err)
	}
	name, err := request.RequireString("name")
	if err != nil {
		return ToolError("name required: %v", err)
	}
	destination, err := m.clients.Odigos.OdigosV1alpha1().Destinations(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return WriteJSON(map[string]any{"found": false, "name": name})
		}
		return ToolError("get Destination %s/%s: %v", namespace, name, err)
	}
	endpoint, sourceKey, found := pickDestinationEndpoint(destination.Spec.Data)
	if !found {
		return WriteJSON(map[string]any{
			"found":           true,
			"probed":          false,
			"reason":          "no recognized endpoint key in spec.data",
			"considered_keys": endpointKeyCandidates,
			"data_keys":       sortedKeys(destination.Spec.Data),
		})
	}
	probeResult := probeTCPAndTLS(ctx, endpoint)
	probeResult["destination"] = name
	probeResult["source_key"] = sourceKey
	probeResult["endpoint"] = endpoint
	probeResult["found"] = true
	probeResult["probed"] = true
	return WriteJSON(probeResult)
}

// ---- helpers ----

// summarizeSecretKeys returns per-key length and looks_empty without leaking
// values. Keys are returned in sorted order so the output is deterministic.
func summarizeSecretKeys(data map[string][]byte) []map[string]any {
	names := make([]string, 0, len(data))
	for key := range data {
		names = append(names, key)
	}
	sort.Strings(names)
	result := make([]map[string]any, 0, len(names))
	for _, key := range names {
		value := data[key]
		trimmed := strings.TrimSpace(string(value))
		result = append(result, map[string]any{
			"key":         key,
			"length":      len(value),
			"looks_empty": trimmed == "",
		})
	}
	return result
}

func pickDestinationEndpoint(data map[string]string) (string, string, bool) {
	for _, key := range endpointKeyCandidates {
		if value, ok := data[key]; ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), key, true
		}
	}
	return "", "", false
}

// probeTCPAndTLS performs a tcp dial and, if the URL scheme is https/grpcs, a
// TLS handshake against the resolved endpoint. Returns a map ready to merge
// into the tool result. No auth headers are sent.
func probeTCPAndTLS(ctx context.Context, endpoint string) map[string]any {
	parsed, err := normalizeEndpointURL(endpoint)
	if err != nil {
		return map[string]any{
			"parse_error": err.Error(),
			"tcp_ok":      false,
		}
	}
	host := parsed.Hostname()
	port := parsed.Port()
	scheme := parsed.Scheme
	if port == "" {
		port = defaultPortForScheme(scheme)
	}
	address := net.JoinHostPort(host, port)

	dialer := &net.Dialer{Timeout: destinationProbeTimeout}
	start := time.Now()
	connection, dialErr := dialer.DialContext(ctx, "tcp", address)
	tcpLatency := time.Since(start)
	if dialErr != nil {
		return map[string]any{
			"host":       host,
			"port":       port,
			"scheme":     scheme,
			"address":    address,
			"tcp_ok":     false,
			"error":      dialErr.Error(),
			"elapsed_ms": tcpLatency.Milliseconds(),
		}
	}
	defer connection.Close()

	result := map[string]any{
		"host":       host,
		"port":       port,
		"scheme":     scheme,
		"address":    address,
		"tcp_ok":     true,
		"elapsed_ms": tcpLatency.Milliseconds(),
	}
	if !schemeUsesTLS(scheme) {
		return result
	}

	// TLS verification can fail two ways the LLM must distinguish:
	//   (a) handshake itself fails - destination is not actually a TLS server
	//       on this port, or the server presents a malformed/expired cert.
	//   (b) handshake succeeds but cert chain is untrusted - common with
	//       self-signed or private-CA destinations in enterprise setups.
	// We do a strict handshake first; on failure that looks like a verify
	// error we retry with InsecureSkipVerify so we can report `tls_handshake_ok`
	// independently of `tls_verified`. The TCP conn from above is consumed by
	// the first attempt, so the retry dials again with a fresh conn.
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(destinationProbeTimeout)
	}
	if err := connection.SetDeadline(deadline); err != nil {
		result["tls_handshake_ok"] = false
		result["tls_verified"] = false
		result["tls_error"] = err.Error()
		return result
	}

	tlsConfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	tlsClient := tls.Client(connection, tlsConfig)
	tlsStart := time.Now()
	strictErr := tlsClient.HandshakeContext(ctx)
	tlsElapsed := time.Since(tlsStart).Milliseconds()

	if strictErr == nil {
		state := tlsClient.ConnectionState()
		result["tls_handshake_ok"] = true
		result["tls_verified"] = true
		result["tls_ok"] = true // backwards-compat field
		result["tls_elapsed_ms"] = tlsElapsed
		result["tls_version"] = tlsVersionString(state.Version)
		result["tls_cipher_suite"] = tls.CipherSuiteName(state.CipherSuite)
		result["tls_certs"] = summarizeCertificates(state.PeerCertificates)
		_ = tlsClient.Close()
		return result
	}
	_ = tlsClient.Close()

	// Retry insecure to distinguish trust-store mismatch from real handshake
	// failure. New TCP conn since the previous one is now closed.
	retryConn, retryDialErr := dialer.DialContext(ctx, "tcp", address)
	if retryDialErr != nil {
		result["tls_handshake_ok"] = false
		result["tls_verified"] = false
		result["tls_error"] = strictErr.Error()
		result["tls_elapsed_ms"] = tlsElapsed
		return result
	}
	defer retryConn.Close()
	if err := retryConn.SetDeadline(deadline); err != nil {
		result["tls_handshake_ok"] = false
		result["tls_verified"] = false
		result["tls_error"] = strictErr.Error()
		result["tls_elapsed_ms"] = tlsElapsed
		return result
	}
	insecureConfig := &tls.Config{
		ServerName:         host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, //nolint:gosec // diagnostic probe only, never exchanges payload
	}
	insecureClient := tls.Client(retryConn, insecureConfig)
	insecureStart := time.Now()
	if err := insecureClient.HandshakeContext(ctx); err != nil {
		result["tls_handshake_ok"] = false
		result["tls_verified"] = false
		result["tls_error"] = strictErr.Error()
		result["tls_insecure_error"] = err.Error()
		result["tls_elapsed_ms"] = tlsElapsed + time.Since(insecureStart).Milliseconds()
		return result
	}
	defer insecureClient.Close()
	state := insecureClient.ConnectionState()
	result["tls_handshake_ok"] = true
	result["tls_verified"] = false
	result["tls_ok"] = false
	result["tls_verify_error"] = strictErr.Error()
	result["tls_elapsed_ms"] = tlsElapsed + time.Since(insecureStart).Milliseconds()
	result["tls_version"] = tlsVersionString(state.Version)
	result["tls_cipher_suite"] = tls.CipherSuiteName(state.CipherSuite)
	result["tls_certs"] = summarizeCertificates(state.PeerCertificates)
	return result
}

// normalizeEndpointURL parses an endpoint string into a URL. Accepts
//   - explicit URLs: https://foo:443, grpcs://bar:4317, http://baz:4318
//   - bare host:port: api.example.com:4317 (assumed grpc / unknown scheme)
//   - bare hostname: api.example.com (assumed grpc, no port - caller fills
//     in 4317)
func normalizeEndpointURL(endpoint string) (*url.URL, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("empty endpoint")
	}
	if !strings.Contains(endpoint, "://") {
		// host:port or host - prepend a sentinel scheme so url.Parse works.
		return &url.URL{Host: endpoint, Scheme: "tcp"}, nil
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("endpoint missing host: %s", endpoint)
	}
	return parsed, nil
}

func defaultPortForScheme(scheme string) string {
	switch strings.ToLower(scheme) {
	case "https", "grpcs":
		return "443"
	case "http":
		return "80"
	}
	return "4317"
}

func schemeUsesTLS(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "https", "grpcs", "tls":
		return true
	}
	return false
}

func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	}
	return fmt.Sprintf("0x%04x", version)
}

func summarizeCertificates(certificates []*x509.Certificate) []map[string]any {
	result := make([]map[string]any, 0, len(certificates))
	for _, certificate := range certificates {
		result = append(result, map[string]any{
			"subject":   certificate.Subject.String(),
			"issuer":    certificate.Issuer.String(),
			"not_after": certificate.NotAfter.UTC().Format(time.RFC3339),
			"dns_names": certificate.DNSNames,
		})
	}
	return result
}

// findExportersForDestination scans the otelcol exporters map for keys that
// reference the destination name. Odigos names exporters like
// `<type>/<destinationName>` (e.g. `otlp/datadog`, `awsxray/aws-traces`), so
// we require the destination name to be either the full key or preceded by a
// slash to avoid false positives for short or generic names like "aws" or
// "gcp".
func findExportersForDestination(parsed map[string]any, destinationName string) map[string]any {
	exporters, ok := parsed["exporters"].(map[string]any)
	if !ok || destinationName == "" {
		return map[string]any{}
	}
	matches := map[string]any{}
	count := 0
	for key, value := range exporters {
		if !exporterKeyMatchesDestination(key, destinationName) {
			continue
		}
		matches[key] = value
		count++
		if count >= maxDestinationGatewayMatches {
			break
		}
	}
	return matches
}

// exporterKeyMatchesDestination checks whether an exporter key refers to the
// given destination. Accepts both exact key match (rare - some destinations
// produce just `<type>` with no suffix) and the canonical `<type>/<name>`
// form, where <name> is the destination CR name.
func exporterKeyMatchesDestination(exporterKey, destinationName string) bool {
	if exporterKey == destinationName {
		return true
	}
	suffix := "/" + destinationName
	return strings.HasSuffix(exporterKey, suffix)
}

// findPipelinesUsingExporters walks service.pipelines.<name>.exporters and
// returns the pipelines that reference any of the matched exporter keys.
func findPipelinesUsingExporters(parsed map[string]any, matchedExporters []string) []map[string]any {
	service, ok := parsed["service"].(map[string]any)
	if !ok {
		return nil
	}
	pipelines, ok := service["pipelines"].(map[string]any)
	if !ok {
		return nil
	}
	matchSet := make(map[string]struct{}, len(matchedExporters))
	for _, name := range matchedExporters {
		matchSet[name] = struct{}{}
	}
	result := make([]map[string]any, 0)
	for pipelineName, raw := range pipelines {
		pipeline, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		exporterList, ok := pipeline["exporters"].([]any)
		if !ok {
			continue
		}
		hits := make([]string, 0, len(exporterList))
		for _, entry := range exporterList {
			if name, ok := entry.(string); ok {
				if _, matched := matchSet[name]; matched {
					hits = append(hits, name)
				}
			}
		}
		if len(hits) > 0 {
			result = append(result, map[string]any{
				"pipeline":          pipelineName,
				"matched_exporters": hits,
			})
		}
	}
	return result
}

func keysOfAnyMap(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// filterExportErrorLines keeps only lines mentioning destinationName AND
// matching the exporter-error pattern.
func filterExportErrorLines(text, destinationName string) string {
	if destinationName == "" {
		return ""
	}
	lower := strings.ToLower(destinationName)
	matched := make([]string, 0)
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(strings.ToLower(line), lower) {
			continue
		}
		if !destinationErrorLogPattern.MatchString(line) {
			continue
		}
		matched = append(matched, line)
	}
	return strings.Join(matched, "\n")
}

func countLines(text string) int {
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}
