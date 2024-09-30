package test

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/instrumentation_instance"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type FakeEbpfSdk struct {
	loadedIndicator chan struct{}
	running         bool
	pid 		    int

	closed          bool
	cancel          context.CancelFunc
	stopped         chan struct{}
}

// compile-time check that FakeEbpfSdk implements ConfigurableOtelEbpfSdk
var _ ebpf.ConfigurableOtelEbpfSdk = (*FakeEbpfSdk)(nil)

func (f *FakeEbpfSdk) ApplyConfig(ctx context.Context, config *odigosv1.InstrumentationConfig) error {
	return nil
}

func (f *FakeEbpfSdk) Close(_ context.Context) error {
	if f.cancel != nil {
		f.cancel()
	}
	<-f.stopped
	f.running = false
	f.closed = true
	return nil
}

func (f *FakeEbpfSdk) Run(ctx context.Context) error {
	// To simulate a real instrumentation, Run is blocking until the context is cancelled
	f.running = true
	close(f.loadedIndicator)

	cancelCtx, cancel := context.WithCancel(ctx)
	f.cancel = cancel
	f.stopped = make(chan struct{})

	<-cancelCtx.Done()
	close(f.stopped)
	return nil
}

type FakeInstrumentationFactory struct {
	kubeclient client.Client
}

func NewFakeInstrumentationFactory(kubeclient client.Client) ebpf.InstrumentationFactory[*FakeEbpfSdk] {
	return &FakeInstrumentationFactory{
		kubeclient: kubeclient,
	}
}

func (f *FakeInstrumentationFactory) CreateEbpfInstrumentation(ctx context.Context, pid int, serviceName string, podWorkload *workload.PodWorkload, containerName string, podName string, loadedIndicator chan struct{}) (*FakeEbpfSdk, error) {
	return &FakeEbpfSdk{
		loadedIndicator: loadedIndicator,
		pid:			 pid,
	}, nil
}

func newFakeDirector(ctx context.Context, client client.Client) ebpf.Director {
	dir := ebpf.NewEbpfDirector(ctx, client, client.Scheme(), common.GoProgrammingLanguage, NewFakeInstrumentationFactory(client))
	return dir
}

func getInstrumentationInstance(client client.Client, pod types.NamespacedName, pid int) *odigosv1.InstrumentationInstance {
	var instrumentationInstance odigosv1.InstrumentationInstance
	err := client.Get(context.Background(), types.NamespacedName{
		Name:      instrumentation_instance.InstrumentationInstanceName(pod.Name, pid),
		Namespace: pod.Namespace,
	}, &instrumentationInstance)
	if err != nil {
		return nil
	}
	return &instrumentationInstance
}

func assertHealthyInstrumentationInstance(t *testing.T, client client.Client, pod types.NamespacedName, pid int, healthy bool) bool {
	// healthy instrumentation instance is created
	var instance *odigosv1.InstrumentationInstance
	assert.Eventually(t, func() bool {
		instance = getInstrumentationInstance(client, pod, pid)
		return instance != nil
	}, 1*time.Second, 5*time.Millisecond)

	if instance == nil {
		return false
	}
	if healthy {
		return assert.True(t, *instance.Status.Healthy)
	}
	return assert.False(t, *instance.Status.Healthy)
}

func assertInstrumentationInstanceDeleted(t *testing.T, client client.Client, pod types.NamespacedName, pid int) bool {
	// instrumentation instance is deleted
	return assert.Eventually(t, func() bool {
		return getInstrumentationInstance(client, pod, pid) == nil
	}, 1*time.Second, 5*time.Millisecond)
}

func TestSingleInstrumentation(t *testing.T) {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	odigosv1.AddToScheme(scheme)

	workload := &workload.PodWorkload{
		Name:      "test-workload",
		Namespace: "default",
		Kind:      "Deployment",
	}
	pod_id := types.NamespacedName{Name: "test", Namespace: "default"}
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod_id.Name,
			Namespace: pod_id.Namespace,
		},
	}

	client := fake.
		NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&odigosv1.InstrumentationInstance{}).
		WithRuntimeObjects(&pod).
		Build()

	origIsProcessExists := ebpf.IsProcessExists
	ebpf.IsProcessExists = func(pid int) bool {
		return true
	}
	t.Cleanup(func() { ebpf.IsProcessExists = origIsProcessExists })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := newFakeDirector(ctx, client).(*ebpf.EbpfDirector[*FakeEbpfSdk])
	err := dir.Instrument(ctx, 1, pod_id, workload, "test-app", "test-container")
	assert.NoError(t, err)

	if !assertHealthyInstrumentationInstance(t, client, pod_id, 1, true) {
		t.FailNow()
	}

	// The instrumented process is tracked by the director
	insts := dir.GetWorkloadInstrumentations(workload)
	assert.Len(t, insts, 1)
	inst := insts[0]
	assert.True(t, inst.running)
	assert.False(t, inst.closed)

	// cleanup
	dir.Cleanup(pod_id)
	// the instrumentation instance is deleted
	if !assertInstrumentationInstanceDeleted(t, client, pod_id, 1) {
		t.FailNow()
	}

	insts = dir.GetWorkloadInstrumentations(workload)
	if !assert.Len(t, insts, 0) {
		t.FailNow()
	}

	// The instrumentation is stopped
	assert.True(t, inst.closed)
}

func TestInstrumentNotExistingProcess(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	odigosv1.AddToScheme(scheme)

	workload := &workload.PodWorkload{
		Name:      "test-workload",
		Namespace: "default",
		Kind:      "Deployment",
	}
	pod_id := types.NamespacedName{Name: "test", Namespace: "default"}
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod_id.Name,
			Namespace: pod_id.Namespace,
		},
	}

	client := fake.
		NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&odigosv1.InstrumentationInstance{}).
		WithRuntimeObjects(&pod).
		Build()

	origIsProcessExists := ebpf.IsProcessExists
	ebpf.IsProcessExists = func(pid int) bool {
		return true
	}
	t.Cleanup(func() { ebpf.IsProcessExists = origIsProcessExists })

	// setup the cleanup interval to be very short for the test to be responsive
	origCleanupInterval := ebpf.CleanupInterval
	ebpf.CleanupInterval = 10 * time.Millisecond
	t.Cleanup(func() { ebpf.CleanupInterval = origCleanupInterval })

	dir := newFakeDirector(ctx, client).(*ebpf.EbpfDirector[*FakeEbpfSdk])
	err := dir.Instrument(ctx, 1, pod_id, workload, "test-app", "test-container")
	assert.NoError(t, err)

	// healthy instrumentation instance is created
	if !assertHealthyInstrumentationInstance(t, client, pod_id, 1, true) {
		t.FailNow()
	}

	// The instrumented process is tracked by the director
	insts := dir.GetWorkloadInstrumentations(workload)
	assert.Len(t, insts, 1)
	inst := insts[0]
	assert.True(t, inst.running)
	assert.False(t, inst.closed)

	// "kill" the process
	ebpf.IsProcessExists = func(pid int) bool {
		return false
	}
	// the instrumentation instance is deleted
	if !assertInstrumentationInstanceDeleted(t, client, pod_id, 1) {
		t.FailNow()
	}
	// the director stopped tracking the instrumentation
	insts = dir.GetWorkloadInstrumentations(workload)
	if !assert.Len(t, insts, 0) {
		t.FailNow()
	}
	// The instrumentation is stopped
	assert.True(t, inst.closed)
}

func TestMultiplePodsInstrumentation(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	odigosv1.AddToScheme(scheme)
	numOfPods := 100

	workload := &workload.PodWorkload{
		Name:      "test-workload",
		Namespace: "default",
		Kind:      "Deployment",
	}

	podList := corev1.PodList{}
	pod_ids := make([]types.NamespacedName, numOfPods)
	for i := 0; i < numOfPods; i++ {
		pod_id := types.NamespacedName{Name: fmt.Sprintf("test-%d", i+1), Namespace: "default"}
		pod_ids[i] = pod_id
		podList.Items = append(podList.Items, corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod_id.Name,
				Namespace: pod_id.Namespace,
			},
		})
	}

	client := fake.
		NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&odigosv1.InstrumentationInstance{}).
		WithLists(&podList).
		Build()

	origIsProcessExists := ebpf.IsProcessExists
	ebpf.IsProcessExists = func(pid int) bool {
		return true
	}
	t.Cleanup(func() { ebpf.IsProcessExists = origIsProcessExists })

	dir := newFakeDirector(ctx, client).(*ebpf.EbpfDirector[*FakeEbpfSdk])
	for i := 0; i < numOfPods; i++ {
		err := dir.Instrument(ctx, i+1, pod_ids[i], workload, "test-app", "test-container")
		assert.NoError(t, err)
	}

	for i := 0; i < numOfPods; i++ {
		if !assertHealthyInstrumentationInstance(t, client, pod_ids[i], i+1, true) {
			t.FailNow()
		}
	}

	// The instrumented processes are tracked by the director
	insts := dir.GetWorkloadInstrumentations(workload)
	assert.Len(t, insts, numOfPods)
	// all the instrumentations are running
	for i := 0; i < numOfPods; i++ {
		inst := insts[i]
		assert.True(t, inst.running)
		assert.False(t, inst.closed)
	}

	// sort the insts slice by the pid so we can use it for deterministic checks
	sort.Slice(insts, func(i, j int) bool {
		return insts[i].pid < insts[j].pid
	})

	// cleanup all the pods accept the last one
	for i := 0; i < (numOfPods - 1); i++ {
		dir.Cleanup(pod_ids[i])
	}

	// the instrumentation instances are deleted
	for i := 0; i < numOfPods - 1; i++ {
		if !assertInstrumentationInstanceDeleted(t, client, pod_ids[i], i+1) {
			t.FailNow()
		}
	}
	// accept the last one
	if !assertHealthyInstrumentationInstance(t, client, pod_ids[numOfPods-1], numOfPods, true) {
		t.FailNow()
	}

	// closed is call for each instrumentation
	for i := 0; i < numOfPods - 1; i++ {
		if !assert.Eventually(t, func() bool { return insts[i].closed }, 1*time.Second, 5*time.Millisecond) {
			t.Logf("instrumentation %d is not closed", i)
			t.FailNow()
		}
	}
	// the last one is still running
	assert.False(t, insts[numOfPods-1].closed)

	// the director stopped tracking the instrumentations accept the last one
	insts = dir.GetWorkloadInstrumentations(workload)
	if !assert.Len(t, insts, 1) {
		t.FailNow()
	}
	// The last instrumentation is the one returned
	assert.Equal(t, insts[0].pid, numOfPods)
}
