package odigosurltemplateprocessor

import (
	"fmt"
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// For fast lookup and efficiency, we use a map of string to search if a workload participates
// in the templating process or not.
// Givin a workload from config as {Namespace, Kind, Name} the WorkloadStringRepresentation
// will be {Namespace}/{Kind}/{Name} (as "/" is disallowed in k8s object names).
type workloadStringRepresentation string

func k8sWorkloadToStringRepresentation(workload K8sWorkload) workloadStringRepresentation {
	lowercaseKind := strings.ToLower(workload.Kind)
	return workloadStringRepresentation(fmt.Sprintf("%s/%s/%s", workload.Namespace, lowercaseKind, workload.Name))
}

// internal representation of the custom id config.
// in this representation, the regexp is already parsed from the input string.
type internalCustomIdConfig struct {
	Regexp regexp.Regexp
	Name   string
}

func resourceToWorkloadStringRepresentation(resource pcommon.Resource) (workloadStringRepresentation, error) {

	ns, ok := resource.Attributes().Get(string(semconv.K8SNamespaceNameKey))
	if !ok {
		return "", fmt.Errorf("namespace not found in resource")
	}
	// Check if the namespace is a string
	if ns.Type() != pcommon.ValueTypeStr {
		return "", fmt.Errorf("namespace is not a string")
	}

	// Check for deployments
	deployment, ok := resource.Attributes().Get(string(semconv.K8SDeploymentNameKey))
	if ok {
		// Check if the deployment is a string
		if deployment.Type() != pcommon.ValueTypeStr {
			return "", fmt.Errorf("deployment is not a string")
		}
		return k8sWorkloadToStringRepresentation(K8sWorkload{
			Namespace: ns.Str(),
			Kind:      "deployment",
			Name:      deployment.Str(),
		}), nil
	}

	// Check for statefulsets
	statefulset, ok := resource.Attributes().Get(string(semconv.K8SStatefulSetNameKey))
	if ok {
		// Check if the statefulset is a string
		if statefulset.Type() != pcommon.ValueTypeStr {
			return "", fmt.Errorf("statefulset is not a string")
		}
		return k8sWorkloadToStringRepresentation(K8sWorkload{
			Namespace: ns.Str(),
			Kind:      "statefulset",
			Name:      statefulset.Str(),
		}), nil
	}

	// Check for daemonsets
	daemonset, ok := resource.Attributes().Get(string(semconv.K8SDaemonSetNameKey))
	if ok {
		// Check if the daemonset is a string
		if daemonset.Type() != pcommon.ValueTypeStr {
			return "", fmt.Errorf("daemonset is not a string")
		}
		return k8sWorkloadToStringRepresentation(K8sWorkload{
			Namespace: ns.Str(),
			Kind:      "daemonset",
			Name:      daemonset.Str(),
		}), nil
	}

	return "", fmt.Errorf("no workload found in resource")
}

type PropertiesMatcher struct {
	workloads map[workloadStringRepresentation]struct{}
}

func NewPropertiesMatcher(config *MatchProperties) *PropertiesMatcher {
	if config == nil {
		return nil
	}

	workloads := make(map[workloadStringRepresentation]struct{}, len(config.K8sWorkloads))
	for _, workload := range config.K8sWorkloads {
		workloadString := k8sWorkloadToStringRepresentation(workload)
		workloads[workloadString] = struct{}{}
	}
	return &PropertiesMatcher{
		workloads: workloads,
	}
}

func (p *PropertiesMatcher) Match(resource pcommon.Resource) bool {
	// at the moment, the match function will only check for the presence of a workload
	// in the resource attributes, but this can be extended in the future to match on other properties
	// if needed.
	// converting to a string in checking in map for efficient lookup and scalability.
	workloadString, err := resourceToWorkloadStringRepresentation(resource)
	if err != nil {
		return false
	}
	_, found := p.workloads[workloadString]
	return found
}
