package ebpf

import (
	"context"
	"errors"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentation/detector"
	"github.com/odigos-io/odigos/odiglet/pkg/process"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestResolveWithoutEnv checks that a process with no injected env, such as a worker forked in a pod
// created before its workload was instrumented, resolves to its pod, container and distribution when
// the container's distribution needs no restart, and is refused otherwise.
func TestResolveWithoutEnv(t *testing.T) {
	const podUID, container, pid = "3d75f212-95fe-4403-9552-13f01467308e", "rates-service", 4242
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := odigosv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: "rates-service-7d4c8b5f9b-abcde", Namespace: "hard2", UID: podUID,
		OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "rates-service-7d4c8b5f9b", Controller: new(bool)}},
	}}
	ic := func(c odigosv1.ContainerAgentConfig) *odigosv1.InstrumentationConfig {
		return &odigosv1.InstrumentationConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "deployment-rates-service", Namespace: "hard2"},
			Spec:       odigosv1.InstrumentationConfigSpec{Containers: []odigosv1.ContainerAgentConfig{c}},
		}
	}
	getter, err := distros.NewCommunityGetter()
	if err != nil {
		t.Fatal(err)
	}
	event := detector.ProcessEvent{EventType: detector.ProcessForkEvent, PID: pid, ExecDetails: &detector.ExecDetails{
		ExePath: "/usr/local/bin/python3.12", Environments: map[string]string{},
	}}
	inPod := func(p int) (process.PodContainerKey, bool) {
		return process.PodContainerKey{PodUID: podUID, ContainerName: container}, p == pid
	}

	for _, tc := range []struct {
		name    string
		config  odigosv1.ContainerAgentConfig
		inPod   func(int) (process.PodContainerKey, bool)
		wantErr error
	}{
		{name: "no restart needed", config: odigosv1.ContainerAgentConfig{
			ContainerName: container, AgentEnabled: true, PodManifestInjectionOptional: true, OtelDistroName: "python-community",
		}, inPod: inPod},
		{name: "restart needed", config: odigosv1.ContainerAgentConfig{
			ContainerName: container, AgentEnabled: true, OtelDistroName: "python-community",
		}, inPod: inPod, wantErr: errPodNameNotReported},
		{name: "agent disabled", config: odigosv1.ContainerAgentConfig{
			ContainerName: container, PodManifestInjectionOptional: true, OtelDistroName: "python-community",
		}, inPod: inPod, wantErr: errPodNameNotReported},
		{name: "not in a pod", config: odigosv1.ContainerAgentConfig{
			ContainerName: container, AgentEnabled: true, PodManifestInjectionOptional: true, OtelDistroName: "python-community",
		}, inPod: func(int) (process.PodContainerKey, bool) { return process.PodContainerKey{}, false }, wantErr: errPodNameNotReported},
		{name: "another container", config: odigosv1.ContainerAgentConfig{
			ContainerName: "sidecar", AgentEnabled: true, PodManifestInjectionOptional: true, OtelDistroName: "python-community",
		}, inPod: inPod, wantErr: errContainerNameNotReported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod, ic(tc.config)).Build()
			dr := &k8sDetailsResolver{client: c, distributionGetter: getter, podContainerOfPid: tc.inPod}
			got, err := dr.Resolve(context.Background(), event)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Resolve = %v, %v; want %v", got, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Pod.Name != pod.Name || got.ContainerName != container || got.Distro == nil || got.Distro.Name != "python-community" ||
				got.Pw.Name != "rates-service" || got.Pw.Kind != k8sconsts.WorkloadKindDeployment || got.ProcEvent.PID != pid {
				t.Fatalf("Resolve = %+v", got)
			}
		})
	}
}
