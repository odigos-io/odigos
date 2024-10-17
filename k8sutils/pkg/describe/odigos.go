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

func getRelevantOdigosResources(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, odigosNs string) (clusterCollector clusterCollectorResources, destinations *odigosv1.DestinationList, instrumentationConfigs *odigosv1.InstrumentationConfigList, err error) {

	clusterCollector, err = getClusterCollectorResources(ctx, kubeClient, odigosClient, odigosNs)
	if err != nil {
		return
	}

	destinations, err = odigosClient.Destinations(odigosNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}

	instrumentationConfigs, err = odigosClient.InstrumentationConfigs("").List(ctx, metav1.ListOptions{})
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

func printClusterGatewayStatus(clusterCollector clusterCollectorResources, expectingPipeline bool, sb *strings.Builder) {
	describeText(sb, 1, "Cluster Gateway:")
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
			describeText(sb, 2, wrapTextInGreen("Deployed: True"))
		} else {
			describeText(sb, 2, wrapTextInRed("Deployed: False"))
			describeText(sb, 2, wrapTextInRed(fmt.Sprintf("Reason: %s", deployedCondition.Message)))
		}
	}

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

func printOdigosPipeline(clusterCollector clusterCollectorResources, destinations *odigosv1.DestinationList, instrumentationConfigs *odigosv1.InstrumentationConfigList, sb *strings.Builder) {
	describeText(sb, 0, "Odigos Pipeline:")
	numDestinations := len(destinations.Items)
	numInstrumentationConfigs := len(instrumentationConfigs.Items)
	// odigos will only initiate pipeline if there are any sources or destinations
	expectingPipeline := numDestinations > 0 || numInstrumentationConfigs > 0

	printOdigosPipelineStatus(numInstrumentationConfigs, numDestinations, expectingPipeline, sb)
	printClusterGatewayStatus(clusterCollector, expectingPipeline, sb)
}

func printDescribeOdigos(odigosVersion string, clusterCollector clusterCollectorResources, destinations *odigosv1.DestinationList, instrumentationConfigs *odigosv1.InstrumentationConfigList) string {
	var sb strings.Builder

	printOdigosVersion(odigosVersion, &sb)
	sb.WriteString("\n")
	printOdigosPipeline(clusterCollector, destinations, instrumentationConfigs, &sb)

	return sb.String()
}

func DescribeOdigos(ctx context.Context, kubeClient *kubernetes.Clientset, odigosClient odigosclientset.OdigosV1alpha1Interface, odigosNs string) string {

	odigosVersion, err := getters.GetOdigosVersionInClusterFromConfigMap(ctx, kubeClient, odigosNs)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	clusterCollector, destinations, instrumentationConfigs, err := getRelevantOdigosResources(ctx, kubeClient, odigosClient, odigosNs)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	return printDescribeOdigos(odigosVersion, clusterCollector, destinations, instrumentationConfigs)
}
