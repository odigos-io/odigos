package gateway

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/util/version"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/api/resource"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta1 "k8s.io/api/autoscaling/v2beta1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

	var hpa metav1.Object
	memLimit := memConfig.gomemlimitMiB * memoryLimitPercentageForHPA / 100.0
	metricQuantity := resource.MustParse(fmt.Sprintf("%dMi", memLimit))

	switch {
	case kubeVersion.LessThan(version.MustParse("1.23.0")):
		fmt.Println("Kube version is less than 1.23.0")
		hpa = &autoscalingv2beta1.HorizontalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "autoscaling/v2beta1",
				Kind:       "HorizontalPodAutoscaler",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      gateway.Name,
				Namespace: gateway.Namespace,
			},
			Spec: autoscalingv2beta1.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2beta1.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       gateway.Name,
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
		fmt.Println("Kube version is less than 1.25.0")
		hpa = &autoscalingv2beta2.HorizontalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "autoscaling/v2beta2",
				Kind:       "HorizontalPodAutoscaler",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      gateway.Name,
				Namespace: gateway.Namespace,
			},
			Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       gateway.Name,
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
		fmt.Println("Kube version is greater than or equal to 1.25.0")

		hpa = &autoscalingv2.HorizontalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "autoscaling/v2",
				Kind:       "HorizontalPodAutoscaler",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      gateway.Name,
				Namespace: gateway.Namespace,
			},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       gateway.Name,
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

	// Also need to type-assert for Patch
	switch v := hpa.(type) {
	case *autoscalingv2beta1.HorizontalPodAutoscaler:
		// Create a JSON map of our HPA
		jsonMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(v)
		if err != nil {
			logger.Error(err, "Failed to convert HPA to unstructured")
			return err
		}

		obj := &unstructured.Unstructured{Object: jsonMap}
		// Explicitly set these again to be sure
		obj.SetAPIVersion("autoscaling/v2beta1")
		obj.SetKind("HorizontalPodAutoscaler")
		obj.SetName(gateway.Name)
		obj.SetNamespace(gateway.Namespace)

		force := true
		patchOptions := client.PatchOptions{
			FieldManager: "odigos",
			Force:        &force,
		}
		return c.Patch(ctx, obj, client.Apply, &patchOptions)
	case *autoscalingv2beta2.HorizontalPodAutoscaler:
		jsonMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(v)
		if err != nil {
			logger.Error(err, "Failed to convert HPA to unstructured")
			return err
		}

		obj := &unstructured.Unstructured{Object: jsonMap}
		obj.SetAPIVersion("autoscaling/v2beta2")
		obj.SetKind("HorizontalPodAutoscaler")
		obj.SetName(gateway.Name)
		obj.SetNamespace(gateway.Namespace)

		logger.Info("Unstructured object after setting fields",
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"apiVersion", obj.GetAPIVersion(),
			"kind", obj.GetKind())

		force := true
		patchOptions := client.PatchOptions{
			FieldManager: "odigos",
			Force:        &force,
		}
		return c.Patch(ctx, obj, client.Apply, &patchOptions)
	case *autoscalingv2.HorizontalPodAutoscaler:
		logger.Info("Original HPA object", "name", v.GetName(), "namespace", v.GetNamespace())

		jsonMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(v)
		if err != nil {
			logger.Error(err, "Failed to convert HPA to unstructured")
			return err
		}

		obj := &unstructured.Unstructured{Object: jsonMap}
		obj.SetAPIVersion("autoscaling/v2")
		obj.SetKind("HorizontalPodAutoscaler")
		obj.SetName(gateway.Name)
		obj.SetNamespace(gateway.Namespace)

		force := true
		patchOptions := client.PatchOptions{
			FieldManager: "odigos",
			Force:        &force,
		}
		return c.Patch(ctx, obj, client.Apply, &patchOptions)
	default:
		return fmt.Errorf("unsupported HPA type")
	}
}
