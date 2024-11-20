package gateway

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta1 "k8s.io/api/autoscaling/v2beta1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	memoryLimitPercentageForHPA = 75
)

var (
	minReplicas                = intPtr(1)
	maxReplicas                = int32(10)
	stabilizationWindowSeconds = intPtr(300) // cooldown period for scaling down
)

func syncHPA(gateway *odigosv1.CollectorsGroup, ctx context.Context, c client.Client, scheme *runtime.Scheme, memConfig *memoryConfigurations, kubeVersion *version.Version) error {
	logger := log.FromContext(ctx)

	var hpa client.Object

	memLimit := memConfig.gomemlimitMiB * memoryLimitPercentageForHPA / 100.0
	metricQuantity := resource.MustParse(fmt.Sprintf("%dMi", memLimit))

	switch {
	case kubeVersion.LessThan(version.MustParse("1.23.0")):
		hpa = &autoscalingv2beta1.HorizontalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "autoscaling/v2beta1",
				Kind:       "HorizontalPodAutoscaler",
			},
			ObjectMeta: buildHPACommonFields(gateway),
			Spec: autoscalingv2beta1.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2beta1.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       consts.OdigosClusterCollectorDeploymentName,
				},
				MinReplicas: minReplicas,
				MaxReplicas: maxReplicas,
				Metrics: []autoscalingv2beta1.MetricSpec{
					{
						Type: autoscalingv2beta1.ResourceMetricSourceType,
						Resource: &autoscalingv2beta1.ResourceMetricSource{
							Name:               "memory",
							TargetAverageValue: &metricQuantity,
						},
					},
				},
			},
		}
	case kubeVersion.LessThan(version.MustParse("1.25.0")):
		hpa = &autoscalingv2beta2.HorizontalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "autoscaling/v2beta2",
				Kind:       "HorizontalPodAutoscaler",
			},
			ObjectMeta: buildHPACommonFields(gateway),
			Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       consts.OdigosClusterCollectorDeploymentName,
				},
				MinReplicas: minReplicas,
				MaxReplicas: maxReplicas,
				Metrics: []autoscalingv2beta2.MetricSpec{
					{
						Type: autoscalingv2beta2.ResourceMetricSourceType,
						Resource: &autoscalingv2beta2.ResourceMetricSource{
							Name: "memory",
							Target: autoscalingv2beta2.MetricTarget{
								Type:         autoscalingv2beta2.AverageValueMetricType,
								AverageValue: &metricQuantity,
							},
						},
					},
				},
				Behavior: &autoscalingv2beta2.HorizontalPodAutoscalerBehavior{
					ScaleDown: &autoscalingv2beta2.HPAScalingRules{
						StabilizationWindowSeconds: stabilizationWindowSeconds,
					},
				},
			},
		}
	default:
		hpa = &autoscalingv2.HorizontalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "autoscaling/v2",
				Kind:       "HorizontalPodAutoscaler",
			},
			ObjectMeta: buildHPACommonFields(gateway),
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       consts.OdigosClusterCollectorDeploymentName,
				},
				MinReplicas: minReplicas,
				MaxReplicas: maxReplicas,
				Metrics: []autoscalingv2.MetricSpec{
					{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: "memory",
							Target: autoscalingv2.MetricTarget{
								Type:         autoscalingv2.AverageValueMetricType,
								AverageValue: &metricQuantity,
							},
						},
					},
				},
				Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
					ScaleDown: &autoscalingv2.HPAScalingRules{
						StabilizationWindowSeconds: stabilizationWindowSeconds,
					},
				},
			},
		}
	}

	if err := controllerutil.SetControllerReference(gateway, hpa, scheme); err != nil {
		logger.Error(err, "Failed to set controller reference")
		return err
	}

	// Use the Apply patch strategy
	applyOpts := &client.PatchOptions{
		FieldManager: "odigos",
		Force:        pointer.Bool(true),
	}

	if err := c.Patch(ctx, hpa, client.Apply, applyOpts); err != nil {
		logger.Error(err, "Failed to apply patch to HPA")
		return err
	}

	logger.Info("Successfully applied HPA", "name", consts.OdigosClusterCollectorDeploymentName, "namespace", gateway.Namespace)
	return nil
}

func buildHPACommonFields(gateway *odigosv1.CollectorsGroup) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      consts.OdigosClusterCollectorDeploymentName,
		Namespace: gateway.Namespace,
	}
}
