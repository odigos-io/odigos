package actions

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
)

func newActionsUnitTestClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, odigosv1.AddToScheme(s))
	return fake.NewClientBuilder().WithScheme(s).WithObjects(objs...).Build()
}

func newActionsUnitTestClientWithListError(t *testing.T, err error) client.Client {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, odigosv1.AddToScheme(s))
	return fake.NewClientBuilder().WithScheme(s).WithInterceptorFuncs(interceptor.Funcs{
		List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			return err
		},
	}).Build()
}

func newURLTemplatizationAction(name, namespace string, disabled bool) *odigosv1.Action {
	return &odigosv1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: odigosv1.ActionSpec{
			Disabled: disabled,
			Signals:  []common.ObservabilitySignal{common.TracesObservabilitySignal},
			URLTemplatization: &odigosactions.URLTemplatizationConfig{
				TemplatizationRulesGroups: []odigosactions.UrlTemplatizationRulesGroup{
					{
						TemplatizationRules: []odigosactions.URLTemplatizationRule{
							{Template: "/users/{id}"},
						},
					},
				},
			},
		},
	}
}

func newProcessorEventObj(name, namespace string) *odigosv1.Processor {
	return &odigosv1.Processor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func TestURLTemplatizationNamespaceSyncKeyIsInvalidK8sObjectName(t *testing.T) {
	errs := validation.IsDNS1123Subdomain(urlTemplatizationNamespaceSyncKey)
	require.NotEmpty(t, errs)
}

func TestMapUrlTemplatizationProcessorToActionRequests_NonTargetProcessorReturnsNil(t *testing.T) {
	cl := newActionsUnitTestClient(t)
	reqs := mapUrlTemplatizationProcessorToActionRequests(context.Background(), cl, newProcessorEventObj("not-url-templatization", "default"))
	assert.Nil(t, reqs)
}

func TestMapUrlTemplatizationProcessorToActionRequests_MapsOnlyActiveURLTemplatizationActions(t *testing.T) {
	// Mapping always enqueues the synthetic namespace-level key so one reconcile runs per namespace.
	cl := newActionsUnitTestClient(t)

	reqs := mapUrlTemplatizationProcessorToActionRequests(context.Background(), cl, newProcessorEventObj(commonconsts.URLTemplatizationProcessorName, "default"))

	assert.Equal(t, []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Namespace: "default",
				Name:      urlTemplatizationNamespaceSyncKey,
			},
		},
	}, reqs)
}

func TestMapUrlTemplatizationProcessorToActionRequests_NoActiveActionsEnqueuesSyntheticKey(t *testing.T) {
	cl := newActionsUnitTestClient(t)

	reqs := mapUrlTemplatizationProcessorToActionRequests(context.Background(), cl, newProcessorEventObj(commonconsts.URLTemplatizationProcessorName, "default"))

	require.Len(t, reqs, 1)
	assert.Equal(t, types.NamespacedName{
		Namespace: "default",
		Name:      urlTemplatizationNamespaceSyncKey,
	}, reqs[0].NamespacedName)
}

func TestMapUrlTemplatizationProcessorToActionRequests_ListErrorEnqueuesSyntheticKey(t *testing.T) {
	cl := newActionsUnitTestClientWithListError(t, errors.New("list failed"))

	reqs := mapUrlTemplatizationProcessorToActionRequests(context.Background(), cl, newProcessorEventObj(commonconsts.URLTemplatizationProcessorName, "default"))

	require.Len(t, reqs, 1)
	assert.Equal(t, types.NamespacedName{
		Namespace: "default",
		Name:      urlTemplatizationNamespaceSyncKey,
	}, reqs[0].NamespacedName)
}

func TestActionReconciler_ReconcileSyntheticKey_DeletesOrphanedSharedProcessor(t *testing.T) {
	processor := newProcessorEventObj(commonconsts.URLTemplatizationProcessorName, "default")
	cl := newActionsUnitTestClient(t, processor)
	r := &ActionReconciler{Client: cl}

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      urlTemplatizationNamespaceSyncKey,
		},
	})
	require.NoError(t, err)

	getErr := cl.Get(context.Background(), types.NamespacedName{
		Namespace: "default",
		Name:      commonconsts.URLTemplatizationProcessorName,
	}, &odigosv1.Processor{})
	assert.True(t, apierrors.IsNotFound(getErr))
}

func TestActionReconciler_ReconcileSyntheticKey_CreatesSharedProcessorForActiveActions(t *testing.T) {
	activeAction := newURLTemplatizationAction("url-template", "default", false)
	// Reconciler lists Actions by label; set the label so sync finds this action and creates the Processor.
	if activeAction.Labels == nil {
		activeAction.Labels = make(map[string]string)
	}
	activeAction.Labels[urlTemplatizationLabelKey] = urlTemplatizationLabelValue
	cl := newActionsUnitTestClient(t, activeAction)
	r := &ActionReconciler{Client: cl}

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      urlTemplatizationNamespaceSyncKey,
		},
	})
	require.NoError(t, err)

	processor := &odigosv1.Processor{}
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{
		Namespace: "default",
		Name:      commonconsts.URLTemplatizationProcessorName,
	}, processor))

	assert.Equal(t, commonconsts.URLTemplatizationProcessorName, processor.Name)
	assert.Equal(t, "odigosurltemplate", processor.Spec.Type)
	require.NotNil(t, processor.Spec.ProcessorConfig.Raw)

	var cfg map[string]interface{}
	require.NoError(t, json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &cfg))
	assert.Equal(t, commonconsts.OdigosConfigK8sExtensionType, cfg["workload_config_extension"])
}
