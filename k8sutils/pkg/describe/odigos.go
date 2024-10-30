package describe

import (
	"context"
	"fmt"
	"strings"

	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/getters"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type clusterCollectorResources struct {
	CollectorsGroup    *odigosv1.CollectorsGroup
	Deployment         *appsv1.Deployment
	LatestRevisionPods *corev1.PodList
}

type nodeCollectorResources struct {
	CollectorsGroup *odigosv1.CollectorsGroup
	DaemonSet       *appsv1.DaemonSet
}

type odigosResources struct {
	ClusterCollector       clusterCollectorResources
	NodeCollector          nodeCollectorResources
	Destinations           *odigosv1.DestinationList
	InstrumentationConfigs *odigosv1.InstrumentationConfigList
}

func getClusterCollectorResources(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, odigosNs string) (clusterCollector clusterCollectorResources, err error) {

	clusterCollector = clusterCollectorResources{}

	clusterCollector.CollectorsGroup, err = odigosClient.CollectorsGroups(odigosNs).Get(ctx, consts.OdigosClusterCollectorCollectorGroupName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return
	}

	clusterCollector.Deployment, err = kubeClient.AppsV1().Deployments(odigosNs).Get(ctx, consts.OdigosClusterCollectorDeploymentName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return
	}

	var clusterRoleRevision string
	if clusterCollector.Deployment != nil {
		revisionAnnotation, found := clusterCollector.Deployment.Annotations["deployment.kubernetes.io/revision"]
		if found {
			clusterRoleRevision = revisionAnnotation
		}
	}

	// get only cluster role replicasets
	if clusterRoleRevision != "" {
		deploymentRs, errRs := kubeClient.AppsV1().ReplicaSets(odigosNs).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(clusterCollector.Deployment.Spec.Selector),
		})
		if errRs != nil {
			err = errRs
			return
		}

		var latestRevisionReplicaSet *appsv1.ReplicaSet
		for i := range deploymentRs.Items {
			rs := &deploymentRs.Items[i]
			if rs.Annotations["deployment.kubernetes.io/revision"] == clusterRoleRevision {
				latestRevisionReplicaSet = rs
				break
			}
		}

		if latestRevisionReplicaSet != nil {
			podTemplateHash := latestRevisionReplicaSet.Labels["pod-template-hash"]
			clusterCollector.LatestRevisionPods, err = kubeClient.CoreV1().Pods(odigosNs).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("pod-template-hash=%s", podTemplateHash),
			})
			if err != nil {
				return
			}
		}
	}

	return
}

func getNodeCollectorResources(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, odigosNs string) (nodeCollector nodeCollectorResources, err error) {

	nodeCollector = nodeCollectorResources{}

	nodeCollector.CollectorsGroup, err = odigosClient.CollectorsGroups(odigosNs).Get(ctx, consts.OdigosNodeCollectorCollectorGroupName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return
	}

	nodeCollector.DaemonSet, err = kubeClient.AppsV1().DaemonSets(odigosNs).Get(ctx, consts.OdigosNodeCollectorDaemonSetName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return
	}

	return
}

func getRelevantOdigosResources(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, odigosNs string) (odigos odigosResources, err error) {

	odigos.ClusterCollector, err = getClusterCollectorResources(ctx, kubeClient, odigosClient, odigosNs)
	if err != nil {
		return
	}

	odigos.NodeCollector, err = getNodeCollectorResources(ctx, kubeClient, odigosClient, odigosNs)
	if err != nil {
		return
	}

	odigos.Destinations, err = odigosClient.Destinations(odigosNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}

	odigos.InstrumentationConfigs, err = odigosClient.InstrumentationConfigs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}

	return
}

func printOdigosVersion(odigosVersion string, sb *strings.Builder) {
	describeText(sb, 0, "Odigos Version: %s", odigosVersion)
}

func printOdigosPipelineStatus(numInstrumentationConfigs, numDestinations int, expectingPipeline bool, sb *strings.Builder) {
	if expectingPipeline {
		describeText(sb, 1, "Status: there are %d sources and %d destinations so pipeline will be deployed\n", numInstrumentationConfigs, numDestinations)
	} else {
		describeText(sb, 1, "Status: no sources or destinations found so pipeline will not be deployed")
	}
}

func printClusterCollectorStatus(clusterCollector clusterCollectorResources, expectingPipeline bool, sb *strings.Builder) {
	describeText(sb, 1, "Cluster Collector:")
	if clusterCollector.CollectorsGroup == nil {
		describeText(sb, 2, wrapTextSuccessOfFailure("Collectors Group Not Created", !expectingPipeline))
		return
	}

	describeText(sb, 2, wrapTextSuccessOfFailure("Collectors Group Created", expectingPipeline))

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

	if clusterCollector.LatestRevisionPods == nil || clusterCollector.Deployment == nil {
		describeText(sb, 2, wrapTextInRed("Number of Replicas: Status Unavailable"))
	} else {
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
		desiredReplicas := *clusterCollector.Deployment.Spec.Replicas
		describeText(sb, 2, fmt.Sprintf("Desired Replicas: %d", desiredReplicas))
		podReplicasText := fmt.Sprintf("Actual Replicas: %d running, %d failed", runningReplicas, failureReplicas)
		deploymentSuccessful := runningReplicas == int(desiredReplicas)
		describeText(sb, 2, wrapTextSuccessOfFailure(podReplicasText, deploymentSuccessful))
		if !deploymentSuccessful {
			describeText(sb, 2, wrapTextInRed(fmt.Sprintf("Replicas Not Ready Reason: %s", failureText)))
		}
	}
}

func printNodeCollectorStatus(nodeCollector nodeCollectorResources, expectingNodeCollector bool, sb *strings.Builder) {
	describeText(sb, 1, "Node Collector:")
	if nodeCollector.CollectorsGroup == nil {
		describeText(sb, 2, wrapTextSuccessOfFailure("Collectors Group Not Created", !expectingNodeCollector))
		return
	}

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

func printOdigosPipeline(odigosResources odigosResources, sb *strings.Builder) {
	describeText(sb, 0, "Odigos Pipeline:")
	numDestinations := len(odigosResources.Destinations.Items)
	numInstrumentationConfigs := len(odigosResources.InstrumentationConfigs.Items)
	// odigos will only initiate pipeline if there are any sources or destinations
	expectingPipeline := numDestinations > 0 || numInstrumentationConfigs > 0

	printOdigosPipelineStatus(numInstrumentationConfigs, numDestinations, expectingPipeline, sb)
	printClusterCollectorStatus(odigosResources.ClusterCollector, expectingPipeline, sb)
	sb.WriteString("\n")
	expectingNodeCollector := odigosResources.ClusterCollector.CollectorsGroup != nil && odigosResources.ClusterCollector.CollectorsGroup.Status.Ready && numInstrumentationConfigs > 0
	printNodeCollectorStatus(odigosResources.NodeCollector, expectingNodeCollector, sb)
}

func printDescribeOdigos(odigosVersion string, odigosResources odigosResources) string {
	var sb strings.Builder

	printOdigosVersion(odigosVersion, &sb)
	sb.WriteString("\n")
	printOdigosPipeline(odigosResources, &sb)

	return sb.String()
}

func DescribeOdigos(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, odigosNs string) string {

	odigosVersion, err := getters.GetOdigosVersionInClusterFromConfigMap(ctx, kubeClient, odigosNs)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	odigosResources, err := getRelevantOdigosResources(ctx, kubeClient, odigosClient, odigosNs)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	return printDescribeOdigos(odigosVersion, odigosResources)
}
