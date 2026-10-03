package instrumentednodes

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// deliberately not the "default" namespace, so a hardcoded namespace in the
	// production code cannot pass the tests.
	instrumentedNodesNamespace = "payments"
	// a second namespace used for decoys, to pin namespace scoping.
	instrumentedNodesOtherNamespace = "payments-staging"

	agentsNodeName      = "node-running-agents"
	otherNodeName       = "node-running-nothing"
	checkoutWorkload    = "checkout"
	checkoutConfigName  = "deployment-checkout"
	checkoutReplicaSet  = "checkout-7d4c8b5f9b"
	reportsCronJob      = "reports"
	reportsConfigName   = "cronjob-reports"
	reportsJobName      = "reports-28901234"
	instrumentedPodHash = "a1b2c3d4"
)

func instrumentedNodesScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = odigosv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = batchv1.AddToScheme(scheme)
	return scheme
}

func instrumentedNodesContext() context.Context {
	return ctrllog.IntoContext(context.Background(), logr.Discard())
}

// newInstrumentedNodesClient builds a client whose pod-by-node index is the very
// index function the production SetupWithManager registers, so a change to the
// index breaks the tests that rely on listing pods by node.
func newInstrumentedNodesClient(t *testing.T, objects ...client.Object) client.WithWatch {
	t.Helper()
	return newInstrumentedNodesClientWithInterceptor(t, interceptor.Funcs{}, objects...)
}

func newInstrumentedNodesClientWithInterceptor(t *testing.T, funcs interceptor.Funcs, objects ...client.Object) client.WithWatch {
	t.Helper()
	return fake.NewClientBuilder().
		WithScheme(instrumentedNodesScheme()).
		WithObjects(objects...).
		WithIndex(&corev1.Pod{}, podNodeNameIndex, registeredPodNodeNameIndex(t)).
		WithInterceptorFuncs(funcs).
		Build()
}

// ****************
// Fixtures
// ****************

func nodeNamed(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"kubernetes.io/os": "linux"},
		},
	}
}

func nodeStampedAt(name string, labelValue string) *corev1.Node {
	node := nodeNamed(name)
	node.Labels[k8sconsts.FirstInstrumentedPodAtNodeLabel] = labelValue
	return node
}

// stampedAgo renders the label value a node would carry if its first
// instrumented pod had been discovered `ago` in the past.
func stampedAgo(ago time.Duration) string {
	return time.Now().UTC().Add(-ago).Format(k8sconsts.FirstInstrumentedPodAtNodeLabelTimeFormat)
}

// podOnNode is a bare pod with no owner: it belongs to no workload at all.
func podOnNode(name string, nodeName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instrumentedNodesNamespace,
		},
		Spec: corev1.PodSpec{NodeName: nodeName},
	}
}

func withAgentsInjected(pod *corev1.Pod) *corev1.Pod {
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel] = instrumentedPodHash
	return pod
}

// deploymentPod is a pod owned by a ReplicaSet of the `checkout` deployment.
func deploymentPod(name string, nodeName string) *corev1.Pod {
	pod := podOnNode(name, nodeName)
	pod.Labels = map[string]string{"app": checkoutWorkload}
	pod.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "apps/v1",
		Kind:       "ReplicaSet",
		Name:       checkoutReplicaSet,
	}}
	return pod
}

// cronJobPod is a pod owned by a Job of the `reports` cron job.
func cronJobPod(name string, nodeName string) *corev1.Pod {
	pod := podOnNode(name, nodeName)
	pod.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "batch/v1",
		Kind:       "Job",
		Name:       reportsJobName,
	}}
	return pod
}

func staticPod(name string, nodeName string) *corev1.Pod {
	pod := podOnNode(name, nodeName)
	pod.Annotations = map[string]string{"kubernetes.io/config.source": "file"}
	pod.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "v1",
		Kind:       "Node",
		Name:       nodeName,
	}}
	return pod
}

func checkoutDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      checkoutWorkload,
			Namespace: instrumentedNodesNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": checkoutWorkload}},
		},
	}
}

func reportsCronJobObject() *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      reportsCronJob,
			Namespace: instrumentedNodesNamespace,
		},
	}
}

func instrumentationConfig(name string, namespace string) *odigosv1.InstrumentationConfig {
	return &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
}

// ****************
// Assertions
// ****************

func nodeLabelValue(t *testing.T, c client.Client, nodeName string) (string, bool) {
	t.Helper()
	var node corev1.Node
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: nodeName}, &node))
	value, found := node.Labels[k8sconsts.FirstInstrumentedPodAtNodeLabel]
	return value, found
}

func requireNodeStamped(t *testing.T, c client.Client, nodeName string) string {
	t.Helper()
	value, found := nodeLabelValue(t, c, nodeName)
	require.True(t, found, "node %s should carry %s", nodeName, k8sconsts.FirstInstrumentedPodAtNodeLabel)
	return value
}

func requireNodeNotStamped(t *testing.T, c client.Client, nodeName string) {
	t.Helper()
	value, found := nodeLabelValue(t, c, nodeName)
	require.False(t, found, "node %s should not carry %s, got %q", nodeName, k8sconsts.FirstInstrumentedPodAtNodeLabel, value)
}

// firstErr drops the requeue duration of a syncNode call whose only interesting
// result is whether it failed.
func firstErr(_ time.Duration, err error) error {
	return err
}
