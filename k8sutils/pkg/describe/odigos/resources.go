package odigos

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

type ClusterCollectorResources struct {
	CollectorsGroup    *odigosv1.CollectorsGroup
	Deployment         *appsv1.Deployment
	LatestRevisionPods *corev1.PodList
}

type NodeCollectorResources struct {
	CollectorsGroup *odigosv1.CollectorsGroup
	DaemonSet       *appsv1.DaemonSet
}

type OdigosResources struct {
	OdigosDeployment       *corev1.ConfigMap // guaranteed to exist
	ClusterCollector       ClusterCollectorResources
	NodeCollector          NodeCollectorResources
	Destinations           *odigosv1.DestinationList
	InstrumentationConfigs *odigosv1.InstrumentationConfigList
}

func getClusterCollectorResources(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, odigosNs string) (*ClusterCollectorResources, error) {
	clusterCollector := ClusterCollectorResources{}

	cg, err := odigosClient.CollectorsGroups(odigosNs).Get(ctx, k8sconsts.OdigosClusterCollectorCollectorGroupName, metav1.GetOptions{})
	if err == nil {
		clusterCollector.CollectorsGroup = cg
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}

	dep, err := kubeClient.AppsV1().Deployments(odigosNs).Get(ctx, k8sconsts.OdigosClusterCollectorDeploymentName, metav1.GetOptions{})
	if err == nil {
		clusterCollector.Deployment = dep
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}

	var clusterRoleRevision string
	if dep != nil {
		revisionAnnotation, found := dep.Annotations["deployment.kubernetes.io/revision"]
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
			return nil, errRs
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
				return nil, err
			}
		}
	}

	return &clusterCollector, nil
}

func getNodeCollectorResources(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, odigosNs string) (*NodeCollectorResources, error) {
	nodeCollector := NodeCollectorResources{}

	cg, err := odigosClient.CollectorsGroups(odigosNs).Get(ctx, k8sconsts.OdigosNodeCollectorCollectorGroupName, metav1.GetOptions{})
	if err == nil {
		nodeCollector.CollectorsGroup = cg
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}

	ds, err := kubeClient.AppsV1().DaemonSets(odigosNs).Get(ctx, k8sconsts.OdigosNodeCollectorDaemonSetName, metav1.GetOptions{})
	if err == nil {
		nodeCollector.DaemonSet = ds
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}

	return &nodeCollector, nil
}

func GetRelevantOdigosResources(ctx context.Context, kubeClient kubernetes.Interface, odigosClient odigosclientset.OdigosV1alpha1Interface, odigosNs string) (*OdigosResources, error) {
	odigos := OdigosResources{}

	odigosDeployment, err := kubeClient.CoreV1().ConfigMaps(odigosNs).Get(ctx, k8sconsts.OdigosDeploymentConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	odigos.OdigosDeployment = odigosDeployment

	cc, err := getClusterCollectorResources(ctx, kubeClient, odigosClient, odigosNs)
	if err != nil {
		return nil, err
	}
	odigos.ClusterCollector = *cc

	nc, err := getNodeCollectorResources(ctx, kubeClient, odigosClient, odigosNs)
	if err != nil {
		return nil, err
	}
	odigos.NodeCollector = *nc

	dest, err := odigosClient.Destinations(odigosNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	odigos.Destinations = dest

	ics, err := odigosClient.InstrumentationConfigs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	odigos.InstrumentationConfigs = ics

	return &odigos, nil
}
