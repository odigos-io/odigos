package clustercollector

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconfig "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/autoscaler/controllers/metricshandler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
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
	logger := commonlogger.FromContext(ctx)

	useCustomMetric := false
	apiSvc := &apiregv1.APIService{}
	if err := c.Get(ctx, client.ObjectKey{Name: k8sconsts.CustomMetricsAPIServiceName}, apiSvc); err == nil {
		useCustomMetric = metricshandler.IsOwnedByOdigos(apiSvc)
	}

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

	gatewayDeploymentName := commonconfig.GetDeploymentName(gateway)

	// ----------------------------------------------------------------------
	// Version switch for Kubernetes compatibility
	// ----------------------------------------------------------------------
	switch {
	// ------------------------------------------------------------------
	// For legacy clusters (<1.23): v2beta1 — no Behavior or Object metrics support
	// ------------------------------------------------------------------
	case kubeVersion.LessThan(version.MustParse("1.23.0")):
		hpa = buildv2beta1HPA(gateway, gatewayDeploymentName, minReplicas, maxReplicas, memQuantity, cpuQuantity)

	// ------------------------------------------------------------------
	// For mid-range clusters (1.23 ≤ version < 1.25): v2beta2
	//      Supports Behavior and Object metrics
	// ------------------------------------------------------------------
	case kubeVersion.LessThan(version.MustParse("1.25.0")):
		hpa = buildv2beta2HPA(gateway, gatewayDeploymentName, minReplicas, maxReplicas, useCustomMetric, memQuantity, cpuQuantity)

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
				Metrics: buildv2Metrics(useCustomMetric, memQuantity, cpuQuantity),
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

// The Go types for autoscaling/v2beta1 and autoscaling/v2beta2 were removed from
// k8s.io/api in v0.36. Those API versions still exist on the clusters Odigos
// supports (see k8sconsts.MinK8SVersionForInstallation), so the legacy objects
// are built as unstructured instead of being dropped. Note that unstructured
// content may only hold JSON-compatible values, hence the int64 and quantity
// string conversions below.

// newLegacyHPA wraps a hand-built HPA spec in an unstructured object carrying
// the given legacy autoscaling API version.
func newLegacyHPA(gateway *odigosv1.CollectorsGroup, apiVersion string, spec map[string]interface{}) *unstructured.Unstructured {
	meta := buildHPACommonFields(gateway)
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       "HorizontalPodAutoscaler",
			"metadata": map[string]interface{}{
				"name":      meta.Name,
				"namespace": meta.Namespace,
			},
			"spec": spec,
		},
	}
}

func legacyScaleTargetRef(deploymentName string) map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"name":       deploymentName,
	}
}

func buildv2beta1HPA(gateway *odigosv1.CollectorsGroup, deploymentName string, minReplicas *int32, maxReplicas int32,
	memQuantity, cpuQuantity resource.Quantity) *unstructured.Unstructured {
	spec := map[string]interface{}{
		"scaleTargetRef": legacyScaleTargetRef(deploymentName),
		"maxReplicas":    int64(maxReplicas),
		"metrics": []interface{}{
			map[string]interface{}{
				"type": "Resource",
				"resource": map[string]interface{}{
					"name":               string(corev1.ResourceMemory),
					"targetAverageValue": memQuantity.String(),
				},
			},
			map[string]interface{}{
				"type": "Resource",
				"resource": map[string]interface{}{
					"name":               string(corev1.ResourceCPU),
					"targetAverageValue": cpuQuantity.String(),
				},
			},
		},
	}
	if minReplicas != nil {
		spec["minReplicas"] = int64(*minReplicas)
	}
	return newLegacyHPA(gateway, "autoscaling/v2beta1", spec)
}

func buildv2beta2HPA(gateway *odigosv1.CollectorsGroup, deploymentName string, minReplicas *int32, maxReplicas int32,
	useCustomMetric bool, memQuantity, cpuQuantity resource.Quantity) *unstructured.Unstructured {
	spec := map[string]interface{}{
		"scaleTargetRef": legacyScaleTargetRef(deploymentName),
		"maxReplicas":    int64(maxReplicas),

		// Behavior supported from v2beta2 onward
		"behavior": map[string]interface{}{
			// Fast scale-up
			"scaleUp": map[string]interface{}{
				"stabilizationWindowSeconds": int64(*ScaleUpStabilizationWindowSeconds),
				"selectPolicy":               "Max",
				"policies": []interface{}{
					map[string]interface{}{
						"type":          "Pods",
						"value":         int64(2), // add up to 2 pods every 15s
						"periodSeconds": int64(15),
					},
				},
			},
			// Slow scale-down (prevent oscillations)
			"scaleDown": map[string]interface{}{
				"stabilizationWindowSeconds": int64(*ScaleDownStabilizationWindowSeconds), // 15 min
				"selectPolicy":               "Min",
				"policies": []interface{}{
					map[string]interface{}{
						"type":          "Pods",
						"value":         int64(1),
						"periodSeconds": int64(60),
					},
					map[string]interface{}{
						"type":          "Percent",
						"value":         int64(25),
						"periodSeconds": int64(60),
					},
				},
			},
		},

		"metrics": buildv2beta2Metrics(useCustomMetric, memQuantity, cpuQuantity),
	}
	if minReplicas != nil {
		spec["minReplicas"] = int64(*minReplicas)
	}
	return newLegacyHPA(gateway, "autoscaling/v2beta2", spec)
}

func buildv2beta2Metrics(useCustomMetric bool, memQuantity, cpuQuantity resource.Quantity) []interface{} {
	metrics := []interface{}{}
	if useCustomMetric {
		metrics = append(metrics, map[string]interface{}{
			"type": "Object",
			"object": map[string]interface{}{
				"describedObject": map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"name":       k8sconsts.OdigosClusterCollectorDeploymentName,
				},
				"metric": map[string]interface{}{
					"name": "odigos_gateway_rejections",
				},
				"target": map[string]interface{}{
					"type":  "Value",
					"value": resource.NewMilliQuantity(500, resource.DecimalSI).String(),
				},
			},
		})
	}
	metrics = append(metrics,
		map[string]interface{}{
			"type": "Resource",
			"resource": map[string]interface{}{
				"name": string(corev1.ResourceMemory),
				"target": map[string]interface{}{
					"type":         "AverageValue",
					"averageValue": memQuantity.String(),
				},
			},
		},
		map[string]interface{}{
			"type": "Resource",
			"resource": map[string]interface{}{
				"name": string(corev1.ResourceCPU),
				"target": map[string]interface{}{
					"type":         "AverageValue",
					"averageValue": cpuQuantity.String(),
				},
			},
		},
	)
	return metrics
}

func buildv2Metrics(useCustomMetric bool, memQuantity, cpuQuantity resource.Quantity) []autoscalingv2.MetricSpec {
	metrics := []autoscalingv2.MetricSpec{}
	if useCustomMetric {
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ObjectMetricSourceType,
			Object: &autoscalingv2.ObjectMetricSource{
				DescribedObject: autoscalingv2.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       k8sconsts.OdigosClusterCollectorDeploymentName,
				},
				Metric: autoscalingv2.MetricIdentifier{
					Name: "odigos_gateway_rejections",
				},
				Target: autoscalingv2.MetricTarget{
					Type:  autoscalingv2.ValueMetricType,
					Value: resource.NewMilliQuantity(500, resource.DecimalSI),
				},
			},
		})
	}
	metrics = append(metrics,
		autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscalingv2.MetricTarget{
					Type:         autoscalingv2.AverageValueMetricType,
					AverageValue: &memQuantity,
				},
			},
		},
		autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:         autoscalingv2.AverageValueMetricType,
					AverageValue: &cpuQuantity,
				},
			},
		},
	)
	return metrics
}
