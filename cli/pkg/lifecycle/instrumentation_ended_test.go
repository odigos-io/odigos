package lifecycle

import (
	"context"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/cli/pkg/kube"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

const (
	testNamespace = "default"
	testAgentHash = "abcd1234"
)

func testDeployment(replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-deployment",
			Namespace:  testNamespace,
			Generation: 1,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
		},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 1,
			Replicas:           replicas,
			UpdatedReplicas:    replicas,
			AvailableReplicas:  replicas,
		},
	}
}

// testPod builds a pod that matches the deployment selector. When instrumented is
// true the pod carries the agent meta hash label, exactly as the pods webhook
// writes it.
func testPod(name string, instrumented bool, phase corev1.PodPhase) *corev1.Pod {
	labels := map[string]string{"app": "test"}
	if instrumented {
		labels[k8sconsts.OdigosAgentsMetaHashLabel] = testAgentHash
	}
	started := true
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
			Labels:    labels,
		},
		Status: corev1.PodStatus{
			Phase: phase,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", Ready: true, Started: &started, RestartCount: 0},
			},
		},
	}
}

func testInstrumentationConfig(podManifestInjectionOptional bool) *odigosv1.InstrumentationConfig {
	return &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test-deployment",
			Namespace: testNamespace,
		},
		Spec: odigosv1.InstrumentationConfigSpec{
			AgentInjectionEnabled:        true,
			AgentsMetaHash:               testAgentHash,
			PodManifestInjectionOptional: podManifestInjectionOptional,
			Containers: []odigosv1.ContainerAgentConfig{
				{ContainerName: "app", AgentEnabled: true, OtelDistroName: "test-distro"},
			},
		},
	}
}

func newTransition(t *testing.T, ic *odigosv1.InstrumentationConfig, pods ...runtime.Object) *InstrumentationEnded {
	t.Helper()

	odigosObjects := []runtime.Object{}
	if ic != nil {
		odigosObjects = append(odigosObjects, ic)
	}

	return &InstrumentationEnded{BaseTransition{
		client: &kube.Client{
			Interface:    k8sfake.NewSimpleClientset(pods...),
			OdigosClient: odigosfake.NewSimpleClientset(odigosObjects...).OdigosV1alpha1(),
		},
		odigosNamespace: "odigos-system",
	}}
}

func TestAllInstrumentedPodsAreRunning(t *testing.T) {
	tests := []struct {
		name string
		ic   *odigosv1.InstrumentationConfig
		pods []runtime.Object
		want bool
	}{
		{
			// The rollout was triggered but the webhook has not relabeled anything yet.
			// Reporting the rollout as done here makes the CLI exit successfully with
			// nothing instrumented.
			name: "no pod carries the agent hash yet",
			ic:   testInstrumentationConfig(false),
			pods: []runtime.Object{
				testPod("old-1", false, corev1.PodRunning),
				testPod("old-2", false, corev1.PodRunning),
				testPod("old-3", false, corev1.PodRunning),
			},
			want: false,
		},
		{
			name: "only the first replica was replaced",
			ic:   testInstrumentationConfig(false),
			pods: []runtime.Object{
				testPod("old-1", false, corev1.PodRunning),
				testPod("old-2", false, corev1.PodRunning),
				testPod("new-1", true, corev1.PodRunning),
			},
			want: false,
		},
		{
			name: "an instrumented pod is still starting",
			ic:   testInstrumentationConfig(false),
			pods: []runtime.Object{
				testPod("new-1", true, corev1.PodRunning),
				testPod("new-2", true, corev1.PodRunning),
				testPod("new-3", true, corev1.PodPending),
			},
			want: false,
		},
		{
			name: "every pod is instrumented and running",
			ic:   testInstrumentationConfig(false),
			pods: []runtime.Object{
				testPod("new-1", true, corev1.PodRunning),
				testPod("new-2", true, corev1.PodRunning),
				testPod("new-3", true, corev1.PodRunning),
			},
			want: true,
		},
		{
			// No-restart distros (OBI) are injected without a pod manifest change, so
			// the running pods never get relabeled and waiting for the label would
			// block until the poll times out.
			name: "pod manifest injection is optional",
			ic:   testInstrumentationConfig(true),
			pods: []runtime.Object{
				testPod("running-1", false, corev1.PodRunning),
				testPod("running-2", false, corev1.PodRunning),
			},
			want: true,
		},
		{
			name: "workload is scaled to zero",
			ic:   testInstrumentationConfig(false),
			pods: []runtime.Object{},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transition := newTransition(t, tt.ic, tt.pods...)

			got, err := transition.allInstrumentedPodsAreRunning(context.Background(), testDeployment(int32(len(tt.pods))))
			if err != nil {
				t.Fatalf("allInstrumentedPodsAreRunning returned an error: %v", err)
			}
			if got != tt.want {
				t.Errorf("allInstrumentedPodsAreRunning = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOrchestratorDoesNotSkipTheRollout drives the state machine over a workload whose
// Source and InstrumentationConfig are ready but whose pods were not replaced yet, which
// is the state right after the instrumentor requested the rollout.
func TestOrchestratorDoesNotSkipTheRollout(t *testing.T) {
	deployment := testDeployment(2)

	source := &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-test-deployment",
			Namespace: testNamespace,
			Labels: map[string]string{
				k8sconsts.WorkloadNameLabel:      deployment.Name,
				k8sconsts.WorkloadNamespaceLabel: testNamespace,
				k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindDeployment),
			},
		},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Kind:      k8sconsts.WorkloadKindDeployment,
				Name:      deployment.Name,
				Namespace: testNamespace,
			},
		},
	}

	ic := testInstrumentationConfig(false)
	ic.Status.Conditions = []metav1.Condition{
		{
			Type:               odigosv1.RuntimeDetectionStatusConditionType,
			Status:             metav1.ConditionTrue,
			Reason:             string(odigosv1.RuntimeDetectionReasonDetectedSuccessfully),
			LastTransitionTime: metav1.Now(),
		},
	}

	client := &kube.Client{
		Interface: k8sfake.NewSimpleClientset(
			testPod("old-1", false, corev1.PodRunning),
			testPod("old-2", false, corev1.PodRunning),
		),
		OdigosClient: odigosfake.NewSimpleClientset(source, ic).OdigosV1alpha1(),
	}
	base := BaseTransition{client: client, odigosNamespace: "odigos-system"}

	orchestrator := &Orchestrator{
		Client: client,
		TransitionsMap: map[State]Transition{
			StateNoSourceCreated:           &PreflightCheck{base},
			StatePreflightChecksPassed:     &RequestLangDetection{base},
			StateSourceCreated:             &WaitForLangDetection{base},
			StateLangDetected:              &InstrumentationStarted{base},
			StateInstrumentationInProgress: &InstrumentationEnded{base},
			StateInstrumented:              &PostCheck{base},
			StatePostCheckPassed:           nil,
		},
	}

	state, err := orchestrator.getCurrentState(context.Background(), deployment)
	if err != nil {
		t.Fatalf("getCurrentState returned an error: %v", err)
	}
	if state != StateInstrumentationInProgress {
		t.Errorf("getCurrentState = %s, want %s", state, StateInstrumentationInProgress)
	}
}

func TestGetTransitionStateWaitsForTheRollout(t *testing.T) {
	transition := newTransition(t, testInstrumentationConfig(false),
		testPod("old-1", false, corev1.PodRunning),
		testPod("old-2", false, corev1.PodRunning),
	)

	state, err := transition.GetTransitionState(context.Background(), testDeployment(2))
	if err != nil {
		t.Fatalf("GetTransitionState returned an error: %v", err)
	}
	if state != StateInstrumentationInProgress {
		t.Errorf("GetTransitionState = %s, want %s", state, StateInstrumentationInProgress)
	}
}
