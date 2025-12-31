package metrics

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	workload "github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	otelMeterName   = "github.com/odigos.io/odigos/odiglet"
	languageAttrKey = "telemetry_distro_name"
)

var meter = otel.Meter(otelMeterName)

// OdigletMetrics tracks unified metrics for all instrumented pods
type OdigletMetrics struct {
	k8sClient client.Client

	// instrumentedPodsByLanguage tracks all instrumented pods per language
	instrumentedPodsByLanguage otelmetric.Int64ObservableGauge

	registration otelmetric.Registration
}

// NewOdigletMetrics creates unified metrics based on InstrumentationConfig
func NewOdigletMetrics(k8sClient client.Client) (*OdigletMetrics, error) {
	m := &OdigletMetrics{
		k8sClient: k8sClient,
	}

	var err error
	m.instrumentedPodsByLanguage, err = meter.Int64ObservableGauge(
		"odigos_odiglet_instrumented_pods_by_language",
		otelmetric.WithDescription("Number of instrumented pods per programming language"),
		otelmetric.WithUnit("{pod}"),
	)
	if err != nil {
		return nil, err
	}

	// Register callback to observe the gauge
	m.registration, err = meter.RegisterCallback(
		m.observeInstrumentedPods,
		m.instrumentedPodsByLanguage,
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// observeInstrumentedPods counts instrumented pods by language using InstrumentationConfig
func (m *OdigletMetrics) observeInstrumentedPods(ctx context.Context, observer otelmetric.Observer) error {
	counts := make(map[string]int)

	// List all pods (the cache is already filtered to this node via manager config)
	var podList corev1.PodList
	if err := m.k8sClient.List(ctx, &podList); err != nil {
		return nil
	}

	// For each pod, check if it has an associated InstrumentationConfig with enabled agents
	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		// Get the workload owner reference
		ownerRef, err := workload.PodWorkloadObject(ctx, &pod)
		if err != nil || ownerRef == nil {
			continue
		}

		// Get the InstrumentationConfig for this workload
		icName := workload.CalculateWorkloadRuntimeObjectName(ownerRef.Name, ownerRef.Kind)
		if icName == "" {
			continue
		}

		var ic odigosv1.InstrumentationConfig
		if err := m.k8sClient.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: icName}, &ic); err != nil {
			continue
		}

		// Check each container for enabled agents
		for _, containerConfig := range ic.Spec.Containers {
			if !containerConfig.AgentEnabled {
				continue
			}

			counts[string(containerConfig.OtelDistroName)]++
		}
	}

	// Emit all counts
	for language, count := range counts {
		observer.ObserveInt64(m.instrumentedPodsByLanguage, int64(count),
			otelmetric.WithAttributes(
				attribute.String(languageAttrKey, language),
			))
	}

	return nil
}

// getWorkloadOwnerRef finds the workload owner reference (Deployment, StatefulSet, DaemonSet, ReplicaSet)
func getWorkloadOwnerRef(pod *corev1.Pod) *metav1.OwnerReference {
	for i := range pod.OwnerReferences {
		ref := &pod.OwnerReferences[i]
		switch ref.Kind {
		case "ReplicaSet", "Deployment", "StatefulSet", "DaemonSet":
			return ref
		}
	}
	return nil
}

// Close unregisters the metrics callback
func (m *OdigletMetrics) Close() error {
	if m.registration != nil {
		return m.registration.Unregister()
	}
	return nil
}
