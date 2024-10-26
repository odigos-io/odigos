package describe

import (
	"context"
	"fmt"
	"strings"

	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	odigos "github.com/odigos-io/odigos/k8sutils/pkg/describe/odigos"
	"github.com/odigos-io/odigos/k8sutils/pkg/getters"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func printOdigosVersion(odigosVersion string, sb *strings.Builder) {
	describeText(sb, 0, "Odigos Version: %s", odigosVersion)
}

func printClusterCollectorStatus(resources *odigos.OdigosResources, sb *strings.Builder) {

	expectingClusterCollector := len(resources.Destinations.Items) > 0

	describeText(sb, 1, "Cluster Collector:")
	clusterCollector := resources.ClusterCollector

	if expectingClusterCollector {
		describeText(sb, 2, "Status: Cluster Collector is expected to be created because there are destinations")
	} else {
		describeText(sb, 2, "Status: Cluster Collector is not expected to be created because there are no destinations")
	}

	if clusterCollector.CollectorsGroup == nil {
		describeText(sb, 2, wrapTextSuccessOfFailure("Collectors Group Not Created", !expectingClusterCollector))
	} else {
		describeText(sb, 2, wrapTextSuccessOfFailure("Collectors Group Created", expectingClusterCollector))

		var deployedCondition *metav1.Condition
		for _, condition := range clusterCollector.CollectorsGroup.Status.Conditions {
			if condition.Type == "Deployed" {
				deployedCondition = &condition
				break
			}
		}
		if deployedCondition == nil {
			describeText(sb, 2, wrapTextInRed("Deployed: Status Unavailable"))
		} else {
			if deployedCondition.Status == metav1.ConditionTrue {
				describeText(sb, 2, wrapTextInGreen("Deployed: true"))
			} else {
				describeText(sb, 2, wrapTextInRed("Deployed: false"))
				describeText(sb, 2, wrapTextInRed(fmt.Sprintf("Reason: %s", deployedCondition.Message)))
			}
		}

		ready := clusterCollector.CollectorsGroup.Status.Ready
		describeText(sb, 2, wrapTextSuccessOfFailure(fmt.Sprintf("Ready: %t", ready), ready))
	}

	expectedReplicas := int32(0)
	if clusterCollector.Deployment == nil {
		describeText(sb, 2, wrapTextSuccessOfFailure("Deployment: Not Found", !expectingClusterCollector))
	} else {
		describeText(sb, 2, wrapTextSuccessOfFailure("Deployment: Found", expectingClusterCollector))
		expectedReplicas = *clusterCollector.Deployment.Spec.Replicas
		describeText(sb, 2, fmt.Sprintf("Expected Replicas: %d", expectedReplicas))
	}

	if clusterCollector.LatestRevisionPods != nil {
		runningReplicas := 0
		failureReplicas := 0
		var failureText string
		for _, pod := range clusterCollector.LatestRevisionPods.Items {
			var condition *corev1.PodCondition
			for i := range pod.Status.Conditions {
				c := pod.Status.Conditions[i]
				if c.Type == corev1.PodReady {
					condition = &c
					break
				}
			}
			if condition == nil {
				failureReplicas++
			} else {
				if condition.Status == corev1.ConditionTrue {
					runningReplicas++
				} else {
					failureReplicas++
					failureText = condition.Message
				}
			}
		}
		podReplicasText := fmt.Sprintf("Actual Replicas: %d running, %d failed", runningReplicas, failureReplicas)
		deploymentSuccessful := runningReplicas == int(expectedReplicas) && failureReplicas == 0
		describeText(sb, 2, wrapTextSuccessOfFailure(podReplicasText, deploymentSuccessful))
		if !deploymentSuccessful {
			describeText(sb, 2, wrapTextInRed(fmt.Sprintf("Replicas Not Ready Reason: %s", failureText)))
		}
	}

}

func printAndCalculateIsNodeCollectorStatus(resources *odigos.OdigosResources, sb *strings.Builder) bool {

	numInstrumentationConfigs := len(resources.InstrumentationConfigs.Items)
	if numInstrumentationConfigs == 0 {
		describeText(sb, 2, "Status: Node Collectors not expected as there are no sources")
		return false
	}

	if resources.ClusterCollector.CollectorsGroup == nil {
		describeText(sb, 2, "Status: Node Collectors not expected as there are no destinations")
		return false
	}

	if !resources.ClusterCollector.CollectorsGroup.Status.Ready {
		describeText(sb, 2, "Status: Node Collectors not expected as the Cluster Collector is not ready")
		return false
	}

	describeText(sb, 2, "Status: Node Collectors expected as cluster collector is ready and there are sources")
	return true
}

func printNodeCollectorStatus(resources *odigos.OdigosResources, sb *strings.Builder) {

	describeText(sb, 1, "Node Collector:")
	nodeCollector := resources.NodeCollector

	expectingNodeCollector := printAndCalculateIsNodeCollectorStatus(resources, sb)

	if nodeCollector.CollectorsGroup == nil {
		describeText(sb, 2, wrapTextSuccessOfFailure("Collectors Group Not Created", !expectingNodeCollector))
	} else {
		describeText(sb, 2, wrapTextSuccessOfFailure("Collectors Group Created", expectingNodeCollector))

		var deployedCondition *metav1.Condition
		for _, condition := range nodeCollector.CollectorsGroup.Status.Conditions {
			if condition.Type == "Deployed" {
				deployedCondition = &condition
				break
			}
		}
		if deployedCondition == nil {
			describeText(sb, 2, wrapTextInRed("Deployed: Status Unavailable"))
		} else {
			if deployedCondition.Status == metav1.ConditionTrue {
				describeText(sb, 2, wrapTextInGreen("Deployed: True"))
			} else {
				describeText(sb, 2, wrapTextInRed("Deployed: False"))
				describeText(sb, 2, wrapTextInRed(fmt.Sprintf("Reason: %s", deployedCondition.Message)))
			}
		}

		ready := nodeCollector.CollectorsGroup.Status.Ready
		describeText(sb, 2, wrapTextSuccessOfFailure(fmt.Sprintf("Ready: %t", ready), ready))
	}

	if nodeCollector.DaemonSet == nil {
		describeText(sb, 2, wrapTextSuccessOfFailure("DaemonSet: Not Found", !expectingNodeCollector))
	} else {
		describeText(sb, 2, wrapTextSuccessOfFailure("DaemonSet: Found", expectingNodeCollector))

		// this is copied from k8sutils/pkg/describe/describe.go
		// I hope the info is accurate since there can be many edge cases
		describeText(sb, 2, "Desired Number of Nodes Scheduled: %d", nodeCollector.DaemonSet.Status.DesiredNumberScheduled)
		currentMeetsDesired := nodeCollector.DaemonSet.Status.DesiredNumberScheduled == nodeCollector.DaemonSet.Status.CurrentNumberScheduled
		describeText(sb, 2, wrapTextSuccessOfFailure(fmt.Sprintf("Current Number of Nodes Scheduled: %d", nodeCollector.DaemonSet.Status.CurrentNumberScheduled), currentMeetsDesired))
		updatedMeetsDesired := nodeCollector.DaemonSet.Status.DesiredNumberScheduled == nodeCollector.DaemonSet.Status.UpdatedNumberScheduled
		describeText(sb, 2, wrapTextSuccessOfFailure(fmt.Sprintf("Number of Nodes Scheduled with Up-to-date Pods: %d", nodeCollector.DaemonSet.Status.UpdatedNumberScheduled), updatedMeetsDesired))
		availableMeetsDesired := nodeCollector.DaemonSet.Status.DesiredNumberScheduled == nodeCollector.DaemonSet.Status.NumberAvailable
		describeText(sb, 2, wrapTextSuccessOfFailure(fmt.Sprintf("Number of Nodes Scheduled with Available Pods: %d", nodeCollector.DaemonSet.Status.NumberAvailable), availableMeetsDesired))
		noMisscheduled := nodeCollector.DaemonSet.Status.NumberMisscheduled == 0
		describeText(sb, 2, wrapTextSuccessOfFailure(fmt.Sprintf("Number of Nodes Misscheduled: %d", nodeCollector.DaemonSet.Status.NumberMisscheduled), noMisscheduled))
	}
}

func printOdigosPipeline(resources *odigos.OdigosResources, sb *strings.Builder) {
	describeText(sb, 0, "Odigos Pipeline:")
	numDestinations := len(resources.Destinations.Items)
	numInstrumentationConfigs := len(resources.InstrumentationConfigs.Items)

	describeText(sb, 1, "Status: there are %d sources and %d destinations\n", numInstrumentationConfigs, numDestinations)
	printClusterCollectorStatus(resources, sb)
	sb.WriteString("\n")
	printNodeCollectorStatus(resources, sb)
}

func printDescribeOdigos(odigosVersion string, resources *odigos.OdigosResources) string {
	var sb strings.Builder

	printOdigosVersion(odigosVersion, &sb)
	sb.WriteString("\n")
	printOdigosPipeline(resources, &sb)

	return sb.String()
}

func DescribeOdigos(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, odigosNs string) string {

	odigosVersion, err := getters.GetOdigosVersionInClusterFromConfigMap(ctx, kubeClient, odigosNs)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	odigosResources, err := odigos.GetRelevantOdigosResources(ctx, kubeClient, odigosClient, odigosNs)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	return printDescribeOdigos(odigosVersion, odigosResources)
}
