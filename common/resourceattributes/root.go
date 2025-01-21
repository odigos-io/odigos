// Package resourceattributes provides a set of functions to create resource attributes for a container.
// Resource attributes in Odigos are injected in two phases:
// 1. Before the container starts, the attributes are set to identify the container.
// 2. After the container starts, additional attributes may be added to the container.
// Notice that the second phase is not always called, see AfterPodStart for more details.
//
// Use the following guidelines when adding new resource attribute:
// - If the attribute is fast to calculate and is needed by all clients, add it to BeforePodStart.
// - If the attribute is slow to calculate or is only needed for OpAMP/eBPF clients, add it to AfterPodStart.
// An example of slow to calculate attribute is the cloud provider metadata that requires a network call.
package resourceattributes

import (
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type ContainerIdentifier struct {
	PodName       string
	Namespace     string
	ContainerName string
}

// BeforePodStart returns the resource attributes that should be set before the container starts.
// This function is called by the PodWebhook before the container starts.
// Currently, the attributes returned are the minimal set of attributes that should be set to identify the container.
// Other attributes (k8s related, cloud provider, etc) are set later by a dedicated processor in odigos-data-collector.
// Notice that clients that calls AfterPodStart will overwrite the attributes set by this function.
func BeforePodStart(identifier *ContainerIdentifier) Attributes {
	if identifier == nil {
		return nil
	}

	return identifyingResourceAttributes(identifier)
}

// AfterPodStart returns the resource attributes that should be set after the container starts.
// This function is called by the eBPF director or by the OpAMP server after the container starts.
// Currently, the attributes returned are the minimal set of attributes that should be set to identify the container.
// Other attributes (k8s related, cloud provider, etc) are set later by a dedicated processor in odigos-data-collector.
// Notice that this function is not always called, only OpAMP clients or eBPF-based clients will call this function.
// Vanilla OpenTelemetry clients will only call BeforePodStart.
// OpAMP/eBPF clients WILL OVERWRITE the attributes set by BeforePodStart with the attributes returned by this function.
func AfterPodStart(identifier *ContainerIdentifier) Attributes {
	if identifier == nil {
		return nil
	}

	return identifyingResourceAttributes(identifier)
}

func identifyingResourceAttributes(identifier *ContainerIdentifier) Attributes {
	return []attribute.KeyValue{
		semconv.K8SContainerName(identifier.ContainerName),
		semconv.K8SPodName(identifier.PodName),
		semconv.K8SNamespaceName(identifier.Namespace),
	}
}
