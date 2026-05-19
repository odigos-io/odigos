package clustercollector

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetDesiredDeploymentEnablesProfilesFeatureGateForProfilesDestination(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, odigosv1.AddToScheme(scheme))

	effectiveConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosEffectiveConfigName,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Data: map[string]string{},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(effectiveConfig).Build()

	dests := &odigosv1.DestinationList{Items: []odigosv1.Destination{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "profiles-destination",
				Namespace: consts.DefaultOdigosNamespace,
			},
			Spec: odigosv1.DestinationSpec{
				Type:    common.PyroscopeDestinationType,
				Signals: []common.ObservabilitySignal{common.ProfilesObservabilitySignal},
				Data: map[string]string{
					"PYROSCOPE_URL": "pyroscope:4040",
				},
			},
		},
	}}
	gateway := &odigosv1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorConfigMapName,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role: odigosv1.CollectorsGroupRoleClusterGateway,
			ResourcesSettings: odigosv1.CollectorsGroupResourcesSettings{
				MemoryRequestMiB:     256,
				MemoryLimitMiB:       512,
				CpuRequestMillicores: 250,
				CpuLimitMillicores:   500,
			},
		},
	}

	deployment, err := getDesiredDeployment(context.Background(), client, dests, "hash", gateway, scheme, "test-version", nil)
	require.NoError(t, err)
	require.Contains(t, deployment.Spec.Template.Spec.Containers[0].Args, "--feature-gates=service.profilesSupport")
}
