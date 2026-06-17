package predicate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

func TestMissingInstrumentationConfigPredicate_Update(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, odigosv1.AddToScheme(scheme))

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
	}
	namespaceSource := &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "default",
			Labels: map[string]string{
				k8sconsts.WorkloadNameLabel:      "default",
				k8sconsts.WorkloadNamespaceLabel: "default",
				k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindNamespace),
			},
		},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Name:      "default",
				Namespace: "default",
				Kind:      k8sconsts.WorkloadKindNamespace,
			},
		},
	}

	ic := &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workload.CalculateWorkloadRuntimeObjectName(deployment.Name, "Deployment"),
			Namespace: deployment.Namespace,
		},
	}

	t.Run("allows update when instrumentation config is missing and source applies", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns, deployment, namespaceSource).Build()
		p := predicate.MissingInstrumentationConfigPredicate{Client: c}

		assert.True(t, p.Update(event.UpdateEvent{ObjectNew: deployment}))
	})

	t.Run("ignores update when instrumentation config exists", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns, deployment, namespaceSource, ic).Build()
		p := predicate.MissingInstrumentationConfigPredicate{Client: c}

		assert.False(t, p.Update(event.UpdateEvent{ObjectNew: deployment}))
	})

	t.Run("ignores update when instrumentation config is missing but no source applies", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns, deployment).Build()
		p := predicate.MissingInstrumentationConfigPredicate{Client: c}

		assert.False(t, p.Update(event.UpdateEvent{ObjectNew: deployment}))
	})
}

func TestWorkloadCreateOrMissingInstrumentationConfig(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, odigosv1.AddToScheme(scheme))

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
	}
	namespaceSource := &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "default",
			Labels: map[string]string{
				k8sconsts.WorkloadNameLabel:      "default",
				k8sconsts.WorkloadNamespaceLabel: "default",
				k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindNamespace),
			},
		},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Name:      "default",
				Namespace: "default",
				Kind:      k8sconsts.WorkloadKindNamespace,
			},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(deployment, namespaceSource).Build()
	p := predicate.WorkloadCreateOrMissingInstrumentationConfig(c)

	assert.True(t, (&predicate.CreationPredicate{}).Create(event.CreateEvent{Object: deployment}))
	assert.True(t, p.Update(event.UpdateEvent{ObjectNew: deployment}))
}
