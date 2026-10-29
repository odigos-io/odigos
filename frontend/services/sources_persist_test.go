package services

import (
	"context"
	"fmt"
	"sync"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

const (
	testNs     = "shop"
	testStream = "default"
)

// sourceRecorder keeps the state of every Source after each create or patch, in the order they were applied.
type sourceRecorder struct {
	mu        sync.Mutex
	client    *odigosfake.Clientset
	snapshots []sourceSnapshot
	creates   []string
	generated int
}

type sourceSnapshot struct {
	verb     string
	workload string
	disabled bool
	labels   map[string]string
}

func workloadID(kind k8sconsts.WorkloadKind, name string) string {
	return fmt.Sprintf("%s/%s", kind, name)
}

// setupFakePersistClient points kube.DefaultClient at a fake odigos clientset holding sources.
// Created sources get a name and the workload labels that the API server webhook would set.
func setupFakePersistClient(t *testing.T, sources ...*v1alpha1.Source) *sourceRecorder {
	t.Helper()

	objects := []runtime.Object{}
	for _, source := range sources {
		objects = append(objects, source)
	}
	recorder := &sourceRecorder{client: odigosfake.NewSimpleClientset(objects...)} //nolint:staticcheck // NewClientset can't type Source objects without generated apply configurations

	recorder.client.PrependReactor("*", "sources", func(action k8stesting.Action) (bool, runtime.Object, error) {
		verb := action.GetVerb()
		if verb != "create" && verb != "patch" {
			return false, nil, nil
		}
		if verb == "create" {
			source := action.(k8stesting.CreateAction).GetObject().(*v1alpha1.Source)
			recorder.mu.Lock()
			recorder.generated++
			source.Name = fmt.Sprintf("%s%d", source.GenerateName, recorder.generated)
			recorder.mu.Unlock()
			if source.Labels == nil {
				source.Labels = map[string]string{}
			}
			source.Labels[k8sconsts.WorkloadNamespaceLabel] = source.Spec.Workload.Namespace
			source.Labels[k8sconsts.WorkloadNameLabel] = source.Spec.Workload.Name
			source.Labels[k8sconsts.WorkloadKindLabel] = string(source.Spec.Workload.Kind)
		}
		handled, obj, err := k8stesting.ObjectReaction(recorder.client.Tracker())(action)
		if err == nil {
			source := obj.(*v1alpha1.Source)
			id := workloadID(source.Spec.Workload.Kind, source.Spec.Workload.Name)
			labels := map[string]string{}
			for key, value := range source.Labels {
				labels[key] = value
			}
			recorder.mu.Lock()
			recorder.snapshots = append(recorder.snapshots, sourceSnapshot{verb: verb, workload: id, disabled: source.Spec.DisableInstrumentation, labels: labels})
			if verb == "create" {
				recorder.creates = append(recorder.creates, id)
			}
			recorder.mu.Unlock()
		}
		return handled, obj, err
	})

	previous := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{OdigosClient: recorder.client.OdigosV1alpha1()})
	t.Cleanup(func() { kube.SetDefaultClient(previous) })

	return recorder
}

func existingSource(name string, kind k8sconsts.WorkloadKind, workload string, disabled bool, streams map[string]string) *v1alpha1.Source {
	labels := map[string]string{
		k8sconsts.WorkloadNamespaceLabel: testNs,
		k8sconsts.WorkloadNameLabel:      workload,
		k8sconsts.WorkloadKindLabel:      string(kind),
	}
	for stream, value := range streams {
		labels[k8sconsts.SourceDataStreamLabelPrefix+stream] = value
	}
	return &v1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNs, Labels: labels},
		Spec: v1alpha1.SourceSpec{
			Workload:               k8sconsts.PodWorkload{Namespace: testNs, Name: workload, Kind: kind},
			DisableInstrumentation: disabled,
		},
	}
}

func persistWorkload(name string, selected bool) *model.PersistNamespaceSourceInput {
	return &model.PersistNamespaceSourceInput{Namespace: testNs, Name: name, Kind: WorkloadKindDeployment, Selected: selected, CurrentStreamName: testStream}
}

func persistNamespace(selected bool) *model.PersistNamespaceSourceInput {
	return &model.PersistNamespaceSourceInput{Namespace: testNs, Name: testNs, Kind: WorkloadKindNamespace, Selected: selected, CurrentStreamName: testStream}
}

func getSource(t *testing.T, kind model.K8sResourceKind, name string) *v1alpha1.Source {
	t.Helper()
	source, err := GetSourceCRD(context.Background(), testNs, name, kind)
	require.NoError(t, err)
	return source
}

func streamLabel(source *v1alpha1.Source, stream string) string {
	return source.Labels[k8sconsts.SourceDataStreamLabelPrefix+stream]
}

func TestSyncWorkloadsInNamespace_ExcludedWorkloadCreatedBeforeNamespace(t *testing.T) {
	recorder := setupFakePersistClient(t)

	err := SyncWorkloadsInNamespace(context.Background(), []*model.PersistNamespaceSourceInput{
		persistNamespace(true),
		persistWorkload("cart", false),
		persistWorkload("checkout", true),
	})
	require.NoError(t, err)

	nsID := workloadID(k8sconsts.WorkloadKindNamespace, testNs)
	require.Len(t, recorder.creates, 3)
	assert.Equal(t, nsID, recorder.creates[2], "the namespace source must be created after every workload source")

	nsSource := getSource(t, WorkloadKindNamespace, testNs)
	assert.False(t, nsSource.Spec.DisableInstrumentation)
	assert.Equal(t, "true", streamLabel(nsSource, testStream))

	excluded := getSource(t, WorkloadKindDeployment, "cart")
	assert.True(t, excluded.Spec.DisableInstrumentation)
	assert.Equal(t, "false", streamLabel(excluded, testStream))

	selected := getSource(t, WorkloadKindDeployment, "checkout")
	assert.False(t, selected.Spec.DisableInstrumentation)
}

func TestSyncWorkloadsInNamespace_ExcludedWorkloadIsNeverEnabled(t *testing.T) {
	recorder := setupFakePersistClient(t)

	err := SyncWorkloadsInNamespace(context.Background(), []*model.PersistNamespaceSourceInput{
		persistWorkload("cart", false),
		persistNamespace(true),
	})
	require.NoError(t, err)

	cartID := workloadID(k8sconsts.WorkloadKindDeployment, "cart")
	seen := 0
	for _, snapshot := range recorder.snapshots {
		if snapshot.workload != cartID {
			continue
		}
		seen++
		assert.True(t, snapshot.disabled, "%s of the excluded source must keep instrumentation disabled", snapshot.verb)
		assert.Equal(t, "false", snapshot.labels[k8sconsts.SourceDataStreamLabelPrefix+testStream])
	}
	assert.Equal(t, 1, seen, "the excluded source must be created once, with no follow-up patches")
}

func TestSyncWorkloadsInNamespace_ExistingNamespaceSourceCreatesDisabledSource(t *testing.T) {
	recorder := setupFakePersistClient(t, existingSource("ns-source", k8sconsts.WorkloadKindNamespace, testNs, false, map[string]string{testStream: "true"}))

	err := SyncWorkloadsInNamespace(context.Background(), []*model.PersistNamespaceSourceInput{persistWorkload("cart", false)})
	require.NoError(t, err)

	require.Len(t, recorder.snapshots, 1)
	assert.Equal(t, "create", recorder.snapshots[0].verb)
	assert.True(t, recorder.snapshots[0].disabled)

	excluded := getSource(t, WorkloadKindDeployment, "cart")
	assert.True(t, excluded.Spec.DisableInstrumentation)
	assert.Equal(t, "false", streamLabel(excluded, testStream))
}

func TestSyncWorkloadsInNamespace_AlreadyExcludedWorkloadIsNotReEnabled(t *testing.T) {
	recorder := setupFakePersistClient(t,
		existingSource("ns-source", k8sconsts.WorkloadKindNamespace, testNs, false, map[string]string{testStream: "true"}),
		existingSource("cart-source", k8sconsts.WorkloadKindDeployment, "cart", true, map[string]string{testStream: "false"}),
	)

	err := SyncWorkloadsInNamespace(context.Background(), []*model.PersistNamespaceSourceInput{
		persistWorkload("cart", false),
		persistNamespace(true),
	})
	require.NoError(t, err)

	for _, snapshot := range recorder.snapshots {
		if snapshot.workload == workloadID(k8sconsts.WorkloadKindDeployment, "cart") {
			assert.True(t, snapshot.disabled, "%s of the excluded source must keep instrumentation disabled", snapshot.verb)
		}
	}
	excluded := getSource(t, WorkloadKindDeployment, "cart")
	assert.True(t, excluded.Spec.DisableInstrumentation)
	assert.Equal(t, "false", streamLabel(excluded, testStream))
}

func TestSyncWorkloadsInNamespace_ReselectingReEnables(t *testing.T) {
	setupFakePersistClient(t,
		existingSource("ns-source", k8sconsts.WorkloadKindNamespace, testNs, false, map[string]string{testStream: "true"}),
		existingSource("cart-source", k8sconsts.WorkloadKindDeployment, "cart", true, map[string]string{testStream: "false"}),
	)

	err := SyncWorkloadsInNamespace(context.Background(), []*model.PersistNamespaceSourceInput{persistWorkload("cart", true)})
	require.NoError(t, err)

	source := getSource(t, WorkloadKindDeployment, "cart")
	assert.False(t, source.Spec.DisableInstrumentation)
	assert.Equal(t, "true", streamLabel(source, testStream))
}

func TestSyncWorkloadsInNamespace_UnselectedWithoutNamespaceLeavesNoSource(t *testing.T) {
	recorder := setupFakePersistClient(t, existingSource("api-source", k8sconsts.WorkloadKindDeployment, "api", false, map[string]string{testStream: "true"}))

	err := SyncWorkloadsInNamespace(context.Background(), []*model.PersistNamespaceSourceInput{
		persistWorkload("api", false),
		persistWorkload("cart", false),
	})
	require.NoError(t, err)

	assert.Empty(t, recorder.creates, "no source is created for a workload without a namespace source")
	list, err := recorder.client.OdigosV1alpha1().Sources(testNs).List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, list.Items)
}

func TestSyncWorkloadsInNamespace_NamespaceTurnedOffDeletesWorkloadSources(t *testing.T) {
	recorder := setupFakePersistClient(t,
		existingSource("ns-source", k8sconsts.WorkloadKindNamespace, testNs, false, map[string]string{testStream: "true"}),
		existingSource("cart-source", k8sconsts.WorkloadKindDeployment, "cart", true, map[string]string{testStream: "false"}),
	)

	err := SyncWorkloadsInNamespace(context.Background(), []*model.PersistNamespaceSourceInput{
		persistWorkload("cart", false),
		persistWorkload("checkout", true),
		persistNamespace(false),
	})
	require.NoError(t, err)

	list, err := recorder.client.OdigosV1alpha1().Sources(testNs).List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, list.Items, 1, "only the kept workload has a source once the namespace is turned off")
	assert.Equal(t, "checkout", list.Items[0].Spec.Workload.Name)
	assert.False(t, list.Items[0].Spec.DisableInstrumentation)
}

func TestSyncWorkloadsInNamespace_ExcludedFromOneOfManyStreams(t *testing.T) {
	const otherStream = "payments"
	recorder := setupFakePersistClient(t, existingSource("ns-source", k8sconsts.WorkloadKindNamespace, testNs, false, map[string]string{testStream: "true", otherStream: "true"}))

	err := SyncWorkloadsInNamespace(context.Background(), []*model.PersistNamespaceSourceInput{persistWorkload("cart", false)})
	require.NoError(t, err)

	require.Len(t, recorder.snapshots, 1, "the source is created in its final state")
	excluded := getSource(t, WorkloadKindDeployment, "cart")
	assert.False(t, excluded.Spec.DisableInstrumentation, "the workload stays instrumented for the other stream")
	assert.Equal(t, "false", streamLabel(excluded, testStream))

	streams := []string{}
	for _, name := range ExtractDataStreamsFromSource(excluded, getSource(t, WorkloadKindNamespace, testNs)) {
		streams = append(streams, *name)
	}
	assert.Equal(t, []string{otherStream}, streams)
}

func TestSyncWorkloadsInNamespace_ExistingSourceInManyStreamsKeepsOtherStreams(t *testing.T) {
	const otherStream = "payments"
	setupFakePersistClient(t, existingSource("api-source", k8sconsts.WorkloadKindDeployment, "api", false, map[string]string{testStream: "true", otherStream: "true"}))

	err := SyncWorkloadsInNamespace(context.Background(), []*model.PersistNamespaceSourceInput{persistWorkload("api", false)})
	require.NoError(t, err)

	source := getSource(t, WorkloadKindDeployment, "api")
	assert.False(t, source.Spec.DisableInstrumentation)
	_, hasCurrent := source.Labels[k8sconsts.SourceDataStreamLabelPrefix+testStream]
	assert.False(t, hasCurrent, "the current stream label is removed")
	assert.Equal(t, "true", streamLabel(source, otherStream))
}
