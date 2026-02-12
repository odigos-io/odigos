package clustercollector

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconfig "github.com/odigos-io/odigos/autoscaler/controllers/common"
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
	cpuLimitPercentageForHPA    = 75
)

var (
	defaultMinReplicas                  = intPtr(1)
	defaultMaxReplicas                  = int32(10)
	ScaleDownStabilizationWindowSeconds = intPtr(900) // 15 minutes cooldown period for scaling down
	ScaleUpStabilizationWindowSeconds   = intPtr(0)   // no cooldown period for scaling up
)

// syncHPA dynamically creates or updates the HorizontalPodAutoscaler (HPA)
// for the Odigos Gateway deployment based on the running Kubernetes version.
//
// Version handling:
//   - < 1.23  → uses autoscaling/v2beta1 (no behavior support)
//   - 1.23 <= version < 1.25 → uses autoscaling/v2beta2 (manual "Min"/"Max" scaling policy)
//   - ≥ 1.25  → uses autoscaling/v2 (stable API, with predefined policy enums)
//
// Scaling logic:
//
//	The HPA combines Odigos custom "odigos_gateway_rejections" metric with
//	standard CPU and memory metrics for hybrid scaling. The custom metric is a
//	binary signal (0 or 1) indicating when ≥50% of gateway pods reject requests
//	due to memory pressure. This allows the autoscaler to react quickly under
//	stress — even when CPU or memory metrics are unavailable (e.g., pods in
//	CrashLoopBackOff and not reporting to the Metrics Server).
//
// Behavior rationale:
//   - ScaleUp → aggressive and fast (add up to +2 pods per 15s if triggered)
//   - ScaleDown → conservative and gradual (reduce ≤1 pod or ≤25% per 60s,
//     with a 15-minute stabilization window to prevent oscillation)
//   - SelectPolicy: Max for scale-up (react to any metric spike),
//     Min for scale-down (only act when all metrics are low)
//
// Metrics summary:
//  1. Object metric  → odigos_gateway_rejections (Value: 0.5 == 50% threshold)
//  2. Resource metric → CPU (AverageValue target based on configured limit)
//  3. Resource metric → Memory (AverageValue target based on configured limit)
//
// This hybrid HPA ensures rapid scale-out when gateways reject due to overload,
// while avoiding aggressive scale-in that could cause instability.
func syncHPA(gateway *odigosv1.CollectorsGroup, ctx context.Context, c client.Client, scheme *runtime.Scheme) error {
	kubeVersion := commonconfig.ControllerConfig.K8sVersion
	logger := log.FromContext(ctx)

	var hpa client.Object

	// Metric thresholds computation
	// Use percentages of the configured resource limits for the HPA targets.
	memLimit := gateway.Spec.ResourcesSettings.GomemlimitMiB * memoryLimitPercentageForHPA / 100
	memQuantity := resource.MustParse(fmt.Sprintf("%dMi", memLimit))

	cpuTargetMillicores := gateway.Spec.ResourcesSettings.CpuLimitMillicores * cpuLimitPercentageForHPA / 100
	cpuQuantity := resource.MustParse(fmt.Sprintf("%dm", cpuTargetMillicores))

	minReplicas := defaultMinReplicas
	if gateway.Spec.ResourcesSettings.MinReplicas != nil && *gateway.Spec.ResourcesSettings.MinReplicas > 0 {
		minReplicas = intPtr(int32(*gateway.Spec.ResourcesSettings.MinReplicas))
	}

	maxReplicas := defaultMaxReplicas
	if gateway.Spec.ResourcesSettings.MaxReplicas != nil && *gateway.Spec.ResourcesSettings.MaxReplicas > 0 {
		maxReplicas = int32(*gateway.Spec.ResourcesSettings.MaxReplicas)
	}

	gatewayDeploymentName := gateway.Spec.DeploymentName
	if gatewayDeploymentName == "" {
		gatewayDeploymentName = k8sconsts.OdigosClusterCollectorDeploymentName
	}

	// ----------------------------------------------------------------------
	// Version switch for Kubernetes compatibility
	// ----------------------------------------------------------------------
	switch {
	// ------------------------------------------------------------------
	// For legacy clusters (<1.23): v2beta1 — no Behavior or Object metrics support
	// ------------------------------------------------------------------
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
					Name:       gatewayDeploymentName,
				},
				MinReplicas: minReplicas,
				MaxReplicas: maxReplicas,
				Metrics: []autoscalingv2beta1.MetricSpec{
					// Memory target
					{
						Type: autoscalingv2beta1.ResourceMetricSourceType,
						Resource: &autoscalingv2beta1.ResourceMetricSource{
							Name:               "memory",
							TargetAverageValue: &memQuantity,
						},
					},
					// CPU target
					{
						Type: autoscalingv2beta1.ResourceMetricSourceType,
						Resource: &autoscalingv2beta1.ResourceMetricSource{
							Name:               "cpu",
							TargetAverageValue: &cpuQuantity,
						},
					},
				},
			},
		}

	// ------------------------------------------------------------------
	// For mid-range clusters (1.23 ≤ version < 1.25): v2beta2
	//      Supports Behavior and Object metrics
	// ------------------------------------------------------------------
	case kubeVersion.LessThan(version.MustParse("1.25.0")):
		minPolicy := autoscalingv2beta2.ScalingPolicySelect("Min")
		maxPolicy := autoscalingv2beta2.ScalingPolicySelect("Max")
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
					Name:       gatewayDeploymentName,
				},
				MinReplicas: minReplicas,
				MaxReplicas: maxReplicas,

				// Behavior supported from v2beta2 onward
				Behavior: &autoscalingv2beta2.HorizontalPodAutoscalerBehavior{
					// Fast scale-up
					ScaleUp: &autoscalingv2beta2.HPAScalingRules{
						StabilizationWindowSeconds: ScaleUpStabilizationWindowSeconds,
						SelectPolicy:               &maxPolicy,
						Policies: []autoscalingv2beta2.HPAScalingPolicy{
							{
								Type:          autoscalingv2beta2.PodsScalingPolicy,
								Value:         2, // add up to 2 pods every 15s
								PeriodSeconds: 15,
							},
						},
					},
					// Slow scale-down (prevent oscillations)
					ScaleDown: &autoscalingv2beta2.HPAScalingRules{
						StabilizationWindowSeconds: ScaleDownStabilizationWindowSeconds, // 15 min
						SelectPolicy:               &minPolicy,
						Policies: []autoscalingv2beta2.HPAScalingPolicy{
							{
								Type:          autoscalingv2beta2.PodsScalingPolicy,
								Value:         1,
								PeriodSeconds: 60,
							},
							{
								Type:          autoscalingv2beta2.PercentScalingPolicy,
								Value:         25,
								PeriodSeconds: 60,
							},
						},
					},
				},

				Metrics: []autoscalingv2beta2.MetricSpec{
					// Custom Odigos binary metric
					{
						Type: autoscalingv2beta2.ObjectMetricSourceType,
						Object: &autoscalingv2beta2.ObjectMetricSource{
							DescribedObject: autoscalingv2beta2.CrossVersionObjectReference{
								APIVersion: "apps/v1",
								Kind:       "Deployment",
								Name:       gatewayDeploymentName,
							},
							Metric: autoscalingv2beta2.MetricIdentifier{
								Name: "odigos_gateway_rejections",
							},
							Target: autoscalingv2beta2.MetricTarget{
								Type:  autoscalingv2beta2.ValueMetricType,
								Value: resource.NewMilliQuantity(500, resource.DecimalSI), // "0.5" equivalent
							},
						},
					},
					// CPU metric
					{
						Type: autoscalingv2beta2.ResourceMetricSourceType,
						Resource: &autoscalingv2beta2.ResourceMetricSource{
							Name: "memory",
							Target: autoscalingv2beta2.MetricTarget{
								Type:         autoscalingv2beta2.AverageValueMetricType,
								AverageValue: &memQuantity,
							},
						},
					},
					// Memory metric
					{
						Type: autoscalingv2beta2.ResourceMetricSourceType,
						Resource: &autoscalingv2beta2.ResourceMetricSource{
							Name: "cpu",
							Target: autoscalingv2beta2.MetricTarget{
								Type:         autoscalingv2beta2.AverageValueMetricType,
								AverageValue: &cpuQuantity,
							},
						},
					},
				},
			},
		}

	// ------------------------------------------------------------------
	// Modern clusters (>=1.25): v2 — fully stable API
	// ------------------------------------------------------------------
	default:
		maxPolicy := autoscalingv2.MaxChangePolicySelect
		minPolicy := autoscalingv2.MinChangePolicySelect
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
					Name:       gatewayDeploymentName,
				},
				MinReplicas: minReplicas,
				MaxReplicas: maxReplicas,
				Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
					ScaleUp: &autoscalingv2.HPAScalingRules{
						StabilizationWindowSeconds: ScaleUpStabilizationWindowSeconds,
						SelectPolicy:               &maxPolicy,
						Policies: []autoscalingv2.HPAScalingPolicy{
							{
								Type:          autoscalingv2.PodsScalingPolicy,
								Value:         2, // add up to 2 pods every 15s
								PeriodSeconds: 15,
							},
						},
					},
					ScaleDown: &autoscalingv2.HPAScalingRules{
						StabilizationWindowSeconds: ScaleDownStabilizationWindowSeconds,
						SelectPolicy:               &minPolicy,
						Policies: []autoscalingv2.HPAScalingPolicy{
							// remove 1 pod or 25% of the pods every 60s whichever is smaller
							{
								Type:          autoscalingv2.PodsScalingPolicy,
								Value:         1,
								PeriodSeconds: 60,
							},
							{
								Type:          autoscalingv2.PercentScalingPolicy,
								Value:         25,
								PeriodSeconds: 60,
							},
						},
					},
				},
				Metrics: []autoscalingv2.MetricSpec{
					{
						Type: autoscalingv2.ObjectMetricSourceType,
						Object: &autoscalingv2.ObjectMetricSource{
							DescribedObject: autoscalingv2.CrossVersionObjectReference{
								APIVersion: "apps/v1",
								Kind:       "Deployment",
								Name:       gatewayDeploymentName,
							},
							Metric: autoscalingv2.MetricIdentifier{
								Name: "odigos_gateway_rejections",
							},
							Target: autoscalingv2.MetricTarget{
								Type:  autoscalingv2.ValueMetricType,
								Value: resource.NewMilliQuantity(500, resource.DecimalSI),
							},
						},
					},
					{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: "memory",
							Target: autoscalingv2.MetricTarget{
								Type:         autoscalingv2.AverageValueMetricType,
								AverageValue: &memQuantity,
							},
						},
					},
					{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: "cpu",
							Target: autoscalingv2.MetricTarget{
								Type:         autoscalingv2.AverageValueMetricType,
								AverageValue: &cpuQuantity,
							},
						},
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

	logger.Info("Successfully applied HPA", "name", k8sconsts.OdigosClusterCollectorHpaName, "namespace", gateway.Namespace)
	return nil
}

func buildHPACommonFields(gateway *odigosv1.CollectorsGroup) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      k8sconsts.OdigosClusterCollectorHpaName,
		Namespace: gateway.Namespace,
	}
}
