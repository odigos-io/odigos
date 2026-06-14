package serviceioconnector

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestBuildTraceTree_Empty(t *testing.T) {
	tree, err := BuildTraceTree(ptrace.NewTraces(), nil)
	require.NoError(t, err)
	require.Empty(t, tree.Nodes)
	require.Empty(t, tree.Leaves)
	require.Empty(t, tree.Roots)
	require.Empty(t, tree.ServiceInstances)
}

func TestBuildTraceTree_SingleSpan(t *testing.T) {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "svc-a")
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetSpanID(pcommon.SpanID([8]byte{1}))
	span.SetName("root")

	tree, err := BuildTraceTree(td, nil)
	require.NoError(t, err)
	require.Len(t, tree.Nodes, 1)
	require.Len(t, tree.Leaves, 1)
	require.Len(t, tree.Roots, 1)
	require.Nil(t, tree.Leaves[0].Parent)
	require.Empty(t, tree.Leaves[0].Children)
	require.Equal(t, "root", tree.Leaves[0].Span.Name())
	require.Equal(t, "svc-a", tree.Leaves[0].ServiceName)
	require.Len(t, tree.ServiceInstances, 1)
	require.Equal(t, "svc-a", tree.ServiceInstances[0].ServiceName)
	require.Len(t, tree.ServiceInstances[0].Spans, 1)
}

func TestBuildTraceTree_Chain(t *testing.T) {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()

	rootID := pcommon.SpanID([8]byte{1})
	childID := pcommon.SpanID([8]byte{2})
	leafID := pcommon.SpanID([8]byte{3})

	root := ss.Spans().AppendEmpty()
	root.SetSpanID(rootID)
	root.SetName("root")

	child := ss.Spans().AppendEmpty()
	child.SetSpanID(childID)
	child.SetParentSpanID(rootID)
	child.SetName("child")

	leaf := ss.Spans().AppendEmpty()
	leaf.SetSpanID(leafID)
	leaf.SetParentSpanID(childID)
	leaf.SetName("leaf")

	tree, err := BuildTraceTree(td, nil)
	require.NoError(t, err)
	require.Len(t, tree.Nodes, 3)
	require.Len(t, tree.Leaves, 1)
	require.Len(t, tree.Roots, 1)

	require.Equal(t, "leaf", tree.Leaves[0].Span.Name())
	require.Equal(t, "child", tree.Leaves[0].Parent.Span.Name())
	require.Equal(t, "root", tree.Leaves[0].Parent.Parent.Span.Name())
	require.Nil(t, tree.Leaves[0].Parent.Parent.Parent)

	rootNode := tree.Roots[0]
	require.Len(t, rootNode.Children, 1)
	require.Equal(t, "child", rootNode.Children[0].Span.Name())
	require.Len(t, rootNode.Children[0].Children, 1)
	require.Equal(t, "leaf", rootNode.Children[0].Children[0].Span.Name())
	require.Empty(t, rootNode.Children[0].Children[0].Children)
}

func TestBuildTraceTree_MultipleLeaves(t *testing.T) {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()

	rootID := pcommon.SpanID([8]byte{1})
	leftID := pcommon.SpanID([8]byte{2})
	rightID := pcommon.SpanID([8]byte{3})

	root := ss.Spans().AppendEmpty()
	root.SetSpanID(rootID)

	left := ss.Spans().AppendEmpty()
	left.SetSpanID(leftID)
	left.SetParentSpanID(rootID)
	left.SetName("left")

	right := ss.Spans().AppendEmpty()
	right.SetSpanID(rightID)
	right.SetParentSpanID(rootID)
	right.SetName("right")

	tree, err := BuildTraceTree(td, nil)
	require.NoError(t, err)
	require.Len(t, tree.Leaves, 2)
	require.Equal(t, rootID, tree.Leaves[0].Parent.Span.SpanID())
	require.Equal(t, rootID, tree.Leaves[1].Parent.Span.SpanID())
	require.Len(t, tree.Roots[0].Children, 2)
}

func TestBuildTraceTree_OrphanParent(t *testing.T) {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()

	span := ss.Spans().AppendEmpty()
	span.SetSpanID(pcommon.SpanID([8]byte{2}))
	span.SetParentSpanID(pcommon.SpanID([8]byte{99}))

	tree, err := BuildTraceTree(td, nil)
	require.NoError(t, err)
	require.Len(t, tree.Roots, 1)
	require.Nil(t, tree.Roots[0].Parent)
	require.Len(t, tree.Leaves, 1)
}

func TestBuildTraceTree_DuplicateSpanID(t *testing.T) {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()

	spanID := pcommon.SpanID([8]byte{1})
	for range 2 {
		span := ss.Spans().AppendEmpty()
		span.SetSpanID(spanID)
	}

	_, err := BuildTraceTree(td, nil)
	require.Error(t, err)
}

func TestBuildTraceTree_ServiceInstances(t *testing.T) {
	td := ptrace.NewTraces()

	rootID := pcommon.SpanID([8]byte{1})
	client1ID := pcommon.SpanID([8]byte{2})
	server1ID := pcommon.SpanID([8]byte{3})
	internal1ID := pcommon.SpanID([8]byte{4})
	client2ID := pcommon.SpanID([8]byte{5})
	server2ID := pcommon.SpanID([8]byte{6})
	internal2ID := pcommon.SpanID([8]byte{7})

	appendSpan := func(serviceName, name string, spanID, parentID pcommon.SpanID, kind ptrace.SpanKind) {
		rs := td.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr("service.name", serviceName)
		ss := rs.ScopeSpans().AppendEmpty()
		span := ss.Spans().AppendEmpty()
		span.SetSpanID(spanID)
		if !parentID.IsEmpty() {
			span.SetParentSpanID(parentID)
		}
		span.SetName(name)
		if kind != ptrace.SpanKindUnspecified {
			span.SetKind(kind)
		}
	}

	appendSpan("svc-1", "root", rootID, pcommon.SpanID{}, ptrace.SpanKindServer)
	appendSpan("svc-1", "client-1", client1ID, rootID, ptrace.SpanKindClient)
	appendSpan("svc-2", "server-1", server1ID, client1ID, ptrace.SpanKindServer)
	appendSpan("svc-2", "internal-1", internal1ID, server1ID, ptrace.SpanKindInternal)
	appendSpan("svc-1", "client-2", client2ID, rootID, ptrace.SpanKindClient)
	appendSpan("svc-2", "server-2", server2ID, client2ID, ptrace.SpanKindServer)
	appendSpan("svc-2", "internal-2", internal2ID, server2ID, ptrace.SpanKindInternal)

	tree, err := BuildTraceTree(td, nil)
	require.NoError(t, err)

	require.Len(t, tree.ServiceInstances, 3)

	svc1Instances := filterInstancesByService(tree.ServiceInstances, "svc-1")
	require.Len(t, svc1Instances, 1)
	require.Len(t, svc1Instances[0].Spans, 3)
	require.Equal(t, "root", svc1Instances[0].Root.Span.Name())
	require.ElementsMatch(t, []string{"root", "client-1", "client-2"}, spanNames(svc1Instances[0].Spans))
	require.ElementsMatch(t, []string{"client-1", "client-2"}, spanNames(svc1Instances[0].OutputLeaves))

	svc2Instances := filterInstancesByService(tree.ServiceInstances, "svc-2")
	require.Len(t, svc2Instances, 2)
	require.Len(t, svc2Instances[0].Spans, 2)
	require.Len(t, svc2Instances[1].Spans, 2)
	require.Equal(t, "server-1", svc2Instances[0].Root.Span.Name())
	require.Equal(t, "server-2", svc2Instances[1].Root.Span.Name())
	require.ElementsMatch(t, []string{"server-1", "internal-1"}, spanNames(svc2Instances[0].Spans))
	require.ElementsMatch(t, []string{"server-2", "internal-2"}, spanNames(svc2Instances[1].Spans))
	require.Empty(t, svc2Instances[0].OutputLeaves)
	require.Empty(t, svc2Instances[1].OutputLeaves)
}

func TestBuildTraceTree_ServiceInstances_ReentrantService(t *testing.T) {
	td := ptrace.NewTraces()

	rootID := pcommon.SpanID([8]byte{1})
	clientID := pcommon.SpanID([8]byte{2})
	serverID := pcommon.SpanID([8]byte{3})
	callbackID := pcommon.SpanID([8]byte{4})

	appendSpan := func(serviceName, name string, spanID, parentID pcommon.SpanID, kind ptrace.SpanKind) {
		rs := td.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr("service.name", serviceName)
		ss := rs.ScopeSpans().AppendEmpty()
		span := ss.Spans().AppendEmpty()
		span.SetSpanID(spanID)
		if !parentID.IsEmpty() {
			span.SetParentSpanID(parentID)
		}
		span.SetName(name)
		if kind != ptrace.SpanKindUnspecified {
			span.SetKind(kind)
		}
	}

	appendSpan("svc-1", "root", rootID, pcommon.SpanID{}, ptrace.SpanKindServer)
	appendSpan("svc-1", "client", clientID, rootID, ptrace.SpanKindClient)
	appendSpan("svc-2", "server", serverID, clientID, ptrace.SpanKindServer)
	appendSpan("svc-1", "callback", callbackID, serverID, ptrace.SpanKindInternal)

	tree, err := BuildTraceTree(td, nil)
	require.NoError(t, err)

	svc1Instances := filterInstancesByService(tree.ServiceInstances, "svc-1")
	require.Len(t, svc1Instances, 2)
	require.ElementsMatch(t, []string{"root", "client"}, spanNames(svc1Instances[0].Spans))
	require.ElementsMatch(t, []string{"client"}, spanNames(svc1Instances[0].OutputLeaves))
	require.ElementsMatch(t, []string{"callback"}, spanNames(svc1Instances[1].Spans))
	require.Empty(t, svc1Instances[1].OutputLeaves)

	svc2Instances := filterInstancesByService(tree.ServiceInstances, "svc-2")
	require.Len(t, svc2Instances, 1)
	require.ElementsMatch(t, []string{"server"}, spanNames(svc2Instances[0].Spans))
	require.ElementsMatch(t, []string{"server"}, spanNames(svc2Instances[0].OutputLeaves))
}

func TestBuildTraceTree_ServiceInstanceOutputLeaves_Producer(t *testing.T) {
	td := ptrace.NewTraces()

	rootID := pcommon.SpanID([8]byte{1})
	producerID := pcommon.SpanID([8]byte{2})

	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "svc-1")
	ss := rs.ScopeSpans().AppendEmpty()

	root := ss.Spans().AppendEmpty()
	root.SetSpanID(rootID)
	root.SetName("root")
	root.SetKind(ptrace.SpanKindServer)

	producer := ss.Spans().AppendEmpty()
	producer.SetSpanID(producerID)
	producer.SetParentSpanID(rootID)
	producer.SetName("publish")
	producer.SetKind(ptrace.SpanKindProducer)

	tree, err := BuildTraceTree(td, nil)
	require.NoError(t, err)
	require.Len(t, tree.ServiceInstances, 1)
	require.ElementsMatch(t, []string{"publish"}, spanNames(tree.ServiceInstances[0].OutputLeaves))
}

func TestBuildTraceTree_ServiceInstances_WorkloadKey(t *testing.T) {
	td := ptrace.NewTraces()

	rootID := pcommon.SpanID([8]byte{1})
	childID := pcommon.SpanID([8]byte{2})
	otherWorkloadID := pcommon.SpanID([8]byte{3})

	appendSpan := func(namespace, deployment, container, serviceName, name string, spanID, parentID pcommon.SpanID) {
		rs := td.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr("service.name", serviceName)
		rs.Resource().Attributes().PutStr("k8s.namespace.name", namespace)
		rs.Resource().Attributes().PutStr("k8s.deployment.name", deployment)
		rs.Resource().Attributes().PutStr("k8s.container.name", container)
		ss := rs.ScopeSpans().AppendEmpty()
		span := ss.Spans().AppendEmpty()
		span.SetSpanID(spanID)
		if !parentID.IsEmpty() {
			span.SetParentSpanID(parentID)
		}
		span.SetName(name)
	}

	appendSpan("default", "checkout", "app", "checkout", "root", rootID, pcommon.SpanID{})
	appendSpan("default", "checkout", "app", "checkout", "internal", childID, rootID)
	appendSpan("default", "checkout", "sidecar", "checkout", "sidecar", otherWorkloadID, rootID)

	td.ResourceSpans().At(0).Resource().Attributes().PutStr(TelemetrySDKLanguageAttribute, "java")
	td.ResourceSpans().At(0).Resource().Attributes().PutStr(ProcessRuntimeNameAttribute, "OpenJDK Runtime Environment")
	td.ResourceSpans().At(0).Resource().Attributes().PutStr(ProcessRuntimeVersionAttribute, "17.0.12")

	resolver := func(resource pcommon.Resource) (string, pcommon.Map, bool) {
		attrs := resource.Attributes()
		namespace, ok := attrs.Get("k8s.namespace.name")
		if !ok {
			return "", pcommon.NewMap(), false
		}
		deployment, ok := attrs.Get("k8s.deployment.name")
		if !ok {
			return "", pcommon.NewMap(), false
		}
		container, ok := attrs.Get("k8s.container.name")
		if !ok {
			return "", pcommon.NewMap(), false
		}
		key := namespace.Str() + "/Deployment/" + deployment.Str() + "/" + container.Str()
		identityAttrs := pcommon.NewMap()
		namespace.CopyTo(identityAttrs.PutEmpty("k8s.namespace.name"))
		deployment.CopyTo(identityAttrs.PutEmpty("k8s.deployment.name"))
		container.CopyTo(identityAttrs.PutEmpty("k8s.container.name"))
		return key, identityAttrs, true
	}

	tree, err := BuildTraceTree(td, resolver)
	require.NoError(t, err)
	require.Len(t, tree.ServiceInstances, 2)

	require.Equal(t, "default/Deployment/checkout/app", tree.ServiceInstances[0].WorkloadKey)
	require.Equal(t, 6, tree.ServiceInstances[0].ResourceAttributes.Len())
	language, ok := tree.ServiceInstances[0].ResourceAttributes.Get(TelemetrySDKLanguageAttribute)
	require.True(t, ok)
	require.Equal(t, "java", language.Str())
	runtimeName, ok := tree.ServiceInstances[0].ResourceAttributes.Get(ProcessRuntimeNameAttribute)
	require.True(t, ok)
	require.Equal(t, "OpenJDK Runtime Environment", runtimeName.Str())
	runtimeVersion, ok := tree.ServiceInstances[0].ResourceAttributes.Get(ProcessRuntimeVersionAttribute)
	require.True(t, ok)
	require.Equal(t, "17.0.12", runtimeVersion.Str())
	require.ElementsMatch(t, []string{"root", "internal"}, spanNames(tree.ServiceInstances[0].Spans))

	require.Equal(t, "default/Deployment/checkout/sidecar", tree.ServiceInstances[1].WorkloadKey)
	require.ElementsMatch(t, []string{"sidecar"}, spanNames(tree.ServiceInstances[1].Spans))
}

func filterInstancesByService(instances []*ServiceInstance, serviceName string) []*ServiceInstance {
	var filtered []*ServiceInstance
	for _, instance := range instances {
		if instance.ServiceName == serviceName {
			filtered = append(filtered, instance)
		}
	}
	return filtered
}

func spanNames(nodes []*TraceTreeNode) []string {
	names := make([]string, 0, len(nodes))
	for _, node := range nodes {
		names = append(names, node.Span.Name())
	}
	return names
}
