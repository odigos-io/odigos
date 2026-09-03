package describe

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
	sourcedescribe "github.com/odigos-io/odigos/k8sutils/pkg/describe/source"
)

const (
	workloadTestNs   = "shop"
	workloadTestName = "checkout"
)

var (
	deploymentConfigGVR = schema.GroupVersionResource{Group: "apps.openshift.io", Version: "v1", Resource: "deploymentconfigs"}
	rolloutGVR          = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "rollouts"}
)

func workloadNamespace() *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: workloadTestNs}}
}

func workloadPodTemplate() corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": workloadTestName}},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: workloadTestName}},
		},
	}
}

func workloadPod(name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: workloadTestNs,
			Labels:    map[string]string{"app": workloadTestName},
		},
		Spec: corev1.PodSpec{
			NodeName:   "node-1",
			Containers: []corev1.Container{{Name: workloadTestName}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
}

func workloadSource(kind k8sconsts.WorkloadKind) *odigosv1.Source {
	return &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workloadTestName + "-source",
			Namespace: workloadTestNs,
			Labels: map[string]string{
				k8sconsts.WorkloadNameLabel:      workloadTestName,
				k8sconsts.WorkloadNamespaceLabel: workloadTestNs,
				k8sconsts.WorkloadKindLabel:      string(kind),
			},
		},
	}
}

func workloadInstrumentationConfig(runtimeObjectName string) *odigosv1.InstrumentationConfig {
	return &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: runtimeObjectName, Namespace: workloadTestNs},
		Spec: odigosv1.InstrumentationConfigSpec{
			Containers: []odigosv1.ContainerAgentConfig{
				{ContainerName: workloadTestName, AgentEnabled: true, OtelDistroName: "golang-community"},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName:  workloadTestName,
					Language:       common.GoProgrammingLanguage,
					RuntimeVersion: "1.24.0",
					EnvVars:        []odigosv1.EnvVar{{Name: "GODEBUG", Value: "http2debug=1"}},
				},
			},
		},
	}
}

func newDynamicClient(gvr schema.GroupVersionResource, listKind string, objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	scheme := runtime.NewScheme()
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{gvr: listKind}, objects...)
}

func TestDescribeSourceToText(t *testing.T) {
	createTime := properties.EntityProperty{Name: "create time", Value: "2026-08-28 00:00:00 +0000 UTC"}
	reason := properties.EntityProperty{Name: "Agent Enabled Reason", Value: "UnsupportedProgrammingLanguage"}
	instanceMessage := properties.EntityProperty{Name: "Message", Value: "failed to load eBPF probes"}
	analyze := &sourcedescribe.SourceAnalyze{
		Name:      properties.EntityProperty{Name: "Name", Value: workloadTestName},
		Kind:      properties.EntityProperty{Name: "Kind", Value: k8sconsts.WorkloadKindDeployment},
		Namespace: properties.EntityProperty{Name: "Namespace", Value: workloadTestNs},
		SourceObjectsAnalysis: sourcedescribe.InstrumentationSourcesAnalyze{
			Instrumented:     properties.EntityProperty{Name: "Instrumented", Value: true},
			InstrumentedText: properties.EntityProperty{Name: "DecisionText", Value: "Workload is instrumented"},
		},
		RuntimeInfo: &sourcedescribe.RuntimeInfoAnalyze{
			Containers: []sourcedescribe.ContainerRuntimeInfoAnalyze{{
				ContainerName:  properties.EntityProperty{Name: "Container Name", Value: workloadTestName, ListKey: true},
				Language:       properties.EntityProperty{Name: "Programming Language", Value: common.GoProgrammingLanguage, Status: properties.PropertyStatusSuccess},
				RuntimeVersion: properties.EntityProperty{Name: "Runtime Version", Value: "1.24.0"},
				EnvVars:        []properties.EntityProperty{{Name: "GODEBUG", Value: "http2debug=1"}},
			}},
		},
		OtelAgents: sourcedescribe.OtelAgentsAnalyze{
			Created:    properties.EntityProperty{Name: "Created", Value: "created", Status: properties.PropertyStatusSuccess},
			CreateTime: &createTime,
			Containers: []sourcedescribe.ContainerAgentConfigAnalyze{{
				ContainerName: properties.EntityProperty{Name: "Container Name", Value: workloadTestName, ListKey: true},
				AgentEnabled:  properties.EntityProperty{Name: "Agent Enabled", Value: false},
				Reason:        &reason,
			}},
		},
		TotalPods:       1,
		PodsPhasesCount: "Running 1",
		Pods: []sourcedescribe.PodAnalyze{{
			PodName:       properties.EntityProperty{Name: "Pod Name", Value: "checkout-1", ListKey: true},
			NodeName:      properties.EntityProperty{Name: "Node Name", Value: "node-1"},
			Phase:         properties.EntityProperty{Name: "Phase", Value: corev1.PodRunning, Status: properties.PropertyStatusSuccess},
			AgentInjected: properties.EntityProperty{Name: "Odigos Agent Injected", Value: true, Status: properties.PropertyStatusSuccess},
			Containers: []sourcedescribe.PodContainerAnalyze{{
				ContainerName: properties.EntityProperty{Name: "Container Name", Value: workloadTestName, ListKey: true},
				ActualDevices: properties.EntityProperty{Name: "Actual Devices", Value: []string{"golang-community"}},
				InstrumentationInstances: []sourcedescribe.InstrumentationInstanceAnalyze{{
					Healthy: properties.EntityProperty{Name: "Healthy", Value: false, Status: properties.PropertyStatusError, ListKey: true},
					Message: &instanceMessage,
					IdentifyingAttributes: []properties.EntityProperty{
						{Name: "service.name", Value: workloadTestName},
					},
				}},
			}},
		}},
	}

	text := DescribeSourceToText(analyze)

	assert.Contains(t, text, "Name: checkout")
	assert.Contains(t, text, "Kind: Deployment")
	assert.Contains(t, text, "Namespace: shop")
	assert.Contains(t, text, "Source Custom Resources:")
	assert.Contains(t, text, "DecisionText: Workload is instrumented")
	assert.Contains(t, text, "Runtime Inspection Details:")
	assert.Contains(t, text, "Detected Containers:")
	assert.Contains(t, text, greenPrefix+"Programming Language: go"+colorSuffix)
	assert.Contains(t, text, "Runtime Version: 1.24.0")
	assert.Contains(t, text, "Relevant Environment Variables:")
	assert.Contains(t, text, "- GODEBUG: http2debug=1")
	assert.Contains(t, text, "Instrumentation Config:")
	assert.Contains(t, text, "create time: 2026-08-28 00:00:00 +0000 UTC")
	assert.Contains(t, text, "Agent Enabled Reason: UnsupportedProgrammingLanguage")
	assert.Contains(t, text, "Pods (Total 1, Running 1):")
	assert.Contains(t, text, "Node Name: node-1")
	assert.Contains(t, text, "Actual Devices: [golang-community]")
	assert.Contains(t, text, "Instrumentation Instances:")
	assert.Contains(t, text, redPrefix+"Healthy: false"+colorSuffix)
	assert.Contains(t, text, "Message: failed to load eBPF probes")
	assert.Contains(t, text, "Identifying Attributes:")
	assert.Contains(t, text, "service.name: checkout")

	// the sections are printed in a fixed order
	sectionOrder := []string{
		"Source Custom Resources:",
		"Runtime Inspection Details:",
		"Instrumentation Config:",
		"Pods (Total 1, Running 1):",
	}
	for i := 1; i < len(sectionOrder); i++ {
		assert.Less(t, strings.Index(text, sectionOrder[i-1]), strings.Index(text, sectionOrder[i]))
	}
}

func TestDescribeSourceToTextWithoutRuntimeDetails(t *testing.T) {
	analyze := &sourcedescribe.SourceAnalyze{
		Name: properties.EntityProperty{Name: "Name", Value: workloadTestName},
		OtelAgents: sourcedescribe.OtelAgentsAnalyze{
			Created: properties.EntityProperty{Name: "Created", Value: "not created"},
		},
		PodsPhasesCount: "",
	}

	text := DescribeSourceToText(analyze)

	assert.Contains(t, text, "No runtime details")
	assert.NotContains(t, text, "Detected Containers:")
	assert.Contains(t, text, "Pods (Total 0, ):")
	// the optional instrumentation config properties are omitted
	assert.NotContains(t, text, "create time")
}

func TestDescribeDeployment(t *testing.T) {
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        workloadTestName + "-abc",
			Namespace:   workloadTestNs,
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			Labels: map[string]string{
				"app":               workloadTestName,
				"pod-template-hash": "abc",
			},
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "Deployment", Name: workloadTestName, UID: types.UID("deployment-uid"),
			}},
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app":               workloadTestName,
				"pod-template-hash": "abc",
			}},
		},
	}
	pod := workloadPod(workloadTestName + "-abc-1")
	pod.Labels["pod-template-hash"] = "abc"
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        workloadTestName,
			Namespace:   workloadTestNs,
			UID:         types.UID("deployment-uid"),
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": workloadTestName}},
			Template: workloadPodTemplate(),
		},
	}

	analyze, err := DescribeSource(context.Background(),
		fake.NewSimpleClientset(workloadNamespace(), deployment, replicaSet, pod),
		odigosfake.NewSimpleClientset(
			workloadSource(k8sconsts.WorkloadKindDeployment),
			workloadInstrumentationConfig("deployment-"+workloadTestName),
		).OdigosV1alpha1(),
		&sourcedescribe.K8sSourceObject{
			Kind:            k8sconsts.WorkloadKindDeployment,
			ObjectMeta:      deployment.ObjectMeta,
			PodTemplateSpec: &deployment.Spec.Template,
			LabelSelector:   deployment.Spec.Selector,
		})

	require.NoError(t, err)
	assert.Equal(t, workloadTestName, analyze.Name.Value)
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, analyze.Kind.Value)
	assert.Equal(t, true, analyze.SourceObjectsAnalysis.Instrumented.Value)
	require.NotNil(t, analyze.RuntimeInfo)
	require.Len(t, analyze.RuntimeInfo.Containers, 1)
	assert.Equal(t, common.GoProgrammingLanguage, analyze.RuntimeInfo.Containers[0].Language.Value)
	assert.Equal(t, 1, analyze.TotalPods)
	require.Len(t, analyze.Pods, 1)
	require.NotNil(t, analyze.Pods[0].RunningLatestWorkloadRevision)
	assert.Equal(t, true, analyze.Pods[0].RunningLatestWorkloadRevision.Value)

	// the describe entry point resolves the workload from the api server itself
	fromApiServer, err := DescribeDeployment(context.Background(),
		fake.NewSimpleClientset(workloadNamespace(), deployment, replicaSet, pod),
		odigosfake.NewSimpleClientset(workloadSource(k8sconsts.WorkloadKindDeployment)).OdigosV1alpha1(),
		workloadTestNs, workloadTestName)

	require.NoError(t, err)
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, fromApiServer.Kind.Value)
	assert.Equal(t, 1, fromApiServer.TotalPods)

	_, err = DescribeDeployment(context.Background(), fake.NewSimpleClientset(workloadNamespace()),
		odigosfake.NewSimpleClientset().OdigosV1alpha1(), workloadTestNs, "does-not-exist")
	require.Error(t, err)
	assert.ErrorContains(t, err, "does-not-exist")

	// the workload namespace must exist for the resources of the source to be collected
	_, err = DescribeSource(context.Background(), fake.NewSimpleClientset(),
		odigosfake.NewSimpleClientset().OdigosV1alpha1(),
		&sourcedescribe.K8sSourceObject{
			Kind:          k8sconsts.WorkloadKindDeployment,
			ObjectMeta:    deployment.ObjectMeta,
			LabelSelector: deployment.Spec.Selector,
		})
	require.Error(t, err)
	assert.ErrorContains(t, err, workloadTestNs)
}

func TestDescribeDaemonSet(t *testing.T) {
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: workloadTestName, Namespace: workloadTestNs},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": workloadTestName}},
			Template: workloadPodTemplate(),
		},
	}

	analyze, err := DescribeDaemonSet(context.Background(),
		fake.NewSimpleClientset(workloadNamespace(), daemonSet, workloadPod(workloadTestName+"-1")),
		odigosfake.NewSimpleClientset(workloadSource(k8sconsts.WorkloadKindDaemonSet)).OdigosV1alpha1(),
		workloadTestNs, workloadTestName)

	require.NoError(t, err)
	assert.Equal(t, k8sconsts.WorkloadKindDaemonSet, analyze.Kind.Value)
	assert.Equal(t, true, analyze.SourceObjectsAnalysis.Instrumented.Value)
	assert.Equal(t, 1, analyze.TotalPods)
	// pods of a daemonset are resolved by the label selector, which knows nothing about revisions
	assert.Nil(t, analyze.Pods[0].RunningLatestWorkloadRevision)

	_, err = DescribeDaemonSet(context.Background(), fake.NewSimpleClientset(workloadNamespace()),
		odigosfake.NewSimpleClientset().OdigosV1alpha1(), workloadTestNs, "does-not-exist")
	require.Error(t, err)
}

func TestDescribeStatefulSet(t *testing.T) {
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: workloadTestName, Namespace: workloadTestNs},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": workloadTestName}},
			Template: workloadPodTemplate(),
		},
	}

	analyze, err := DescribeStatefulSet(context.Background(),
		fake.NewSimpleClientset(workloadNamespace(), statefulSet, workloadPod(workloadTestName+"-0")),
		odigosfake.NewSimpleClientset(workloadSource(k8sconsts.WorkloadKindStatefulSet)).OdigosV1alpha1(),
		workloadTestNs, workloadTestName)

	require.NoError(t, err)
	assert.Equal(t, k8sconsts.WorkloadKindStatefulSet, analyze.Kind.Value)
	assert.Equal(t, true, analyze.SourceObjectsAnalysis.Instrumented.Value)
	assert.Equal(t, 1, analyze.TotalPods)

	_, err = DescribeStatefulSet(context.Background(), fake.NewSimpleClientset(workloadNamespace()),
		odigosfake.NewSimpleClientset().OdigosV1alpha1(), workloadTestNs, "does-not-exist")
	require.Error(t, err)
}

func TestDescribeStaticPod(t *testing.T) {
	// a static pod has no label selector, so the pod is resolved by the workload name
	analyze, err := DescribeStaticPod(context.Background(),
		fake.NewSimpleClientset(workloadNamespace(), workloadPod(workloadTestName)),
		odigosfake.NewSimpleClientset(workloadSource(k8sconsts.WorkloadKindStaticPod)).OdigosV1alpha1(),
		workloadTestNs, workloadTestName)

	require.NoError(t, err)
	assert.Equal(t, k8sconsts.WorkloadKindStaticPod, analyze.Kind.Value)
	assert.Equal(t, true, analyze.SourceObjectsAnalysis.Instrumented.Value)
	require.Len(t, analyze.Pods, 1)
	assert.Equal(t, workloadTestName, analyze.Pods[0].PodName.Value)

	_, err = DescribeStaticPod(context.Background(), fake.NewSimpleClientset(workloadNamespace()),
		odigosfake.NewSimpleClientset().OdigosV1alpha1(), workloadTestNs, "does-not-exist")
	require.Error(t, err)
}

func TestDescribeDeploymentConfig(t *testing.T) {
	deploymentConfig := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps.openshift.io/v1",
		"kind":       "DeploymentConfig",
		"metadata": map[string]interface{}{
			"name":      workloadTestName,
			"namespace": workloadTestNs,
		},
		"spec": map[string]interface{}{
			"selector": map[string]interface{}{"app": workloadTestName},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{"app": workloadTestName},
				},
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{"name": workloadTestName},
					},
				},
			},
		},
	}}

	t.Run("the openshift selector map is converted to a label selector", func(t *testing.T) {
		analyze, err := DescribeDeploymentConfig(context.Background(),
			fake.NewSimpleClientset(workloadNamespace(), workloadPod(workloadTestName+"-1")),
			newDynamicClient(deploymentConfigGVR, "DeploymentConfigList", deploymentConfig),
			odigosfake.NewSimpleClientset(workloadSource(k8sconsts.WorkloadKindDeploymentConfig)).OdigosV1alpha1(),
			workloadTestNs, workloadTestName)

		require.NoError(t, err)
		assert.Equal(t, k8sconsts.WorkloadKindDeploymentConfig, analyze.Kind.Value)
		assert.Equal(t, true, analyze.SourceObjectsAnalysis.Instrumented.Value)
		assert.Equal(t, 1, analyze.TotalPods)
	})

	t.Run("a missing deployment config is an error", func(t *testing.T) {
		_, err := DescribeDeploymentConfig(context.Background(),
			fake.NewSimpleClientset(workloadNamespace()),
			newDynamicClient(deploymentConfigGVR, "DeploymentConfigList"),
			odigosfake.NewSimpleClientset().OdigosV1alpha1(), workloadTestNs, "does-not-exist")

		require.Error(t, err)
		assert.ErrorContains(t, err, "does-not-exist")
	})

	t.Run("a deployment config that cannot be converted is reported", func(t *testing.T) {
		malformed := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "apps.openshift.io/v1",
			"kind":       "DeploymentConfig",
			"metadata": map[string]interface{}{
				"name":      workloadTestName,
				"namespace": workloadTestNs,
			},
			"spec": map[string]interface{}{"replicas": "not-a-number"},
		}}

		_, err := DescribeDeploymentConfig(context.Background(),
			fake.NewSimpleClientset(workloadNamespace()),
			newDynamicClient(deploymentConfigGVR, "DeploymentConfigList", malformed),
			odigosfake.NewSimpleClientset().OdigosV1alpha1(), workloadTestNs, workloadTestName)

		require.ErrorContains(t, err, "failed to convert unstructured to DeploymentConfig")
	})
}

func TestDescribeRollout(t *testing.T) {
	rollout := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Rollout",
		"metadata": map[string]interface{}{
			"name":      workloadTestName,
			"namespace": workloadTestNs,
		},
		"spec": map[string]interface{}{
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{"app": workloadTestName},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{"app": workloadTestName},
				},
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{"name": workloadTestName},
					},
				},
			},
		},
	}}

	t.Run("the rollout match labels become the pod selector", func(t *testing.T) {
		analyze, err := DescribeRollout(context.Background(),
			fake.NewSimpleClientset(workloadNamespace(), workloadPod(workloadTestName+"-1")),
			newDynamicClient(rolloutGVR, "RolloutList", rollout),
			odigosfake.NewSimpleClientset(workloadSource(k8sconsts.WorkloadKindArgoRollout)).OdigosV1alpha1(),
			workloadTestNs, workloadTestName)

		require.NoError(t, err)
		assert.Equal(t, k8sconsts.WorkloadKindArgoRollout, analyze.Kind.Value)
		assert.Equal(t, true, analyze.SourceObjectsAnalysis.Instrumented.Value)
		assert.Equal(t, 1, analyze.TotalPods)
	})

	t.Run("a missing rollout is an error", func(t *testing.T) {
		_, err := DescribeRollout(context.Background(),
			fake.NewSimpleClientset(workloadNamespace()),
			newDynamicClient(rolloutGVR, "RolloutList"),
			odigosfake.NewSimpleClientset().OdigosV1alpha1(), workloadTestNs, "does-not-exist")

		require.Error(t, err)
		assert.ErrorContains(t, err, "does-not-exist")
	})

	t.Run("a rollout that cannot be converted is reported", func(t *testing.T) {
		malformed := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Rollout",
			"metadata": map[string]interface{}{
				"name":      workloadTestName,
				"namespace": workloadTestNs,
			},
			"spec": map[string]interface{}{"replicas": "not-a-number"},
		}}

		_, err := DescribeRollout(context.Background(),
			fake.NewSimpleClientset(workloadNamespace()),
			newDynamicClient(rolloutGVR, "RolloutList", malformed),
			odigosfake.NewSimpleClientset().OdigosV1alpha1(), workloadTestNs, workloadTestName)

		require.ErrorContains(t, err, "failed to convert unstructured to Rollout")
	})
}
