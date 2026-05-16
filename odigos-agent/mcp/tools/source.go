package tools

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	maxLogTailLines      = 2000
	defaultLogTailLines  = 200
	maxWorkloadPodsList  = 50
	maxContainerEnvItems = 100
)

// RegisterSourceTools wires the source/instrumentation MCP tools.
func RegisterSourceTools(server *mcpserver.MCPServer, clients *Clients, approvals *ApprovalCache) {
	manager := &sourceManager{clients: clients, approvals: approvals}
	manager.register(server)
}

type sourceManager struct {
	clients   *Clients
	approvals *ApprovalCache
}

func (m *sourceManager) register(server *mcpserver.MCPServer) {
	server.AddTool(mcp.NewTool(
		"get_source",
		mcp.WithDescription("Look up the Source CR(s) that apply to a workload: the workload-scope Source and the namespace-scope Source if present."),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Workload namespace.")),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Workload kind: Deployment, StatefulSet, DaemonSet, CronJob, Job, Rollout, DeploymentConfig.")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Workload name.")),
	), m.getSource)

	server.AddTool(mcp.NewTool(
		"get_instrumentation_config",
		mcp.WithDescription("Read the InstrumentationConfig CR for a workload. The CR's name follows the canonical <kind>-<name> lowercased pattern."),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Workload namespace.")),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Workload kind.")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Workload name.")),
	), m.getInstrumentationConfig)

	server.AddTool(mcp.NewTool(
		"list_instrumentation_instances",
		mcp.WithDescription("List per-pod InstrumentationInstance CRs for a workload. Each reports runtime instrumentation status for one SDK in one container."),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Workload namespace.")),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Workload kind.")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Workload name.")),
	), m.listInstrumentationInstances)

	server.AddTool(mcp.NewTool(
		"get_workload",
		mcp.WithDescription("Fetch a workload's pod template (containers, env, volumes, initContainers, resource requests) plus node selector and owner refs."),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Workload namespace.")),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Workload kind.")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Workload name.")),
	), m.getWorkload)

	server.AddTool(mcp.NewTool(
		"list_workload_pods",
		mcp.WithDescription("List pods belonging to a workload with restart counts and container statuses."),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Workload namespace.")),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Workload kind.")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Workload name.")),
	), m.listWorkloadPods)

	server.AddTool(mcp.NewTool(
		"get_pod_env",
		mcp.WithDescription("Return resolved env vars for one container in one pod. Literals are inlined; ConfigMap/Secret/Field refs are noted with source but Secret values are NEVER returned."),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Pod namespace.")),
		mcp.WithString("pod", mcp.Required(), mcp.Description("Pod name.")),
		mcp.WithString("container", mcp.Required(), mcp.Description("Container name within the pod.")),
	), m.getPodEnv)

	server.AddTool(mcp.NewTool(
		"get_odiglet_logs_for_node",
		mcp.WithDescription("Fetch recent logs from the odiglet DaemonSet pod running on the given node."),
		mcp.WithString("node", mcp.Required(), mcp.Description("Kubernetes node name.")),
		mcp.WithNumber("tail", mcp.Description("Number of trailing log lines (default 200, max 2000).")),
		mcp.WithNumber("since_seconds", mcp.Description("Only return log lines from the last N seconds. Optional.")),
	), m.getOdigletLogsForNode)

	server.AddTool(mcp.NewTool(
		"list_instrumentation_rules",
		mcp.WithDescription("List all InstrumentationRule CRs. Useful to detect rules that exclude or override a workload."),
	), m.listInstrumentationRules)

	server.AddTool(mcp.NewTool(
		"propose_create_source",
		mcp.WithDescription("Server-side dry-run create of a Source CR for a workload. Returns request_id (5-min TTL), YAML, diff, and rollback hint. Caller must then invoke apply_create_source with the request_id once a user approves."),
		mcp.WithString("namespace", mcp.Required(), mcp.Description("Workload namespace.")),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Workload kind: Deployment, StatefulSet, DaemonSet, CronJob, Job, Rollout, DeploymentConfig.")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Workload name.")),
	), m.proposeCreateSource)

	server.AddTool(mcp.NewTool(
		"apply_create_source",
		mcp.WithDescription("Apply a previously-proposed Source CR creation. The request_id must match a recent propose_create_source call and not be expired or already consumed."),
		mcp.WithString("request_id", mcp.Required(), mcp.Description("The request_id returned by propose_create_source.")),
	), m.applyCreateSource)
}

// ---- handlers ----

func (m *sourceManager) getSource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, err := request.RequireString("namespace")
	if err != nil {
		return ToolError("namespace required: %v", err)
	}
	kind, err := request.RequireString("kind")
	if err != nil {
		return ToolError("kind required: %v", err)
	}
	name, err := request.RequireString("name")
	if err != nil {
		return ToolError("name required: %v", err)
	}

	workloadSource, err := m.findWorkloadSource(ctx, namespace, kind, name)
	if err != nil {
		return ToolError("list sources in %s: %v", namespace, err)
	}
	namespaceSource, err := m.findNamespaceSource(ctx, namespace)
	if err != nil {
		return ToolError("list namespace-scope sources in %s: %v", namespace, err)
	}
	return WriteJSON(map[string]any{
		"workload":  workloadSource,
		"namespace": namespaceSource,
	})
}

func (m *sourceManager) getInstrumentationConfig(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, err := request.RequireString("namespace")
	if err != nil {
		return ToolError("namespace required: %v", err)
	}
	kind, err := request.RequireString("kind")
	if err != nil {
		return ToolError("kind required: %v", err)
	}
	name, err := request.RequireString("name")
	if err != nil {
		return ToolError("name required: %v", err)
	}

	configName := workload.CalculateWorkloadRuntimeObjectName(name, kind)
	cr, err := m.clients.Odigos.OdigosV1alpha1().InstrumentationConfigs(namespace).
		Get(ctx, configName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return WriteJSON(map[string]any{
				"found":         false,
				"expected_name": configName,
				"namespace":     namespace,
			})
		}
		return ToolError("get InstrumentationConfig %s/%s: %v", namespace, configName, err)
	}
	return WriteJSON(map[string]any{
		"found":         true,
		"expected_name": configName,
		"config":        cr,
	})
}

func (m *sourceManager) listInstrumentationInstances(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, err := request.RequireString("namespace")
	if err != nil {
		return ToolError("namespace required: %v", err)
	}
	kind, err := request.RequireString("kind")
	if err != nil {
		return ToolError("kind required: %v", err)
	}
	name, err := request.RequireString("name")
	if err != nil {
		return ToolError("name required: %v", err)
	}

	runtimeObject := workload.CalculateWorkloadRuntimeObjectName(name, kind)
	selector := fmt.Sprintf("%s=%s", consts.InstrumentedAppNameLabel, runtimeObject)
	list, err := m.clients.Odigos.OdigosV1alpha1().InstrumentationInstances(namespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return ToolError("list InstrumentationInstances in %s: %v", namespace, err)
	}
	return WriteJSON(map[string]any{
		"runtime_object_name": runtimeObject,
		"label_selector":      selector,
		"items":               list.Items,
	})
}

func (m *sourceManager) getWorkload(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, err := request.RequireString("namespace")
	if err != nil {
		return ToolError("namespace required: %v", err)
	}
	kind, err := request.RequireString("kind")
	if err != nil {
		return ToolError("kind required: %v", err)
	}
	name, err := request.RequireString("name")
	if err != nil {
		return ToolError("name required: %v", err)
	}

	template, selector, ownerRefs, err := m.fetchPodTemplate(ctx, namespace, kind, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return WriteJSON(map[string]any{
				"found": false,
				"kind":  kind,
				"name":  name,
			})
		}
		return ToolError("get %s %s/%s: %v", kind, namespace, name, err)
	}
	containers := summarizeContainers(template.Spec.Containers)
	initContainers := summarizeContainers(template.Spec.InitContainers)
	return WriteJSON(map[string]any{
		"found":            true,
		"kind":             kind,
		"name":             name,
		"namespace":        namespace,
		"owner_references": ownerRefs,
		"selector":         selector,
		"node_selector":    template.Spec.NodeSelector,
		"service_account":  template.Spec.ServiceAccountName,
		"containers":       containers,
		"init_containers":  initContainers,
		"volumes":          summarizeVolumes(template.Spec.Volumes),
	})
}

func (m *sourceManager) listWorkloadPods(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, err := request.RequireString("namespace")
	if err != nil {
		return ToolError("namespace required: %v", err)
	}
	kind, err := request.RequireString("kind")
	if err != nil {
		return ToolError("kind required: %v", err)
	}
	name, err := request.RequireString("name")
	if err != nil {
		return ToolError("name required: %v", err)
	}

	_, selector, _, err := m.fetchPodTemplate(ctx, namespace, kind, name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return WriteJSON(map[string]any{"found": false, "kind": kind, "name": name})
		}
		return ToolError("get %s %s/%s: %v", kind, namespace, name, err)
	}
	if selector == "" {
		return ToolError("workload %s %s/%s has no resolvable label selector", kind, namespace, name)
	}
	pods, err := m.clients.Core.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return ToolError("list pods for %s/%s: %v", namespace, name, err)
	}
	items := pods.Items
	truncated := false
	if len(items) > maxWorkloadPodsList {
		items = items[:maxWorkloadPodsList]
		truncated = true
	}
	return WriteJSON(map[string]any{
		"found":          true,
		"kind":           kind,
		"name":           name,
		"label_selector": selector,
		"truncated":      truncated,
		"pods":           summarizePods(items),
	})
}

func (m *sourceManager) getPodEnv(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, err := request.RequireString("namespace")
	if err != nil {
		return ToolError("namespace required: %v", err)
	}
	podName, err := request.RequireString("pod")
	if err != nil {
		return ToolError("pod required: %v", err)
	}
	containerName, err := request.RequireString("container")
	if err != nil {
		return ToolError("container required: %v", err)
	}

	pod, err := m.clients.Core.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return WriteJSON(map[string]any{"found": false, "pod": podName})
		}
		return ToolError("get pod %s/%s: %v", namespace, podName, err)
	}

	var container *corev1.Container
	for index := range pod.Spec.Containers {
		if pod.Spec.Containers[index].Name == containerName {
			container = &pod.Spec.Containers[index]
			break
		}
	}
	if container == nil {
		return WriteJSON(map[string]any{
			"found":     false,
			"pod":       podName,
			"container": containerName,
			"reason":    "container not present in pod spec",
		})
	}

	envEntries, more := describeEnv(container.Env)
	return WriteJSON(map[string]any{
		"found":          true,
		"pod":            podName,
		"container":      containerName,
		"env":            envEntries,
		"more_available": more,
		"env_from":       container.EnvFrom,
	})
}

func (m *sourceManager) getOdigletLogsForNode(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	node, err := request.RequireString("node")
	if err != nil {
		return ToolError("node required: %v", err)
	}
	tail := ClampInt(request.GetInt("tail", defaultLogTailLines), 1, maxLogTailLines)
	sinceSeconds := request.GetInt("since_seconds", 0)

	namespace := OdigosNamespace()
	pods, err := m.clients.Core.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", k8sconsts.OdigletAppLabelValue),
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", node),
	})
	if err != nil {
		return ToolError("list odiglet pods on node %s: %v", node, err)
	}
	if len(pods.Items) == 0 {
		return WriteJSON(map[string]any{
			"found": false,
			"node":  node,
			"reason": fmt.Sprintf("no pod with label app=%s on node %s in namespace %s",
				k8sconsts.OdigletAppLabelValue, node, namespace),
		})
	}
	odigletPod := pods.Items[0]

	logOptions := &corev1.PodLogOptions{
		Container:  k8sconsts.OdigletContainerName,
		TailLines:  ptrInt64(int64(tail)),
		Timestamps: true,
	}
	if sinceSeconds > 0 {
		logOptions.SinceSeconds = ptrInt64(int64(sinceSeconds))
	}
	raw, err := m.clients.Core.CoreV1().Pods(namespace).
		GetLogs(odigletPod.Name, logOptions).
		DoRaw(ctx)
	if err != nil {
		return ToolError("fetch logs for %s/%s: %v", namespace, odigletPod.Name, err)
	}
	return WriteJSON(map[string]any{
		"found":     true,
		"node":      node,
		"pod":       odigletPod.Name,
		"namespace": namespace,
		"container": k8sconsts.OdigletContainerName,
		"tail":      tail,
		"logs":      string(raw),
	})
}

func (m *sourceManager) listInstrumentationRules(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	list, err := m.clients.Odigos.OdigosV1alpha1().InstrumentationRules("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return ToolError("list InstrumentationRules: %v", err)
	}
	return WriteJSON(map[string]any{
		"items": list.Items,
		"count": len(list.Items),
	})
}

func (m *sourceManager) proposeCreateSource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace, err := request.RequireString("namespace")
	if err != nil {
		return ToolError("namespace required: %v", err)
	}
	kind, err := request.RequireString("kind")
	if err != nil {
		return ToolError("kind required: %v", err)
	}
	name, err := request.RequireString("name")
	if err != nil {
		return ToolError("name required: %v", err)
	}

	if !IsSupportedWorkloadKind(kind) {
		return ToolError("unsupported workload kind: %s", kind)
	}

	if _, _, _, err := m.fetchPodTemplate(ctx, namespace, kind, name); err != nil {
		if apierrors.IsNotFound(err) {
			return ToolError("workload %s %s/%s not found - refusing to create Source for a missing workload", kind, namespace, name)
		}
		return ToolError("validate workload %s %s/%s: %v", kind, namespace, name, err)
	}

	existing, err := m.findWorkloadSource(ctx, namespace, kind, name)
	if err != nil {
		return ToolError("look up existing Source: %v", err)
	}
	if existing != nil {
		return ToolError("Source already exists for %s/%s/%s (name=%s) - nothing to create", namespace, kind, name, existing.GetName())
	}

	desired := buildSourceForWorkload(namespace, kind, name)
	dryRun, err := m.clients.Odigos.OdigosV1alpha1().Sources(namespace).
		Create(ctx, desired, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
	if err != nil {
		return ToolError("server-side dry-run create Source for %s/%s/%s: %v", namespace, kind, name, err)
	}
	yamlBytes, err := yaml.Marshal(stripServerFields(dryRun))
	if err != nil {
		return ToolError("marshal dry-run Source to YAML: %v", err)
	}
	yamlText := string(yamlBytes)
	diff := prefixLines(yamlText, "+ ")
	rollback := fmt.Sprintf(
		"kubectl delete source -n %s -l %s=%s,%s=%s",
		namespace,
		k8sconsts.WorkloadNameLabel, name,
		k8sconsts.WorkloadKindLabel, kind,
	)

	requestID := m.approvals.Put(&PendingMutation{
		Operation:    "create_source",
		Namespace:    namespace,
		WorkloadKind: kind,
		WorkloadName: name,
		YAML:         yamlText,
		Diff:         diff,
		RollbackHint: rollback,
	})
	log.Printf("audit: op=propose_create_source ns=%s kind=%s name=%s request_id=%s",
		namespace, kind, name, requestID)

	return WriteJSON(map[string]any{
		"request_id":       requestID,
		"op":               "create_source",
		"namespace":        namespace,
		"kind":             kind,
		"name":             name,
		"yaml":             yamlText,
		"diff":             diff,
		"rollback_command": rollback,
		"expires_in":       int(defaultApprovalTTL.Seconds()),
	})
}

func (m *sourceManager) applyCreateSource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestID, err := request.RequireString("request_id")
	if err != nil {
		return ToolError("request_id required: %v", err)
	}

	pending := m.approvals.Take(requestID)
	if pending == nil {
		return ToolError("request_id %s not found, expired, or already consumed - call propose_create_source again", requestID)
	}
	if pending.Operation != "create_source" {
		return ToolError("request_id %s is for operation %q, not create_source", requestID, pending.Operation)
	}

	existing, err := m.findWorkloadSource(ctx, pending.Namespace, pending.WorkloadKind, pending.WorkloadName)
	if err != nil {
		return ToolError("recheck existing Source: %v", err)
	}
	if existing != nil {
		log.Printf("audit: op=apply_create_source request_id=%s result=skipped reason=already_exists name=%s",
			requestID, existing.GetName())
		return ToolError("Source %s already exists for %s/%s/%s - skipping apply",
			existing.GetName(), pending.Namespace, pending.WorkloadKind, pending.WorkloadName)
	}

	desired := buildSourceForWorkload(pending.Namespace, pending.WorkloadKind, pending.WorkloadName)
	created, err := m.clients.Odigos.OdigosV1alpha1().Sources(pending.Namespace).
		Create(ctx, desired, metav1.CreateOptions{})
	if err != nil {
		log.Printf("audit: op=apply_create_source request_id=%s result=failed err=%v", requestID, err)
		return ToolError("create Source for %s/%s/%s: %v",
			pending.Namespace, pending.WorkloadKind, pending.WorkloadName, err)
	}
	appliedYAML, marshalErr := yaml.Marshal(stripServerFields(created))
	if marshalErr != nil {
		appliedYAML = []byte(fmt.Sprintf("# yaml marshal failed: %v", marshalErr))
	}
	log.Printf("audit: op=apply_create_source request_id=%s result=created name=%s uid=%s",
		requestID, created.GetName(), created.GetUID())

	return WriteJSON(map[string]any{
		"applied":          true,
		"name":             created.GetName(),
		"namespace":        created.GetNamespace(),
		"uid":              string(created.GetUID()),
		"applied_yaml":     string(appliedYAML),
		"rollback_command": pending.RollbackHint,
	})
}

// ---- helpers ----

// IsSupportedWorkloadKind reports whether the agent's create_source mutation
// is willing to touch this workload kind. Matches frontend's EnsureSourceCRD.
func IsSupportedWorkloadKind(kind string) bool {
	switch k8sconsts.WorkloadKind(kind) {
	case k8sconsts.WorkloadKindDeployment,
		k8sconsts.WorkloadKindStatefulSet,
		k8sconsts.WorkloadKindDaemonSet,
		k8sconsts.WorkloadKindCronJob,
		k8sconsts.WorkloadKindJob,
		k8sconsts.WorkloadKindNamespace,
		k8sconsts.WorkloadKindDeploymentConfig,
		k8sconsts.WorkloadKindArgoRollout:
		return true
	}
	return false
}

// buildSourceForWorkload constructs the canonical Source CR for a workload.
// Matches frontend/services/sources.go:EnsureSourceCRD: GenerateName "source-"
// plus Spec.Workload. Labels are filled in by the odigos webhook.
func buildSourceForWorkload(namespace, kind, name string) *odigosv1.Source {
	return &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "source-",
			Namespace:    namespace,
		},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Namespace: namespace,
				Name:      name,
				Kind:      k8sconsts.WorkloadKind(kind),
			},
		},
	}
}

// findWorkloadSource returns the workload-scope Source for the given target,
// or nil if none exists. Robust against missing labels: lists all Sources in
// the namespace and matches by Spec.Workload (literal or regex).
func (m *sourceManager) findWorkloadSource(ctx context.Context, namespace, kind, name string) (*odigosv1.Source, error) {
	list, err := m.clients.Odigos.OdigosV1alpha1().Sources(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for index := range list.Items {
		source := &list.Items[index]
		if string(source.Spec.Workload.Kind) != kind {
			continue
		}
		if source.Spec.MatchWorkloadNameAsRegex {
			pattern, regexpErr := regexp.Compile(source.Spec.Workload.Name)
			if regexpErr == nil && pattern.MatchString(name) {
				return source, nil
			}
			continue
		}
		if source.Spec.Workload.Name == name {
			return source, nil
		}
	}
	return nil, nil
}

// findNamespaceSource returns the namespace-scope Source for the given
// namespace, or nil if none exists.
func (m *sourceManager) findNamespaceSource(ctx context.Context, namespace string) (*odigosv1.Source, error) {
	list, err := m.clients.Odigos.OdigosV1alpha1().Sources(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for index := range list.Items {
		source := &list.Items[index]
		if source.Spec.Workload.Kind == k8sconsts.WorkloadKindNamespace && source.Spec.Workload.Name == namespace {
			return source, nil
		}
	}
	return nil, nil
}

// fetchPodTemplate returns the pod template, label selector string, and owner
// references for a workload. The selector is k8s-list-friendly (e.g.
// "app=foo,role=bar"). Errors from the underlying k8s API bubble up unchanged
// so callers can spot apierrors.IsNotFound.
func (m *sourceManager) fetchPodTemplate(ctx context.Context, namespace, kind, name string) (*corev1.PodTemplateSpec, string, []metav1.OwnerReference, error) {
	apps := m.clients.Core.AppsV1()
	batch := m.clients.Core.BatchV1()
	switch k8sconsts.WorkloadKind(kind) {
	case k8sconsts.WorkloadKindDeployment:
		deployment, err := apps.Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, "", nil, err
		}
		return &deployment.Spec.Template, labelSelectorString(deployment.Spec.Selector), deployment.OwnerReferences, nil
	case k8sconsts.WorkloadKindStatefulSet:
		statefulSet, err := apps.StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, "", nil, err
		}
		return &statefulSet.Spec.Template, labelSelectorString(statefulSet.Spec.Selector), statefulSet.OwnerReferences, nil
	case k8sconsts.WorkloadKindDaemonSet:
		daemonSet, err := apps.DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, "", nil, err
		}
		return &daemonSet.Spec.Template, labelSelectorString(daemonSet.Spec.Selector), daemonSet.OwnerReferences, nil
	case k8sconsts.WorkloadKindCronJob:
		cronJob, err := batch.CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, "", nil, err
		}
		template := cronJob.Spec.JobTemplate.Spec.Template
		return &template, labelSelectorString(cronJob.Spec.JobTemplate.Spec.Selector), cronJob.OwnerReferences, nil
	case k8sconsts.WorkloadKindJob:
		job, err := batch.Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, "", nil, err
		}
		return &job.Spec.Template, labelSelectorString(job.Spec.Selector), job.OwnerReferences, nil
	}
	return nil, "", nil, errWorkloadKindUnsupported(kind)
}

func errWorkloadKindUnsupported(kind string) error {
	return fmt.Errorf("workload kind %q not supported for pod-template lookup (supports Deployment, StatefulSet, DaemonSet, CronJob, Job)", kind)
}

func labelSelectorString(selector *metav1.LabelSelector) string {
	if selector == nil {
		return ""
	}
	parts := make([]string, 0, len(selector.MatchLabels))
	for key, value := range selector.MatchLabels {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, ",")
}

func summarizeContainers(containers []corev1.Container) []map[string]any {
	result := make([]map[string]any, 0, len(containers))
	for _, container := range containers {
		envEntries, more := describeEnv(container.Env)
		result = append(result, map[string]any{
			"name":           container.Name,
			"image":          container.Image,
			"command":        container.Command,
			"args":           container.Args,
			"env":            envEntries,
			"more_env":       more,
			"env_from":       container.EnvFrom,
			"resources":      container.Resources,
			"volume_mounts":  container.VolumeMounts,
			"ports":          container.Ports,
			"working_dir":    container.WorkingDir,
		})
	}
	return result
}

func summarizeVolumes(volumes []corev1.Volume) []map[string]any {
	result := make([]map[string]any, 0, len(volumes))
	for _, volume := range volumes {
		kind := "unknown"
		switch {
		case volume.ConfigMap != nil:
			kind = "configMap"
		case volume.Secret != nil:
			kind = "secret"
		case volume.PersistentVolumeClaim != nil:
			kind = "persistentVolumeClaim"
		case volume.EmptyDir != nil:
			kind = "emptyDir"
		case volume.HostPath != nil:
			kind = "hostPath"
		case volume.Projected != nil:
			kind = "projected"
		}
		result = append(result, map[string]any{
			"name": volume.Name,
			"kind": kind,
		})
	}
	return result
}

func summarizePods(pods []corev1.Pod) []map[string]any {
	result := make([]map[string]any, 0, len(pods))
	for index := range pods {
		pod := &pods[index]
		containerStatuses := make([]map[string]any, 0, len(pod.Status.ContainerStatuses))
		for _, status := range pod.Status.ContainerStatuses {
			containerStatuses = append(containerStatuses, map[string]any{
				"name":                    status.Name,
				"image":                   status.Image,
				"ready":                   status.Ready,
				"restart_count":           status.RestartCount,
				"last_termination_reason": lastTerminationReason(status),
			})
		}
		result = append(result, map[string]any{
			"name":               pod.Name,
			"node":               pod.Spec.NodeName,
			"phase":              string(pod.Status.Phase),
			"pod_ip":             pod.Status.PodIP,
			"creation_timestamp": pod.CreationTimestamp.UTC().Format(time.RFC3339),
			"container_statuses": containerStatuses,
		})
	}
	return result
}

func lastTerminationReason(status corev1.ContainerStatus) string {
	if status.LastTerminationState.Terminated != nil {
		return status.LastTerminationState.Terminated.Reason
	}
	return ""
}

// describeEnv flattens container env, masking Secret refs and recording the
// source of every value. Returns (entries, moreAvailable).
func describeEnv(env []corev1.EnvVar) ([]map[string]any, bool) {
	limit := maxContainerEnvItems
	more := len(env) > limit
	if more {
		env = env[:limit]
	}
	entries := make([]map[string]any, 0, len(env))
	for _, envVar := range env {
		entry := map[string]any{"name": envVar.Name}
		switch {
		case envVar.ValueFrom == nil:
			entry["value"] = envVar.Value
			entry["source"] = "literal"
		case envVar.ValueFrom.SecretKeyRef != nil:
			ref := envVar.ValueFrom.SecretKeyRef
			entry["source"] = fmt.Sprintf("secret/%s/%s", ref.Name, ref.Key)
			// Deliberately omit value.
		case envVar.ValueFrom.ConfigMapKeyRef != nil:
			ref := envVar.ValueFrom.ConfigMapKeyRef
			entry["source"] = fmt.Sprintf("configmap/%s/%s", ref.Name, ref.Key)
		case envVar.ValueFrom.FieldRef != nil:
			entry["source"] = fmt.Sprintf("field/%s", envVar.ValueFrom.FieldRef.FieldPath)
		case envVar.ValueFrom.ResourceFieldRef != nil:
			ref := envVar.ValueFrom.ResourceFieldRef
			entry["source"] = fmt.Sprintf("resource/%s.%s", ref.ContainerName, ref.Resource)
		default:
			entry["source"] = "unknown"
		}
		entries = append(entries, entry)
	}
	return entries, more
}

// stripServerFields zeroes out server-populated metadata fields that aren't
// meaningful for a user-facing YAML preview (managed fields blob, resource
// version, etc.). Operates on a deep-ish copy: we only modify the metadata.
func stripServerFields(source *odigosv1.Source) *odigosv1.Source {
	if source == nil {
		return nil
	}
	clone := source.DeepCopy()
	clone.ManagedFields = nil
	clone.ResourceVersion = ""
	clone.SelfLink = ""
	clone.Generation = 0
	clone.CreationTimestamp = metav1.Time{}
	clone.UID = ""
	return clone
}

func prefixLines(text, prefix string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for index, line := range lines {
		lines[index] = prefix + line
	}
	return strings.Join(lines, "\n") + "\n"
}

func ptrInt64(value int64) *int64 { return &value }

// compile-time use checks so unused imports surface early.
var (
	_ = appsv1.Deployment{}
	_ = batchv1.Job{}
	_ = errors.New
)
