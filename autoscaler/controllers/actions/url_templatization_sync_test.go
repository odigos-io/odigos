package actions

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	urltemplateactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSyncUrlTemplatizationProcessorDeletesStaleActionOwnedProcessors(t *testing.T) {
	ctx := context.Background()
	ns := consts.DefaultOdigosNamespace
	t.Setenv(consts.CurrentNamespaceEnvVar, ns)

	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	actionUID := types.UID("url-action-uid")
	urlAction := &v1alpha1.Action{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Action",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "url-action",
			Namespace: ns,
			UID:       actionUID,
		},
		Spec: v1alpha1.ActionSpec{
			ActionName: "URL Templatization",
			Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
			URLTemplatization: &urltemplateactions.URLTemplatizationConfig{
				Rules: []urltemplateactions.UrlTemplatizationRule{
					{Templates: []string{"/users/{id}"}},
				},
			},
		},
	}

	staleActionOwnedProcessor := &v1alpha1.Processor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Processor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      urlAction.Name,
			Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "odigos.io/v1alpha1",
					Kind:       "Action",
					Name:       urlAction.Name,
					UID:        actionUID,
				},
			},
		},
		Spec: v1alpha1.ProcessorSpec{
			Type:           consts.OdigosURLTemplateProcessorType,
			Signals:        []common.ObservabilitySignal{common.TracesObservabilitySignal},
			CollectorRoles: []v1alpha1.CollectorsGroupRole{v1alpha1.CollectorsGroupRoleClusterGateway},
		},
	}

	manualUrlProcessor := &v1alpha1.Processor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Processor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "manual-url-processor",
			Namespace: ns,
		},
		Spec: v1alpha1.ProcessorSpec{
			Type:           consts.OdigosURLTemplateProcessorType,
			Signals:        []common.ObservabilitySignal{common.TracesObservabilitySignal},
			CollectorRoles: []v1alpha1.CollectorsGroupRole{v1alpha1.CollectorsGroupRoleClusterGateway},
		},
	}

	otherActionOwnedProcessor := &v1alpha1.Processor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Processor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delete-attribute",
			Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "odigos.io/v1alpha1",
					Kind:       "Action",
					Name:       "delete-attribute",
					UID:        types.UID("delete-attribute-uid"),
				},
			},
		},
		Spec: v1alpha1.ProcessorSpec{
			Type:           "transform",
			Signals:        []common.ObservabilitySignal{common.TracesObservabilitySignal},
			CollectorRoles: []v1alpha1.CollectorsGroupRole{v1alpha1.CollectorsGroupRoleClusterGateway},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(urlAction, staleActionOwnedProcessor, manualUrlProcessor, otherActionOwnedProcessor).
		Build()

	require.NoError(t, SyncUrlTemplatizationProcessor(ctx, k8sClient, URLTemplatizationSyncApplyFull))

	var deleted v1alpha1.Processor
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: staleActionOwnedProcessor.Name}, &deleted)
	require.True(t, apierrors.IsNotFound(err))

	var shared v1alpha1.Processor
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: consts.URLTemplatizationProcessorName}, &shared))
	require.Equal(t, consts.OdigosURLTemplateProcessorType, shared.Spec.Type)

	var manual v1alpha1.Processor
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: manualUrlProcessor.Name}, &manual))

	var other v1alpha1.Processor
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: otherActionOwnedProcessor.Name}, &other))
}
