package odigos

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
)

type ClusterCollectorAnalyze struct {
	Enabled              properties.EntityProperty  `json:"enabled"`
	CollectorGroup       properties.EntityProperty  `json:"collectorGroup"`
	Deployed             *properties.EntityProperty `json:"deployed,omitempty"`
	DeployedError        *properties.EntityProperty `json:"deployedError,omitempty"`
	CollectorReady       *properties.EntityProperty `json:"collectorReady,omitempty"`
	DeploymentCreated    properties.EntityProperty  `json:"deployment,omitempty"`
	ExpectedReplicas     *properties.EntityProperty `json:"expectedReplicas,omitempty"`
	HealthyReplicas      *properties.EntityProperty `json:"healthyReplicas,omitempty"`
	FailedReplicas       *properties.EntityProperty `json:"failedReplicas,omitempty"`
	FailedReplicasReason *properties.EntityProperty `json:"failedReplicasReason,omitempty"`
}

type NodeCollectorAnalyze struct {
	Enabled        properties.EntityProperty  `json:"enabled"`
	CollectorGroup properties.EntityProperty  `json:"collectorGroup"`
	Deployed       *properties.EntityProperty `json:"deployed,omitempty"`
	DeployedError  *properties.EntityProperty `json:"deployedError,omitempty"`
	CollectorReady *properties.EntityProperty `json:"collectorReady,omitempty"`
	DaemonSet      properties.EntityProperty  `json:"daemonSet,omitempty"`
	DesiredNodes   *properties.EntityProperty `json:"desiredNodes,omitempty"`
	CurrentNodes   *properties.EntityProperty `json:"currentNodes,omitempty"`
	UpdatedNodes   *properties.EntityProperty `json:"updatedNodes,omitempty"`
	AvailableNodes *properties.EntityProperty `json:"availableNodes,omitempty"`
}

type OdigosAnalyze struct {
	OdigosVersion        properties.EntityProperty `json:"odigosVersion"`
	Tier                 properties.EntityProperty `json:"tier"`
	InstallationMethod   properties.EntityProperty `json:"installationMethod"`
	NumberOfDestinations int                       `json:"numberOfDestinations"`
	NumberOfSources      int                       `json:"numberOfSources"`
	ClusterCollector     ClusterCollectorAnalyze   `json:"clusterCollector"`
	NodeCollector        NodeCollectorAnalyze      `json:"nodeCollector"`

	// is settled is true if all resources are created and ready
	IsSettled bool `json:"isSettled"`
	HasErrors bool `json:"hasErrors"`
}

func analyzeDeployed(cg *odigosv1.CollectorsGroup) (*properties.EntityProperty, *properties.EntityProperty) {
	if cg == nil {
		return nil, nil
	}

	var deployedCondition *metav1.Condition
	for _, condition := range cg.Status.Conditions {
		if condition.Type == "Deployed" {
			deployedCondition = &condition
			break
		}
	}
	if deployedCondition == nil {
		// scheduler created the cg but autoscaler did not reconcile it yet
		return &properties.EntityProperty{
				Name:   "Deployed",
				Value:  false,
				Status: properties.PropertyStatusTransitioning,
				Explain: "deployed means the relevant k8s objects (deployment, configmap, secret, daemonset, etc) were " +
					"created successfully and are expected to start. It does not mean the relevant pods were actually created, started, or are healthy.",
			}, &properties.EntityProperty{
				Name:    "Deployed Error",
				Value:   "waiting for reconciliation",
				Status:  properties.PropertyStatusTransitioning,
				Explain: "in case the deployment resources were not able to be calculated or created, this error will show the reason",
			}
	}

	if deployedCondition.Status == metav1.ConditionTrue {
		// successfully reconciled to collectors deployment
		return &properties.EntityProperty{
			Name:   "Deployed",
			Value:  true,
			Status: properties.PropertyStatusSuccess,
			Explain: "deployed means the relevant k8s objects (deployment, configmap, secret, daemonset, etc) were created" +
				" successfully and are expected to start. It does not mean the relevant pods were actually created, started, or are healthy.",
		}, nil
	} else {
		// had an error during reconciliation to k8s deployment objects
		return &properties.EntityProperty{
				Name:   "Deployed",
				Value:  false,
				Status: properties.PropertyStatusError,
				Explain: "deployed means the relevant k8s objects (deployment, configmap, secret, daemonset, etc) were created" +
					" successfully and are expected to start. It does not mean the relevant pods were actually created, started, or are healthy.",
			}, &properties.EntityProperty{
				Name:    "Deployed Error",
				Value:   deployedCondition.Message,
				Status:  properties.PropertyStatusError,
				Explain: "in case the deployment resources were not able to be calculated or created, this error will show the reason",
			}
	}
}

func analyzeCollectorReady(cg *odigosv1.CollectorsGroup) *properties.EntityProperty {
	if cg == nil {
		return nil
	}

	// TODO: ready is true only once deployment is ready
	// but there is no difference between deployment starting and deployment failed to start
	ready := cg.Status.Ready

	return &properties.EntityProperty{
		Name:    "Ready",
		Value:   ready,
		Status:  properties.GetSuccessOrTransitioning(ready),
		Explain: "ready means that odigos has detected the collectors group as ready to start collecting/receiving data",
	}
}

func analyzeDeployment(dep *appsv1.Deployment, enabled bool) (properties.EntityProperty, *properties.EntityProperty, int) {
	depFound := dep != nil
	deployment := properties.EntityProperty{
		Name:    "Deployment",
		Value:   properties.GetTextCreated(depFound),
		Status:  properties.GetSuccessOrTransitioning(depFound == enabled),
		Explain: "is the k8s deployment object for cluster collector exists in the cluster",
	}
	if !depFound {
		return deployment, nil, 0
	} else {
		expectedReplicas := int(*dep.Spec.Replicas)
		return deployment, &properties.EntityProperty{
			Name:    "Expected Replicas",
			Value:   expectedReplicas,
			Explain: "the number of pods that should be scheduled to run the cluster collector",
		}, expectedReplicas
	}
}

func analyzeDaemonSet(ds *appsv1.DaemonSet, enabled bool) properties.EntityProperty {
	dsFound := ds != nil
	return properties.EntityProperty{
		Name:    "DaemonSet",
		Value:   properties.GetTextCreated(dsFound),
		Status:  properties.GetSuccessOrTransitioning(dsFound == enabled),
		Explain: "is the k8s daemonset object for node collector exists in the cluster",
	}
}

func analyzeDsReplicas(ds *appsv1.DaemonSet) (*properties.EntityProperty, *properties.EntityProperty,
	*properties.EntityProperty, *properties.EntityProperty) {
	if ds == nil {
		return nil, nil, nil, nil
	}

	desiredNodes := int(ds.Status.DesiredNumberScheduled)
	currentReplicas := int(ds.Status.CurrentNumberScheduled)
	updatedReplicas := int(ds.Status.UpdatedNumberScheduled)
	availableNodes := int(ds.Status.NumberAvailable)
	return &properties.EntityProperty{
			// The total number of nodes that should be running this daemon.
			// Regardless of what is actually running (0, 1, or more), rollouts, failures, etc.
			// this number can be less than the number of nodes in the cluster if affinity rules and node selectors are used.
			Name:    "Desired Nodes",
			Value:   desiredNodes,
			Explain: "the number of k8s nodes that should be running the node collector daemonset",
		}, &properties.EntityProperty{
			// The number of nodes that are running at least 1
			// daemon pod and are supposed to run the daemon pod.
			// if this number is less than the desired number, the daemonset is not fully scheduled.
			// it can be due to an active rollout (which is ok), or due to a problem with the nodes / pods
			// this prevents the daemonset pod from being scheduled.
			Name:   "Current Nodes",
			Value:  currentReplicas,
			Status: properties.GetSuccessOrTransitioning(currentReplicas == desiredNodes),
			Explain: "the number of k8s nodes that have at least one pod of the node collector daemonset. this number counts " +
				"the pod objects that were created on this node, regardless of the pod status or revision.",
		}, &properties.EntityProperty{
			// The number of nodes that are running pods from the latest version of the daemonset and do not have old pods from previous versions.
			// if this number is less than the desired number, the daemonset is not fully updated.
			// it can be due to an active rollout (which is ok), or due to a problem with the nodes / pods
			// this prevents the daemonset pod from being updated.
			// this number does not indicate if the pods are indeed running and healthy, only that the only pods scheduled to them is only the latest.
			Name:   "Updated Nodes",
			Value:  updatedReplicas,
			Status: properties.GetSuccessOrTransitioning(updatedReplicas == desiredNodes),
			Explain: "the number of k8s nodes that have only the latest version of the node collector daemonset pods. this " +
				"number counts the pod objects that were created on this node with the latest revision, regardless of the pod status or readiness",
		}, &properties.EntityProperty{
			// available nodes are the nodes for which the oldest pod is ready and available.
			// it can count nodes that are running an old version of the daemonset,
			// so it alone cannot be used to determine if the daemonset is updated and healthy.
			Name:   "Available Nodes",
			Value:  availableNodes,
			Status: properties.GetSuccessOrTransitioning(availableNodes == desiredNodes),
			Explain: "the number of k8s nodes that have at least one pod of the node collector daemonset that is ready and " +
				"available. this number counts the pod objects that were created on this node, regardless of the pod status or revision.",
		}
}

func analyzePodsHealth(pods *corev1.PodList, expectedReplicas int) (*properties.EntityProperty,
	*properties.EntityProperty, *properties.EntityProperty) {
	if pods == nil { // should not happen, but check just in case
		return nil, nil, nil
	}

	runningReplicas := 0
	failureReplicas := 0
	var failureText string
	for j := range pods.Items {
		pod := &pods.Items[j]
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

	healthyReplicas := properties.EntityProperty{
		Name:    "Healthy Replicas",
		Value:   runningReplicas,
		Status:  properties.GetSuccessOrTransitioning(runningReplicas == expectedReplicas),
		Explain: "the number of k8s pods running the updated revision of the cluster collector and are healthy",
	}
	unhealthyReplicas := properties.EntityProperty{
		Name:    "Failed Replicas",
		Value:   failureReplicas,
		Status:  properties.GetSuccessOrError(failureReplicas == 0),
		Explain: "the number of k8s pods running the updated revision of the cluster collector and are not healthy",
	}
	if failureText == "" {
		return &healthyReplicas, &unhealthyReplicas, nil
	} else {
		return &healthyReplicas, &unhealthyReplicas, &properties.EntityProperty{
			Name:    "Failed Replicas Reason",
			Value:   failureText,
			Status:  properties.PropertyStatusError,
			Explain: "if not all replicas are healthy, this error will show the reason",
		}
	}
}

func analyzeClusterCollector(resources *OdigosResources) ClusterCollectorAnalyze {
	isEnabled := len(resources.Destinations.Items) > 0

	enabled := properties.EntityProperty{
		Name:    "Enabled",
		Value:   isEnabled,
		Explain: "should odigos create a cluster collector in the cluster",
	}

	hasCg := resources.ClusterCollector.CollectorsGroup != nil
	cg := properties.EntityProperty{
		Name:    "Collector Group",
		Value:   properties.GetTextCreated(hasCg),
		Status:  properties.GetSuccessOrTransitioning(hasCg == isEnabled),
		Explain: "is the k8s collectors group object for cluster collector exists in the cluster",
	}

	deployed, deployedError := analyzeDeployed(resources.ClusterCollector.CollectorsGroup)
	ready := analyzeCollectorReady(resources.ClusterCollector.CollectorsGroup)
	dep, depExpected, expectedReplicas := analyzeDeployment(resources.ClusterCollector.Deployment, isEnabled)
	healthyPodsCount, failedPodsCount, failedPodsReason := analyzePodsHealth(resources.ClusterCollector.LatestRevisionPods, expectedReplicas)

	return ClusterCollectorAnalyze{
		Enabled:              enabled,
		CollectorGroup:       cg,
		Deployed:             deployed,
		DeployedError:        deployedError,
		CollectorReady:       ready,
		DeploymentCreated:    dep,
		ExpectedReplicas:     depExpected,
		HealthyReplicas:      healthyPodsCount,
		FailedReplicas:       failedPodsCount,
		FailedReplicasReason: failedPodsReason,
	}
}

func analyzeNodeCollector(resources *OdigosResources) NodeCollectorAnalyze {
	hasClusterCollector := resources.ClusterCollector.CollectorsGroup != nil
	isClusterCollectorReady := hasClusterCollector && resources.ClusterCollector.CollectorsGroup.Status.Ready
	hasInstrumentedSources := len(resources.InstrumentationConfigs.Items) > 0
	isEnabled := hasClusterCollector && isClusterCollectorReady && hasInstrumentedSources

	enabled := properties.EntityProperty{
		Name:    "Enabled",
		Value:   isEnabled,
		Explain: "should odigos deploy node collector daemonset in the cluster",
	}

	hasCg := resources.ClusterCollector.CollectorsGroup != nil
	cg := properties.EntityProperty{
		Name:    "Collector Group",
		Value:   properties.GetTextCreated(hasCg),
		Status:  properties.GetSuccessOrTransitioning(hasCg == isEnabled),
		Explain: "is the k8s collectors group object for node collector exists in the cluster",
	}

	deployed, deployedError := analyzeDeployed(resources.ClusterCollector.CollectorsGroup)
	ready := analyzeCollectorReady(resources.ClusterCollector.CollectorsGroup)
	ds := analyzeDaemonSet(resources.NodeCollector.DaemonSet, isEnabled)
	// TODO: implement our oun pod lister to figure out how many are updated and ready which isn't available in the daemonset status
	desiredNodes, currentNodes, updatedNodes, availableNodes := analyzeDsReplicas(resources.NodeCollector.DaemonSet)

	return NodeCollectorAnalyze{
		Enabled:        enabled,
		CollectorGroup: cg,
		Deployed:       deployed,
		DeployedError:  deployedError,
		CollectorReady: ready,
		DaemonSet:      ds,
		DesiredNodes:   desiredNodes,
		CurrentNodes:   currentNodes,
		UpdatedNodes:   updatedNodes,
		AvailableNodes: availableNodes,
	}
}

func summarizeStatus(clusterCollector *ClusterCollectorAnalyze, nodeCollector *NodeCollectorAnalyze) (bool, bool) {
	isSettled := true  // everything is settled, unless we find property with status transitioning
	hasErrors := false // there is no error, unless we find property with status error

	var allProperties = []*properties.EntityProperty{
		&clusterCollector.Enabled,
		&clusterCollector.CollectorGroup,
		clusterCollector.Deployed,
		clusterCollector.DeployedError,
		clusterCollector.CollectorReady,
		&clusterCollector.DeploymentCreated,
		clusterCollector.ExpectedReplicas,
		clusterCollector.HealthyReplicas,
		clusterCollector.FailedReplicas,
		clusterCollector.FailedReplicasReason,
		&nodeCollector.Enabled,
		&nodeCollector.CollectorGroup,
		nodeCollector.Deployed,
		nodeCollector.DeployedError,
		nodeCollector.CollectorReady,
		&nodeCollector.DaemonSet,
		nodeCollector.DesiredNodes,
		nodeCollector.CurrentNodes,
		nodeCollector.UpdatedNodes,
		nodeCollector.AvailableNodes,
	}

	for _, property := range allProperties {
		if property == nil {
			continue
		}
		switch property.Status {
		case properties.PropertyStatusError:
			hasErrors = true
		case properties.PropertyStatusTransitioning:
			isSettled = false
		}
	}

	return isSettled, hasErrors
}

func AnalyzeOdigos(resources *OdigosResources) *OdigosAnalyze {
	clusterCollector := analyzeClusterCollector(resources)
	nodeCollector := analyzeNodeCollector(resources)
	isSettled, hasErrors := summarizeStatus(&clusterCollector, &nodeCollector)

	odigosVersion := resources.OdigosDeployment.Data[k8sconsts.OdigosDeploymentConfigMapVersionKey]
	tier := resources.OdigosDeployment.Data[k8sconsts.OdigosDeploymentConfigMapTierKey]
	installationMethod := resources.OdigosDeployment.Data[k8sconsts.OdigosDeploymentConfigMapInstallationMethodKey]

	odigosVersionProperty := properties.EntityProperty{
		Name:    "Odigos Version",
		Value:   odigosVersion,
		Explain: "the version of odigos deployment currently installed in the cluster",
	}

	odigosTierProperty := properties.EntityProperty{
		Name:    "Tier",
		Value:   tier,
		Explain: "the tier of odigos deployment (community, enterprise, cloud)",
	}

	installationMethodProperty := properties.EntityProperty{
		Name:    "Installation Method",
		Value:   installationMethod,
		Explain: "the method used to deploy odigos in the cluster (helm or odigos cli)",
	}

	return &OdigosAnalyze{
		OdigosVersion:        odigosVersionProperty,
		Tier:                 odigosTierProperty,
		InstallationMethod:   installationMethodProperty,
		NumberOfDestinations: len(resources.Destinations.Items),
		NumberOfSources:      len(resources.InstrumentationConfigs.Items),
		ClusterCollector:     clusterCollector,
		NodeCollector:        nodeCollector,

		IsSettled: isSettled,
		HasErrors: hasErrors,
	}
}
