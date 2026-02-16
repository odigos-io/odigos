package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = odigosv1alpha1.AddToScheme(scheme)
	return scheme
}

func newFakeClient(scheme *runtime.Scheme, objects []client.Object, statusSubresources ...client.Object) client.WithWatch {
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(statusSubresources...).
		Build()
}

func newWorkloadSource(namespace, workloadName, kind string) *odigosv1alpha1.Source {
	return &odigosv1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workloadName + "-source",
			Namespace: namespace,
			Labels: map[string]string{
				k8sconsts.WorkloadNameLabel:      workloadName,
				k8sconsts.WorkloadNamespaceLabel: namespace,
				k8sconsts.WorkloadKindLabel:      kind,
			},
		},
		Spec: odigosv1alpha1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Name:      workloadName,
				Namespace: namespace,
				Kind:      k8sconsts.WorkloadKind(kind),
			},
		},
	}
}

func TestRecoverFromRollback_Success(t *testing.T) {
	// Arrange: Source with no previous recovery timestamp
	scheme := newScheme()
	source := newWorkloadSource("test-ns", "my-app", "Deployment")
	fakeClient := newFakeClient(scheme, []client.Object{source})

	// Act
	err := RecoverFromRollback(context.Background(), fakeClient, "test-ns", "my-app", "Deployment")

	// Assert: RecoveredFromRollbackAt is set on the Source
	require.NoError(t, err)
	updatedSource := &odigosv1alpha1.Source{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKeyFromObject(source), updatedSource))
	assert.NotNil(t, updatedSource.Spec.RecoveredFromRollbackAt, "expected RecoveredFromRollbackAt to be set")
}

func TestRecoverFromRollback_AlreadySet(t *testing.T) {
	// Arrange: Source already has a recovery timestamp from 1 minute ago
	scheme := newScheme()
	source := newWorkloadSource("test-ns", "my-app", "Deployment")
	oldTime := metav1.NewTime(time.Now().Add(-time.Minute))
	source.Spec.RecoveredFromRollbackAt = &oldTime
	fakeClient := newFakeClient(scheme, []client.Object{source})

	// Act
	err := RecoverFromRollback(context.Background(), fakeClient, "test-ns", "my-app", "Deployment")

	// Assert: Timestamp is updated to a newer value
	require.NoError(t, err)
	updatedSource := &odigosv1alpha1.Source{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKeyFromObject(source), updatedSource))
	assert.NotNil(t, updatedSource.Spec.RecoveredFromRollbackAt, "expected RecoveredFromRollbackAt to be set")
	assert.True(t, updatedSource.Spec.RecoveredFromRollbackAt.After(oldTime.Time), "expected a newer timestamp")
}

func TestRecoverFromRollback_ReRecovery(t *testing.T) {
	// Arrange: Source with a previous recovery timestamp, simulating recover -> rollback -> recover again
	scheme := newScheme()
	source := newWorkloadSource("test-ns", "my-app", "Deployment")
	firstTimestamp := metav1.NewTime(time.Now().Add(-time.Minute))
	source.Spec.RecoveredFromRollbackAt = &firstTimestamp
	fakeClient := newFakeClient(scheme, []client.Object{source})
	ctx := context.Background()
	sourceKey := client.ObjectKeyFromObject(source)

	// Act
	require.NoError(t, RecoverFromRollback(ctx, fakeClient, "test-ns", "my-app", "Deployment"))

	// Assert: New timestamp is strictly after the first, signaling a fresh recovery to the instrumentor
	updatedSource := &odigosv1alpha1.Source{}
	require.NoError(t, fakeClient.Get(ctx, sourceKey, updatedSource))
	secondTimestamp := updatedSource.Spec.RecoveredFromRollbackAt
	require.NotNil(t, secondTimestamp, "expected second recovery timestamp to be set")
	assert.True(t, secondTimestamp.After(firstTimestamp.Time),
		"second recovery timestamp should be strictly after the first")
}

func TestRecoverFromRollback_SourceNotFound(t *testing.T) {
	// Arrange: No Source objects exist
	scheme := newScheme()
	fakeClient := newFakeClient(scheme, nil)

	// Act
	err := RecoverFromRollback(context.Background(), fakeClient, "test-ns", "my-app", "Deployment")

	// Assert: A new workload Source is created
	require.NoError(t, err)
	sourceList := &odigosv1alpha1.SourceList{}
	require.NoError(t, fakeClient.List(context.Background(), sourceList, client.InNamespace("test-ns")))
	require.Len(t, sourceList.Items, 1, "expected exactly one Source to be created")

	created := sourceList.Items[0]
	assert.Equal(t, "my-app", created.Name)
	assert.Equal(t, "test-ns", created.Namespace)
	assert.Equal(t, "my-app", created.Spec.Workload.Name)
	assert.Equal(t, "test-ns", created.Spec.Workload.Namespace)
	assert.Equal(t, k8sconsts.WorkloadKind("Deployment"), created.Spec.Workload.Kind)
}

func TestRecoverFromRollback_UpdateFailure(t *testing.T) {
	// Arrange: Source exists, but client.Update is intercepted to return an error
	scheme := newScheme()
	source := newWorkloadSource("test-ns", "my-app", "Deployment")
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(source).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				if _, ok := obj.(*odigosv1alpha1.Source); ok {
					return errors.New("simulated update error")
				}
				return c.Update(ctx, obj, opts...)
			},
		}).
		Build()

	// Act
	err := RecoverFromRollback(context.Background(), fakeClient, "test-ns", "my-app", "Deployment")

	// Assert: Error - update failed
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update Source with RecoveredFromRollbackAt")
}

func TestRecoverFromRollback_NamespaceSourceOnly_CreatesWorkloadSource(t *testing.T) {
	// Arrange: Only a namespace-level source exists, no workload-level source.
	// This simulates a workload instrumented via a namespace source that needs recovery.
	scheme := newScheme()
	nsSource := &odigosv1alpha1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ns",
			Namespace: "test-ns",
			Labels: map[string]string{
				k8sconsts.WorkloadNameLabel:      "test-ns",
				k8sconsts.WorkloadNamespaceLabel: "test-ns",
				k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindNamespace),
			},
		},
		Spec: odigosv1alpha1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Name:      "test-ns",
				Namespace: "test-ns",
				Kind:      k8sconsts.WorkloadKindNamespace,
			},
		},
	}
	fakeClient := newFakeClient(scheme, []client.Object{nsSource})
	ctx := context.Background()

	// Act
	err := RecoverFromRollback(ctx, fakeClient, "test-ns", "my-app", "Deployment")

	// Assert: No error, and a new workload-level Source was created with the correct spec
	require.NoError(t, err)

	sourceList := &odigosv1alpha1.SourceList{}
	require.NoError(t, fakeClient.List(ctx, sourceList, client.InNamespace("test-ns")))

	var workloadSource *odigosv1alpha1.Source
	for i := range sourceList.Items {
		if sourceList.Items[i].Name == "my-app" {
			workloadSource = &sourceList.Items[i]
			break
		}
	}
	require.NotNil(t, workloadSource, "expected a workload-level Source to be created")
	assert.Equal(t, "my-app", workloadSource.Spec.Workload.Name)
	assert.Equal(t, "test-ns", workloadSource.Spec.Workload.Namespace)
	assert.Equal(t, k8sconsts.WorkloadKind("Deployment"), workloadSource.Spec.Workload.Kind)

	// The namespace source should still exist untouched
	var namespaceSource *odigosv1alpha1.Source
	for i := range sourceList.Items {
		if sourceList.Items[i].Name == "test-ns" {
			namespaceSource = &sourceList.Items[i]
			break
		}
	}
	require.NotNil(t, namespaceSource, "expected the namespace source to still exist")
}
