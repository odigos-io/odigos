package ebpf

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/instrumentation"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCustomStatusPreservesSDKHealthAndChecksOwner(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := odigosv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	yes := true
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "test", UID: "pod-1"}}
	details := &K8sProcessDetails{Pod: pod, ContainerName: "app", Pw: &k8sconsts.PodWorkload{Name: "app", Kind: "Deployment"}}
	inst := &odigosv1.InstrumentationInstance{ObjectMeta: metav1.ObjectMeta{Name: "app-42", Namespace: "test", UID: "instance-1", OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "Pod", Name: "app", UID: pod.UID, Controller: &yes}}}, Spec: odigosv1.InstrumentationInstanceSpec{ContainerName: "app"}, Status: odigosv1.InstrumentationInstanceStatus{Healthy: &yes, Reason: "LoadedSuccessfully", Components: []odigosv1.InstrumentationLibraryStatus{{Name: "net/http", Healthy: &yes}}}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(inst).WithObjects(pod, inst).Build()
	r := &k8sReporter{client: c}
	status := instrumentation.Status{CustomProbes: &instrumentationrules.CustomProbeReport{RuntimeID: "native-1", Revision: 3, Probes: []instrumentationrules.CustomProbeStatus{{Revision: 3, Probe: "main.work", Generation: "0123456789abcdef0123456789abcdef", State: "installed", UpdatedAt: "2026-09-10T00:00:00Z"}}}}
	if err := r.OnStatus(t.Context(), 42, details, status); err != nil {
		t.Fatal(err)
	}
	var got odigosv1.InstrumentationInstance
	key := client.ObjectKeyFromObject(inst)
	if err := c.Get(t.Context(), key, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.CustomProbes.Revision != 3 || got.Status.Reason != "LoadedSuccessfully" || len(got.Status.Components) != 1 || !*got.Status.Healthy {
		t.Fatal("custom report changed unrelated SDK health")
	}
	version := got.ResourceVersion
	status.CustomProbes.Revision = 2
	if err := r.OnStatus(t.Context(), 42, details, status); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(t.Context(), key, &got); err != nil {
		t.Fatal(err)
	}
	if got.ResourceVersion != version {
		t.Fatal("older report wrote to Kubernetes")
	}
	pod.UID = "replacement-pod"
	status.CustomProbes.Revision = 4
	if err := r.OnStatus(t.Context(), 42, details, status); err == nil {
		t.Fatal("accepted status for a different pod incarnation")
	}
}
