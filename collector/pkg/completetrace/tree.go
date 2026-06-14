package completetrace

import (
	"errors"
	"fmt"
	"sort"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

var ServiceInstanceRuntimeAttributeKeys = []string{
	string(semconv.TelemetrySDKLanguageKey),
	string(semconv.ProcessRuntimeNameKey),
	string(semconv.ProcessRuntimeVersionKey),
}

// WorkloadIdentityResolver derives workload identity from a span resource.
// When it returns false, service instance grouping falls back to service.name.
type WorkloadIdentityResolver func(resource pcommon.Resource) (cacheKey string, attrs pcommon.Map, ok bool)

// TraceTreeNode is a span in a complete trace with links to its parent and children.
// Span and Resource are pdata handles valid for the lifetime of the source ptrace.Traces.
type TraceTreeNode struct {
	Span        ptrace.Span
	Resource    pcommon.Resource
	Scope       pcommon.InstrumentationScope
	ServiceName string

	// the key for the service we are monitoring. for k8s it's namespace/deployment/name/container.
	// other platforms can use any resource attributes combination using the workload config extension.
	ServiceKey string

	// ServiceAttributes holds identifying resource attributes when a WorkloadIdentityResolver is used.
	ServiceAttributes pcommon.Map

	Parent   *TraceTreeNode
	Children []*TraceTreeNode
}

// ServiceInstance is a connected group of spans in the trace tree that share the same
// workload identity when available, otherwise service.name. A new instance starts at each
// tree root and whenever a child span belongs to a different workload than its parent.
type ServiceInstance struct {
	ServiceName string
	WorkloadKey string
	// ResourceAttributes holds identifying resource attributes for this instance's root span resource.
	ResourceAttributes pcommon.Map
	// Root is the topmost span of this service instance in the trace tree.
	Root *TraceTreeNode
	// Spans contains every span in this service instance, including Root.
	Spans []*TraceTreeNode
	// OutputLeaves contains spans that represent outputs from this service instance:
	// CLIENT/PRODUCER spans, or spans with a child in a different service.
	OutputLeaves []*TraceTreeNode
}

// TraceTree is the parent-linked span tree for a complete trace batch.
type TraceTree struct {
	// Nodes contains every span in the trace.
	Nodes []*TraceTreeNode
	// Leaves contains spans that have no children in the trace.
	Leaves []*TraceTreeNode
	// Roots contains spans whose parent is missing from the trace (including true roots).
	Roots []*TraceTreeNode
	// ServiceInstances contains connected same-workload span groups in the trace.
	ServiceInstances []*ServiceInstance
}

// BuildTraceTree indexes all spans in td, links each node to its parent, and returns the leaves.
// The input is expected to be a complete trace batch (e.g. from groupbytrace), but this function
// does not validate that; callers can use ValidateCompleteTrace first when needed.
func BuildTraceTree(td ptrace.Traces, resolver WorkloadIdentityResolver) (*TraceTree, error) {
	nodesBySpanID := make(map[pcommon.SpanID]*TraceTreeNode)

	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resourceSpan := td.ResourceSpans().At(i)
		resource := resourceSpan.Resource()
		serviceName := serviceNameFromResource(resource)
		serviceKey, serviceAttributes := resolveResourceIdentity(resource, resolver)

		for j := 0; j < resourceSpan.ScopeSpans().Len(); j++ {
			scopeSpan := resourceSpan.ScopeSpans().At(j)
			scope := scopeSpan.Scope()

			for k := 0; k < scopeSpan.Spans().Len(); k++ {
				span := scopeSpan.Spans().At(k)
				spanID := span.SpanID()
				if spanID.IsEmpty() {
					return nil, errors.New("span has empty span ID")
				}
				if _, exists := nodesBySpanID[spanID]; exists {
					return nil, fmt.Errorf("duplicate span ID in trace: %s", spanID.String())
				}

				node := &TraceTreeNode{
					Span:              span,
					Resource:          resource,
					Scope:             scope,
					ServiceName:       serviceName,
					ServiceKey:        serviceKey,
					ServiceAttributes: serviceAttributes,
				}
				nodesBySpanID[spanID] = node
			}
		}
	}

	tree := &TraceTree{
		Nodes: make([]*TraceTreeNode, 0, len(nodesBySpanID)),
	}

	for _, node := range nodesBySpanID {
		parentSpanID := node.Span.ParentSpanID()
		if parentSpanID.IsEmpty() {
			tree.Roots = append(tree.Roots, node)
			continue
		}

		parent, found := nodesBySpanID[parentSpanID]
		if !found {
			tree.Roots = append(tree.Roots, node)
			continue
		}
		node.Parent = parent
		parent.Children = append(parent.Children, node)
	}

	for _, node := range nodesBySpanID {
		if len(node.Children) == 0 {
			tree.Leaves = append(tree.Leaves, node)
		}
		tree.Nodes = append(tree.Nodes, node)
	}

	tree.ServiceInstances = buildServiceInstances(tree.Roots)

	return tree, nil
}

func buildServiceInstances(roots []*TraceTreeNode) []*ServiceInstance {
	instances := make([]*ServiceInstance, 0, len(roots))

	var collect func(node *TraceTreeNode, instance *ServiceInstance)
	collect = func(node *TraceTreeNode, instance *ServiceInstance) {
		instance.Spans = append(instance.Spans, node)
		for _, child := range node.Children {
			if belongsToServiceInstance(child, instance) {
				collect(child, instance)
				continue
			}

			childInstance := newServiceInstance(child)
			collect(child, childInstance)
			instances = append(instances, childInstance)
		}
	}

	for _, root := range roots {
		instance := newServiceInstance(root)
		collect(root, instance)
		instances = append(instances, instance)
	}

	sort.Slice(instances, func(i, j int) bool {
		left := instances[i].Root.Span.StartTimestamp()
		right := instances[j].Root.Span.StartTimestamp()
		if left != right {
			return left < right
		}
		if instances[i].WorkloadKey != instances[j].WorkloadKey {
			return instances[i].WorkloadKey < instances[j].WorkloadKey
		}
		return instances[i].Root.Span.SpanID().String() < instances[j].Root.Span.SpanID().String()
	})

	for _, instance := range instances {
		populateServiceInstanceOutputLeaves(instance)
	}

	return instances
}

func newServiceInstance(root *TraceTreeNode) *ServiceInstance {
	return &ServiceInstance{
		ServiceName:        root.ServiceName,
		WorkloadKey:        root.ServiceKey,
		ResourceAttributes: buildServiceInstanceResourceAttributes(root),
		Root:               root,
	}
}

func buildServiceInstanceResourceAttributes(root *TraceTreeNode) pcommon.Map {
	attrs := cloneAttributesMap(root.ServiceAttributes)
	resourceAttrs := root.Resource.Attributes()
	for _, key := range ServiceInstanceRuntimeAttributeKeys {
		copyStringResourceAttributeIfPresent(resourceAttrs, attrs, key)
	}
	return attrs
}

func copyStringResourceAttributeIfPresent(source, destination pcommon.Map, key string) {
	value, ok := source.Get(key)
	if !ok || value.Type() != pcommon.ValueTypeStr || value.Str() == "" {
		return
	}
	destination.PutStr(key, value.Str())
}

func belongsToServiceInstance(node *TraceTreeNode, instance *ServiceInstance) bool {
	if node.ServiceKey != "" && instance.WorkloadKey != "" {
		return node.ServiceKey == instance.WorkloadKey
	}
	return node.ServiceName == instance.ServiceName
}

func populateServiceInstanceOutputLeaves(instance *ServiceInstance) {
	for _, node := range instance.Spans {
		if isServiceInstanceOutputLeaf(node, instance) {
			instance.OutputLeaves = append(instance.OutputLeaves, node)
		}
	}
}

func isServiceInstanceOutputLeaf(node *TraceTreeNode, instance *ServiceInstance) bool {
	switch node.Span.Kind() {
	case ptrace.SpanKindClient, ptrace.SpanKindProducer:
		return true
	}

	for _, child := range node.Children {
		if !belongsToServiceInstance(child, instance) {
			return true
		}
	}

	return false
}

func resolveResourceIdentity(resource pcommon.Resource, resolver WorkloadIdentityResolver) (string, pcommon.Map) {
	if resolver == nil {
		return "", pcommon.NewMap()
	}
	cacheKey, attrs, ok := resolver(resource)
	if !ok {
		return "", pcommon.NewMap()
	}
	return cacheKey, attrs
}

func cloneAttributesMap(attrs pcommon.Map) pcommon.Map {
	if attrs.Len() == 0 {
		return pcommon.NewMap()
	}
	cloned := pcommon.NewMap()
	attrs.CopyTo(cloned)
	return cloned
}

func serviceNameFromResource(resource pcommon.Resource) string {
	v, ok := resource.Attributes().Get(string(semconv.ServiceNameKey))
	if !ok {
		return ""
	}
	return v.Str()
}
