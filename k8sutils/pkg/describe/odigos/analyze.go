package odigos

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterCollectorAnalyze struct {

	// enable means we should have cluster collector in the cluster
	// cluster collectors are enabled if there are destinations
	Enabled              properties.EntityProperty[bool]    `json:"enabled"`
	CollectorGroup       properties.EntityProperty[string]  `json:"collectorGroup"`
	Deployed             *properties.EntityProperty[bool]   `json:"deployed,omitempty"`
	DeployedError        *properties.EntityProperty[string] `json:"deployedError,omitempty"`
	CollectorReady       *properties.EntityProperty[bool]   `json:"collectorReady,omitempty"`
	DeploymentCreated    properties.EntityProperty[string]  `json:"deployment,omitempty"`
	ExpectedReplicas     *properties.EntityProperty[int]    `json:"expectedReplicas,omitempty"`
	HealthyReplicas      *properties.EntityProperty[int]    `json:"healthyReplicas,omitempty"`
	FailedReplicas       *properties.EntityProperty[int]    `json:"failedReplicas,omitempty"`
	FailedReplicasReason *properties.EntityProperty[string] `json:"failedReplicasReason,omitempty"`
}

type NodeCollectorAnalyze struct {
}

type OdigosAnalyze struct {
	ClusterCollector ClusterCollectorAnalyze `json:"clusterCollector"`
	NodeCollector    NodeCollectorAnalyze    `json:"nodeCollector"`
}

func analyzeDeployed(cg *odigosv1.CollectorsGroup) (*properties.EntityProperty[bool], *properties.EntityProperty[string]) {
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
		return &properties.EntityProperty[bool]{
				Name:   "Deployed",
				Value:  false,
				Status: properties.PropertyStatusTransitioning,
			}, &properties.EntityProperty[string]{
				Name:   "Deployed Error",
				Value:  "waiting for reconciliation",
				Status: properties.PropertyStatusTransitioning,
			}
	}

	if deployedCondition.Status == metav1.ConditionTrue {
		// successfully reconciled to collectors deployment
		return &properties.EntityProperty[bool]{
			Name:   "Deployed",
			Value:  true,
			Status: properties.PropertyStatusSuccess,
		}, nil
	} else {
		// had an error during reconciliation to k8s deployment
		return &properties.EntityProperty[bool]{
				Name:   "Deployed",
				Value:  false,
				Status: properties.PropertyStatusError,
			}, &properties.EntityProperty[string]{
				Name:   "Deployed Error",
				Value:  deployedCondition.Message,
				Status: properties.PropertyStatusError,
			}
	}
}

func analyzeCollectorReady(cg *odigosv1.CollectorsGroup) *properties.EntityProperty[bool] {
	if cg == nil {
		return nil
	}

	// TODO: ready is true only once deployment is ready
	// but there is no difference between deployment starting and deployment failed to start
	ready := cg.Status.Ready

	return &properties.EntityProperty[bool]{
		Name:   "Ready",
		Value:  ready,
		Status: properties.GetSuccessOrTransitioning(ready),
	}
}

func analyzeDeployment(dep *appsv1.Deployment, enabled bool) (properties.EntityProperty[string], *properties.EntityProperty[int], int) {
	depFound := dep != nil
	deployment := properties.EntityProperty[string]{
		Name:   "Deployment",
		Value:  properties.GetTextCreated(depFound),
		Status: properties.GetSuccessOrTransitioning(depFound == enabled),
	}
	if !depFound {
		return deployment, nil, 0
	} else {
		expectedReplicas := int(*dep.Spec.Replicas)
		return deployment, &properties.EntityProperty[int]{
			Name:  "Expected Replicas",
			Value: expectedReplicas,
		}, expectedReplicas
	}
}

func analyzePodsHealth(pods *corev1.PodList, expectedReplicas int) (*properties.EntityProperty[int], *properties.EntityProperty[int], *properties.EntityProperty[string]) {
	if pods == nil { // should not happen, but check just in case
		return nil, nil, nil
	}

	runningReplicas := 0
	failureReplicas := 0
	var failureText string
	for _, pod := range pods.Items {
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

	healthyReplicas := properties.EntityProperty[int]{
		Name:   "Healthy Replicas",
		Value:  runningReplicas,
		Status: properties.GetSuccessOrTransitioning(runningReplicas == expectedReplicas),
	}
	unhealthyReplicas := properties.EntityProperty[int]{
		Name:   "Failed Replicas",
		Value:  failureReplicas,
		Status: properties.GetSuccessOrError(failureReplicas == 0),
	}
	if failureText == "" {
		return &healthyReplicas, &unhealthyReplicas, nil
	} else {
		return &healthyReplicas, &unhealthyReplicas, &properties.EntityProperty[string]{
			Name:   "Failed Replicas Reason",
			Value:  failureText,
			Status: properties.PropertyStatusError,
		}
	}
}

func analyzeClusterCollector(resources *OdigosResources) ClusterCollectorAnalyze {

	isEnabled := len(resources.Destinations.Items) > 0

	enabled := properties.EntityProperty[bool]{
		Name:  "Enabled",
		Value: isEnabled,
		// There is no expected state for this property, so not status is set
	}

	hasCg := resources.ClusterCollector.CollectorsGroup != nil
	cg := properties.EntityProperty[string]{
		Name:   "Collector Group",
		Value:  properties.GetTextCreated(hasCg),
		Status: properties.GetSuccessOrTransitioning(hasCg == isEnabled),
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

func AnalyzeOdigos(resources *OdigosResources) *OdigosAnalyze {
	clusterCollector := analyzeClusterCollector(resources)
	return &OdigosAnalyze{
		ClusterCollector: clusterCollector,
	}
}
