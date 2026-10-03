package podsmanifestinjectionstatus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	podsManifestInjection "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Changing the cluster-wide rollout setting changes the reason reported for every workload, so
// this reconciler re-syncs all of them. One instrumentation config it cannot parse must not stop
// the rest of the cluster from being updated.
func TestEffectiveConfigReconcilerSyncsEveryWorkload(t *testing.T) {
	ctx := newInjectionTestContext()
	c := newInjectionTestClient(t,
		newEffectiveConfigMap("rollout:\n  automaticRolloutDisabled: true\n"),
		newInjectionDeployment("web", map[string]string{"app": "web"}),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
		withPodLabel(newInjectionPod(injectionNamespace, "web-1", ""), "app", "web"),
		newInjectionDeployment("api", map[string]string{"app": "api"}),
		newInjectionInstrumentationConfig("api", k8sconsts.WorkloadKindDeployment),
		withPodLabel(newInjectionPod(injectionNamespace, "api-1", ""), "app", "api"),
		&odigosv1.InstrumentationConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "bogus-workload", Namespace: injectionNamespace},
		},
	)
	r := &EffectiveConfigReconciler{Client: c}

	_, err := r.Reconcile(ctx, ctrl.Request{})
	require.Error(t, err, "the unparsable instrumentation config name must be reported")

	for _, name := range []string{"web", "api"} {
		ic := syncedInstrumentationConfig(t, ctx, c, name, k8sconsts.WorkloadKindDeployment)
		assert.Equal(t,
			string(podsManifestInjection.PodsManifestInjectionReasonRestartRequiredAutoRolloutDisabled_Enabled),
			injectionConditionReason(ic), "workload %s", name)
	}
}

// A workload that fails to sync has to surface, otherwise a rollout configuration change silently
// leaves part of the cluster reporting the previous reason.
func TestEffectiveConfigReconcilerReportsSyncFailures(t *testing.T) {
	ctx := newInjectionTestContext()
	listErr := apierrors.NewInternalError(errors.New("list pods failed"))
	c := interceptor.NewClient(newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionDeployment("web", map[string]string{"app": "web"}),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
	), interceptor.Funcs{
		List: func(ctx context.Context, cl client.WithWatch, list client.ObjectList,
			opts ...client.ListOption) error {
			if _, ok := list.(*corev1.PodList); ok {
				return listErr
			}
			return cl.List(ctx, list, opts...)
		},
	})
	r := &EffectiveConfigReconciler{Client: c}

	_, err := r.Reconcile(ctx, ctrl.Request{})
	assert.ErrorIs(t, err, listErr)
}

// The effective config is created asynchronously, so before it exists the reconciler has to back
// off rather than report an error.
func TestEffectiveConfigReconcilerRequeuesWithoutTheEffectiveConfig(t *testing.T) {
	ctx := newInjectionTestContext()
	c := newInjectionTestClient(t,
		newInjectionDeployment("web", map[string]string{"app": "web"}),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
	)
	r := &EffectiveConfigReconciler{Client: c}

	result, err := r.Reconcile(ctx, ctrl.Request{})
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second}, result)
}
