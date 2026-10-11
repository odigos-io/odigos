package graph

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/loaders"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/graph/status"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// The rollback durations in the odigos configuration are free text: the settings UI
// writes them with no validation and helm/GitOps users set them directly. A value that
// time.ParseDuration rejects must not take down the autoRollback field for every
// instrumented workload in the cluster.
func TestAutoRollbackResolver_UnparsableRollbackDurationInConfig(t *testing.T) {
	const odigosNamespace = "odigos-system"

	effectiveConfig := func(configYaml string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      consts.OdigosEffectiveConfigName,
				Namespace: odigosNamespace,
			},
			Data: map[string]string{consts.OdigosConfigurationFileName: configYaml},
		}
	}

	instrumentedDeployment := func() *odigosv1alpha1.InstrumentationConfig {
		ic := &odigosv1alpha1.InstrumentationConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      workload.CalculateWorkloadRuntimeObjectName("my-app", k8sconsts.WorkloadKindDeployment),
				Namespace: "test-ns",
			},
		}
		ic.Spec.AgentInjectionEnabled = true
		return ic
	}

	tests := []struct {
		name       string
		configYaml string
		wantReason string
	}{
		{
			// what the "time" settings component makes easy to type
			name:       "grace time missing a unit",
			configYaml: "rollbackGraceTime: \"5\"\n",
			wantReason: string(status.AutoRollbackReasonWaitingForRollout),
		},
		{
			name:       "stability window in prose",
			configYaml: "rollbackStabilityWindow: \"5 minutes\"\n",
			wantReason: string(status.AutoRollbackReasonWaitingForRollout),
		},
		{
			name:       "valid durations",
			configYaml: "rollbackGraceTime: \"30s\"\nrollbackStabilityWindow: \"10m\"\n",
			wantReason: string(status.AutoRollbackReasonWaitingForRollout),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(consts.CurrentNamespaceEnvVar, odigosNamespace)

			scheme := runtime.NewScheme()
			require.NoError(t, clientgoscheme.AddToScheme(scheme))
			require.NoError(t, odigosv1alpha1.AddToScheme(scheme))

			cacheClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects([]client.Object{effectiveConfig(tt.configYaml), instrumentedDeployment()}...).
				Build()

			l := loaders.NewLoaders(logr.Discard(), cacheClient)
			ctx := loaders.WithLoaders(context.Background(), l)
			require.NoError(t, l.LoadConfig(ctx))

			got, err := (&Resolver{}).K8sWorkload().AutoRollback(ctx, &model.K8sWorkload{
				ID: &model.K8sWorkloadID{
					Namespace: "test-ns",
					Name:      "my-app",
					Kind:      model.K8sResourceKindDeployment,
				},
			})
			require.NoError(t, err)
			require.NotNil(t, got)
			require.NotNil(t, got.AutoRollbackStatus)
			require.NotNil(t, got.AutoRollbackStatus.ReasonEnum)
			assert.Equal(t, tt.wantReason, *got.AutoRollbackStatus.ReasonEnum)
		})
	}
}
