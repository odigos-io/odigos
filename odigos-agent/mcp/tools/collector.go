package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/odigos-io/odigos/api/k8sconsts"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	maxCollectorConfigBytes = 64 * 1024
	maxCollectorPods        = 50
	collectorMetricsPort    = 8888
	collectorMetricsPath    = "/metrics"
	metricsScrapeTimeout    = 5 * time.Second
	maxProcessorsListResult = 200
	maxActionsListResult    = 200
)

// otelcolInterestingMetrics is the curated set we surface back to the agent.
// Anything else from the collector's /metrics endpoint is ignored - the LLM
// doesn't need the full prometheus exposition dump.
var otelcolInterestingMetrics = []string{
	"otelcol_receiver_accepted_spans",
	"otelcol_receiver_refused_spans",
	"otelcol_receiver_accepted_metric_points",
	"otelcol_receiver_refused_metric_points",
	"otelcol_receiver_accepted_log_records",
	"otelcol_receiver_refused_log_records",
	"otelcol_processor_dropped_spans",
	"otelcol_processor_dropped_metric_points",
	"otelcol_processor_dropped_log_records",
	"otelcol_exporter_sent_spans",
	"otelcol_exporter_send_failed_spans",
	"otelcol_exporter_sent_metric_points",
	"otelcol_exporter_send_failed_metric_points",
	"otelcol_exporter_sent_log_records",
	"otelcol_exporter_send_failed_log_records",
}

// RegisterCollectorTools wires the collector MCP tools onto the server.
func RegisterCollectorTools(server *mcpserver.MCPServer, clients *Clients) {
	manager := &collectorManager{
		clients:   clients,
		namespace: OdigosNamespace(),
		http:      &http.Client{Timeout: metricsScrapeTimeout},
	}
	manager.register(server)
}

type collectorManager struct {
	clients   *Clients
	namespace string
	http      *http.Client
}

func (m *collectorManager) register(server *mcpserver.MCPServer) {
	server.AddTool(mcp.NewTool(
		"get_collectors_group",
		mcp.WithDescription("Fetch the CollectorsGroup CR (cluster gateway or per-node collector) including its spec and status."),
		mcp.WithString("role", mcp.Required(), mcp.Description("CLUSTER_GATEWAY or NODE_COLLECTOR.")),
	), m.getCollectorsGroup)

	server.AddTool(mcp.NewTool(
		"get_collector_config",
		mcp.WithDescription("Read the rendered ConfigMap for the gateway or node collector and return its parsed otelcol pipelines (receivers, processors, exporters, service.pipelines)."),
		mcp.WithString("role", mcp.Required(), mcp.Description("CLUSTER_GATEWAY or NODE_COLLECTOR.")),
	), m.getCollectorConfig)

	server.AddTool(mcp.NewTool(
		"list_collector_pods",
		mcp.WithDescription("List collector pods filtered by role: pod name, node, phase, container statuses, restart counts."),
		mcp.WithString("role", mcp.Required(), mcp.Description("CLUSTER_GATEWAY or NODE_COLLECTOR.")),
	), m.listCollectorPods)

	server.AddTool(mcp.NewTool(
		"get_collector_logs",
		mcp.WithDescription("Fetch logs from a collector pod. If pod is omitted, the first ready pod for the role is used."),
		mcp.WithString("role", mcp.Required(), mcp.Description("CLUSTER_GATEWAY or NODE_COLLECTOR.")),
		mcp.WithString("pod", mcp.Description("Specific pod name. Optional.")),
		mcp.WithNumber("tail", mcp.Description("Trailing log lines (default 200, max 2000).")),
		mcp.WithNumber("since_seconds", mcp.Description("Only return log lines from the last N seconds. Optional.")),
		mcp.WithString("grep", mcp.Description("Optional Go regex; only matching lines are returned.")),
	), m.getCollectorLogs)

	server.AddTool(mcp.NewTool(
		"get_collector_metrics",
		mcp.WithDescription("Scrape /metrics on a collector pod and return the curated otelcol counters (accepted, refused, dropped, sent, send_failed for spans/metric_points/log_records). HTTP scrape from the MCP pod - no exec."),
		mcp.WithString("role", mcp.Required(), mcp.Description("CLUSTER_GATEWAY or NODE_COLLECTOR.")),
		mcp.WithString("pod", mcp.Description("Specific pod name. Optional - defaults to the first ready pod.")),
	), m.getCollectorMetrics)

	server.AddTool(mcp.NewTool(
		"get_processors",
		mcp.WithDescription("List Processor CRs across all namespaces. Capped at 200 items."),
	), m.getProcessors)

	server.AddTool(mcp.NewTool(
		"get_actions",
		mcp.WithDescription("List Action CRs across all namespaces. Capped at 200 items."),
	), m.getActions)
}

// ---- handlers ----

func (m *collectorManager) getCollectorsGroup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	role, err := readCollectorRole(request)
	if err != nil {
		return ToolError("%v", err)
	}
	name, _ := collectorResourceNames(role)
	cr, err := m.clients.Odigos.OdigosV1alpha1().CollectorsGroups(m.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return WriteJSON(map[string]any{"found": false, "role": role, "expected_name": name})
		}
		return ToolError("get CollectorsGroup %s/%s: %v", m.namespace, name, err)
	}
	return WriteJSON(map[string]any{"found": true, "role": role, "collectors_group": cr})
}

func (m *collectorManager) getCollectorConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	role, err := readCollectorRole(request)
	if err != nil {
		return ToolError("%v", err)
	}
	configMapName, configKey := collectorConfigCoordinates(role)
	configMap, err := m.clients.Core.CoreV1().ConfigMaps(m.namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return WriteJSON(map[string]any{
				"found":          false,
				"role":           role,
				"configmap_name": configMapName,
				"namespace":      m.namespace,
			})
		}
		return ToolError("get ConfigMap %s/%s: %v", m.namespace, configMapName, err)
	}
	raw, ok := configMap.Data[configKey]
	if !ok {
		return WriteJSON(map[string]any{
			"found":          true,
			"role":           role,
			"configmap_name": configMapName,
			"missing_key":    configKey,
			"available_keys": keysOf(configMap.Data),
		})
	}
	parsed, parseErr := parseCollectorYAML(raw)
	displayRaw := raw
	truncated := false
	if len(displayRaw) > maxCollectorConfigBytes {
		displayRaw = displayRaw[:maxCollectorConfigBytes]
		truncated = true
	}
	result := map[string]any{
		"found":          true,
		"role":           role,
		"configmap_name": configMapName,
		"configmap_key":  configKey,
		"raw":            displayRaw,
		"truncated":      truncated,
		"parsed":         parsed,
	}
	if parseErr != nil {
		result["parse_error"] = parseErr.Error()
	}
	return WriteJSON(result)
}

func (m *collectorManager) listCollectorPods(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	role, err := readCollectorRole(request)
	if err != nil {
		return ToolError("%v", err)
	}
	pods, err := m.listPodsForRole(ctx, role)
	if err != nil {
		return ToolError("list collector pods for role %s: %v", role, err)
	}
	truncated := false
	if len(pods) > maxCollectorPods {
		pods = pods[:maxCollectorPods]
		truncated = true
	}
	return WriteJSON(map[string]any{
		"role":           role,
		"namespace":      m.namespace,
		"label_selector": collectorRoleLabelSelector(role),
		"truncated":      truncated,
		"pods":           summarizePods(pods),
	})
}

func (m *collectorManager) getCollectorLogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	role, err := readCollectorRole(request)
	if err != nil {
		return ToolError("%v", err)
	}
	tail := ClampInt(request.GetInt("tail", defaultLogTailLines), 1, maxLogTailLines)
	sinceSeconds := ClampInt(request.GetInt("since_seconds", 0), 0, 7*24*3600)
	podName := strings.TrimSpace(request.GetString("pod", ""))
	grep := strings.TrimSpace(request.GetString("grep", ""))

	var grepRegex *regexp.Regexp
	if grep != "" {
		compiled, regexErr := regexp.Compile(grep)
		if regexErr != nil {
			return ToolError("invalid grep regex %q: %v", grep, regexErr)
		}
		grepRegex = compiled
	}

	chosen, selectErr := m.pickCollectorPod(ctx, role, podName)
	if selectErr != nil {
		return ToolError("%v", selectErr)
	}
	if chosen == nil {
		return WriteJSON(map[string]any{
			"found":  false,
			"role":   role,
			"reason": "no collector pod found for role",
		})
	}

	options := &corev1.PodLogOptions{TailLines: ptrInt64(int64(tail)), Timestamps: true}
	if sinceSeconds > 0 {
		options.SinceSeconds = ptrInt64(int64(sinceSeconds))
	}
	raw, err := m.clients.Core.CoreV1().Pods(m.namespace).
		GetLogs(chosen.Name, options).DoRaw(ctx)
	if err != nil {
		return ToolError("get logs for %s/%s: %v", m.namespace, chosen.Name, err)
	}
	logs := string(raw)
	if grepRegex != nil {
		logs = filterLinesByRegex(logs, grepRegex)
	}
	return WriteJSON(map[string]any{
		"found":        true,
		"role":         role,
		"pod":          chosen.Name,
		"namespace":    m.namespace,
		"tail":         tail,
		"grep_applied": grepRegex != nil,
		"logs":         logs,
	})
}

func (m *collectorManager) getCollectorMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	role, err := readCollectorRole(request)
	if err != nil {
		return ToolError("%v", err)
	}
	podName := strings.TrimSpace(request.GetString("pod", ""))

	chosen, selectErr := m.pickCollectorPod(ctx, role, podName)
	if selectErr != nil {
		return ToolError("%v", selectErr)
	}
	if chosen == nil {
		return WriteJSON(map[string]any{"found": false, "role": role, "reason": "no collector pod found"})
	}
	// Metrics scrape needs a routable pod IP; non-ready pods often have none
	// and would otherwise produce a confusing "has no PodIP" error.
	if podName == "" && !isPodReady(chosen) {
		return WriteJSON(map[string]any{
			"found":  false,
			"role":   role,
			"pod":    chosen.Name,
			"reason": "no ready collector pod (scrape needs PodIP)",
		})
	}
	if chosen.Status.PodIP == "" {
		return ToolError("collector pod %s has no PodIP - is it still starting?", chosen.Name)
	}
	scrapeURL := fmt.Sprintf("http://%s:%d%s", chosen.Status.PodIP, collectorMetricsPort, collectorMetricsPath)

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, scrapeURL, nil)
	if err != nil {
		return ToolError("build scrape request: %v", err)
	}
	response, err := m.http.Do(httpRequest)
	if err != nil {
		return ToolError("scrape %s: %v", scrapeURL, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ToolError("scrape %s returned %d", scrapeURL, response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 8*1024*1024))
	if err != nil {
		return ToolError("read scrape body: %v", err)
	}
	metrics := ParsePromCountersForNames(string(body), otelcolInterestingMetrics)
	return WriteJSON(map[string]any{
		"found":      true,
		"role":       role,
		"pod":        chosen.Name,
		"pod_ip":     chosen.Status.PodIP,
		"scrape_url": scrapeURL,
		"metrics":    metrics,
	})
}

func (m *collectorManager) getProcessors(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	list, err := m.clients.Odigos.OdigosV1alpha1().Processors("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return ToolError("list Processors: %v", err)
	}
	items := list.Items
	truncated := false
	if len(items) > maxProcessorsListResult {
		items = items[:maxProcessorsListResult]
		truncated = true
	}
	return WriteJSON(map[string]any{
		"items":     items,
		"count":     len(items),
		"total":     len(list.Items),
		"truncated": truncated,
	})
}

func (m *collectorManager) getActions(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	list, err := m.clients.Odigos.OdigosV1alpha1().Actions("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return ToolError("list Actions: %v", err)
	}
	items := list.Items
	truncated := false
	if len(items) > maxActionsListResult {
		items = items[:maxActionsListResult]
		truncated = true
	}
	return WriteJSON(map[string]any{
		"items":     items,
		"count":     len(items),
		"total":     len(list.Items),
		"truncated": truncated,
	})
}

// ---- helpers ----

func readCollectorRole(request mcp.CallToolRequest) (string, error) {
	role, err := request.RequireString("role")
	if err != nil {
		return "", fmt.Errorf("role required: %w", err)
	}
	switch k8sconsts.CollectorRole(role) {
	case k8sconsts.CollectorsRoleClusterGateway, k8sconsts.CollectorsRoleNodeCollector:
		return role, nil
	}
	return "", fmt.Errorf("role must be CLUSTER_GATEWAY or NODE_COLLECTOR, got %q", role)
}

// collectorResourceNames returns the (CollectorsGroup name, workload name) for
// the role. Both happen to be the same string in v1, but isolating the lookup
// keeps the call sites readable.
func collectorResourceNames(role string) (groupName string, workloadName string) {
	if k8sconsts.CollectorRole(role) == k8sconsts.CollectorsRoleNodeCollector {
		return k8sconsts.OdigosNodeCollectorCollectorGroupName, k8sconsts.OdigosNodeCollectorDaemonSetName
	}
	return k8sconsts.OdigosClusterCollectorCollectorGroupName, k8sconsts.OdigosClusterCollectorDeploymentName
}

func collectorConfigCoordinates(role string) (configMapName string, configKey string) {
	if k8sconsts.CollectorRole(role) == k8sconsts.CollectorsRoleNodeCollector {
		return k8sconsts.OdigosNodeCollectorConfigMapName, k8sconsts.OdigosNodeCollectorConfigMapKey
	}
	return k8sconsts.OdigosClusterCollectorConfigMapName, k8sconsts.OdigosClusterCollectorConfigMapKey
}

func collectorRoleLabelSelector(role string) string {
	return fmt.Sprintf("%s=%s", k8sconsts.OdigosCollectorRoleLabel, role)
}

func (m *collectorManager) listPodsForRole(ctx context.Context, role string) ([]corev1.Pod, error) {
	pods, err := m.clients.Core.CoreV1().Pods(m.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: collectorRoleLabelSelector(role),
	})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func (m *collectorManager) pickCollectorPod(ctx context.Context, role, namedPod string) (*corev1.Pod, error) {
	pods, err := m.listPodsForRole(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("list collector pods: %w", err)
	}
	if namedPod != "" {
		for index := range pods {
			if pods[index].Name == namedPod {
				return &pods[index], nil
			}
		}
		return nil, fmt.Errorf("pod %s not found among role=%s pods in %s", namedPod, role, m.namespace)
	}
	for index := range pods {
		if isPodReady(&pods[index]) {
			return &pods[index], nil
		}
	}
	if len(pods) > 0 {
		return &pods[0], nil
	}
	return nil, nil
}

func isPodReady(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// parseCollectorYAML decodes an otelcol-style config into a generic map.
// On error, returns (nil, err); the caller surfaces the message under
// `parse_error` while keeping the raw text available.
func parseCollectorYAML(raw string) (map[string]any, error) {
	out := map[string]any{}
	if err := yaml.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func keysOf(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// filterLinesByRegex returns only the lines of text that match pattern,
// preserving order. Newlines are preserved between matching lines.
func filterLinesByRegex(text string, pattern *regexp.Regexp) string {
	lines := strings.Split(text, "\n")
	matched := make([]string, 0, len(lines))
	for _, line := range lines {
		if pattern.MatchString(line) {
			matched = append(matched, line)
		}
	}
	return strings.Join(matched, "\n")
}

// ParsePromCountersForNames parses a Prometheus text-exposition body and
// returns the *summed* value for each requested metric name. Labels are
// collapsed (sum across all label sets) so the agent gets one number per
// metric, which is what we want for "did spans flow?".
func ParsePromCountersForNames(body string, wanted []string) map[string]float64 {
	wantedSet := make(map[string]struct{}, len(wanted))
	for _, name := range wanted {
		wantedSet[name] = struct{}{}
	}
	results := map[string]float64{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := parsePromLine(line)
		if !ok {
			continue
		}
		if _, want := wantedSet[name]; !want {
			continue
		}
		results[name] += value
	}
	return results
}

// parsePromLine pulls (metric_name, value) out of a single non-comment
// exposition line of the form `name{labels} value` or `name value`. Returns
// ok=false for malformed lines.
func parsePromLine(line string) (string, float64, bool) {
	name := line
	if braceStart := strings.Index(line, "{"); braceStart != -1 {
		name = line[:braceStart]
		closingBrace := strings.Index(line, "}")
		if closingBrace == -1 || closingBrace < braceStart {
			return "", 0, false
		}
		line = line[closingBrace+1:]
	} else if firstSpace := strings.IndexAny(line, " \t"); firstSpace != -1 {
		name = line[:firstSpace]
		line = line[firstSpace:]
	} else {
		return "", 0, false
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", 0, false
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "", 0, false
	}
	return strings.TrimSpace(name), value, true
}
