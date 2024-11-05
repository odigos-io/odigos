package gateway

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/api/resource"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	autoscaling "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	memoryLimitPercentageForHPA = 75
)

var (
	minReplicas                = intPtr(1)
	maxReplicas                = int32(10)
	stabilizationWindowSeconds = intPtr(300) // cooldown period for scaling down
)

func syncHPA(gateway *odigosv1.CollectorsGroup, ctx context.Context, c client.Client, scheme *runtime.Scheme, memConfig *memoryConfigurations) error {
	logger := log.FromContext(ctx)
	memLimit := memConfig.gomemlimitMiB * memoryLimitPercentageForHPA / 100.0
	metricQuantity := resource.MustParse(fmt.Sprintf("%dMi", memLimit))
	hpa := &autoscaling.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "autoscaling/v2",
			Kind:       "HorizontalPodAutoscaler",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      gateway.Name,
			Namespace: gateway.Namespace,
		},
		Spec: autoscaling.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscaling.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       gateway.Name,
			},
			MinReplicas: minReplicas,
			MaxReplicas: maxReplicas,
			Metrics: []autoscaling.MetricSpec{
				{
					Type: autoscaling.ResourceMetricSourceType,
					Resource: &autoscaling.ResourceMetricSource{
						Name: "memory",
						Target: autoscaling.MetricTarget{
							Type:         autoscaling.AverageValueMetricType,
							AverageValue: &metricQuantity,
						},
					},
				},
			},
			Behavior: &autoscaling.HorizontalPodAutoscalerBehavior{
				ScaleDown: &autoscaling.HPAScalingRules{
					StabilizationWindowSeconds: stabilizationWindowSeconds,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(gateway, hpa, scheme); err != nil {
		logger.Error(err, "Failed to set controller reference")
		return err
	}

	hpaBytes, _ := yaml.Marshal(hpa)

	force := true
	patchOptions := client.PatchOptions{
		FieldManager: "odigos",
		Force:        &force,
	}

	return c.Patch(ctx, hpa, client.RawPatch(types.ApplyPatchType, hpaBytes), &patchOptions)
}
