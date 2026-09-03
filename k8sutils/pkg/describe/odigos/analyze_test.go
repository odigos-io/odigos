package odigos

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
)

func collectorsGroupWithConditions(ready bool, conditions ...metav1.Condition) *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		Status: odigosv1.CollectorsGroupStatus{
			Ready:      ready,
			Conditions: conditions,
		},
	}
}

func deploymentWithReplicas(replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
}

func podWithConditions(conditions ...corev1.PodCondition) corev1.Pod {
	return corev1.Pod{
		Status: corev1.PodStatus{
			Conditions: conditions,
		},
	}
}

func TestAnalyzeDeployed(t *testing.T) {
	t.Run("no collectors group", func(t *testing.T) {
		deployed, deployedErr := analyzeDeployed(nil)
		assert.Nil(t, deployed)
		assert.Nil(t, deployedErr)
	})

	t.Run("collectors group not reconciled yet", func(t *testing.T) {
		deployed, deployedErr := analyzeDeployed(collectorsGroupWithConditions(false))

		require.NotNil(t, deployed)
		assert.Equal(t, false, deployed.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, deployed.Status)
		require.NotNil(t, deployedErr)
		assert.Equal(t, "waiting for reconciliation", deployedErr.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, deployedErr.Status)
	})

	t.Run("only unrelated conditions are reported", func(t *testing.T) {
		cg := collectorsGroupWithConditions(true,
			metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue, Message: "collector is ready"},
			metav1.Condition{Type: "TransformedToDeployment", Status: metav1.ConditionTrue},
		)

		deployed, deployedErr := analyzeDeployed(cg)

		require.NotNil(t, deployed)
		assert.Equal(t, false, deployed.Value)
		require.NotNil(t, deployedErr)
		assert.Equal(t, "waiting for reconciliation", deployedErr.Value)
	})

	t.Run("deployed successfully", func(t *testing.T) {
		cg := collectorsGroupWithConditions(true,
			metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Message: "not the deployed condition"},
			metav1.Condition{Type: "Deployed", Status: metav1.ConditionTrue},
		)

		deployed, deployedErr := analyzeDeployed(cg)

		require.NotNil(t, deployed)
		assert.Equal(t, true, deployed.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, deployed.Status)
		assert.Nil(t, deployedErr)
	})

	t.Run("deployment reconciliation failed", func(t *testing.T) {
		cg := collectorsGroupWithConditions(false,
			metav1.Condition{Type: "Deployed", Status: metav1.ConditionFalse, Message: "failed to calculate config"},
		)

		deployed, deployedErr := analyzeDeployed(cg)

		require.NotNil(t, deployed)
		assert.Equal(t, false, deployed.Value)
		assert.Equal(t, properties.PropertyStatusError, deployed.Status)
		require.NotNil(t, deployedErr)
		assert.Equal(t, "failed to calculate config", deployedErr.Value)
		assert.Equal(t, properties.PropertyStatusError, deployedErr.Status)
	})

	t.Run("first deployed condition wins", func(t *testing.T) {
		cg := collectorsGroupWithConditions(false,
			metav1.Condition{Type: "Deployed", Status: metav1.ConditionFalse, Message: "first"},
			metav1.Condition{Type: "Deployed", Status: metav1.ConditionTrue, Message: "second"},
		)

		_, deployedErr := analyzeDeployed(cg)

		require.NotNil(t, deployedErr)
		assert.Equal(t, "first", deployedErr.Value)
	})
}

func TestAnalyzeCollectorReady(t *testing.T) {
	assert.Nil(t, analyzeCollectorReady(nil))

	ready := analyzeCollectorReady(collectorsGroupWithConditions(true))
	require.NotNil(t, ready)
	assert.Equal(t, true, ready.Value)
	assert.Equal(t, properties.PropertyStatusSuccess, ready.Status)

	notReady := analyzeCollectorReady(collectorsGroupWithConditions(false))
	require.NotNil(t, notReady)
	assert.Equal(t, false, notReady.Value)
	assert.Equal(t, properties.PropertyStatusTransitioning, notReady.Status)
}

func TestAnalyzeDeployment(t *testing.T) {
	t.Run("missing while enabled", func(t *testing.T) {
		deployment, expectedReplicasProperty, expectedReplicas := analyzeDeployment(nil, true)

		assert.Equal(t, "not created", deployment.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, deployment.Status)
		assert.Nil(t, expectedReplicasProperty)
		assert.Equal(t, 0, expectedReplicas)
	})

	t.Run("missing while disabled", func(t *testing.T) {
		deployment, expectedReplicasProperty, expectedReplicas := analyzeDeployment(nil, false)

		assert.Equal(t, "not created", deployment.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, deployment.Status)
		assert.Nil(t, expectedReplicasProperty)
		assert.Equal(t, 0, expectedReplicas)
	})

	t.Run("created while enabled", func(t *testing.T) {
		deployment, expectedReplicasProperty, expectedReplicas := analyzeDeployment(deploymentWithReplicas(3), true)

		assert.Equal(t, "created", deployment.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, deployment.Status)
		require.NotNil(t, expectedReplicasProperty)
		assert.Equal(t, 3, expectedReplicasProperty.Value)
		assert.Equal(t, 3, expectedReplicas)
	})

	t.Run("created while disabled", func(t *testing.T) {
		deployment, _, _ := analyzeDeployment(deploymentWithReplicas(1), false)

		assert.Equal(t, "created", deployment.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, deployment.Status)
	})
}

func TestAnalyzeDaemonSet(t *testing.T) {
	missingEnabled := analyzeDaemonSet(nil, true)
	assert.Equal(t, "not created", missingEnabled.Value)
	assert.Equal(t, properties.PropertyStatusTransitioning, missingEnabled.Status)

	missingDisabled := analyzeDaemonSet(nil, false)
	assert.Equal(t, "not created", missingDisabled.Value)
	assert.Equal(t, properties.PropertyStatusSuccess, missingDisabled.Status)

	createdEnabled := analyzeDaemonSet(&appsv1.DaemonSet{}, true)
	assert.Equal(t, "created", createdEnabled.Value)
	assert.Equal(t, properties.PropertyStatusSuccess, createdEnabled.Status)

	createdDisabled := analyzeDaemonSet(&appsv1.DaemonSet{}, false)
	assert.Equal(t, "created", createdDisabled.Value)
	assert.Equal(t, properties.PropertyStatusTransitioning, createdDisabled.Status)
}

func TestAnalyzeDsReplicas(t *testing.T) {
	t.Run("no daemonset", func(t *testing.T) {
		desired, current, updated, available := analyzeDsReplicas(nil)
		assert.Nil(t, desired)
		assert.Nil(t, current)
		assert.Nil(t, updated)
		assert.Nil(t, available)
	})

	// every counter gets a distinct value so that reading the wrong daemonset status field is visible
	t.Run("each counter is read from its own status field", func(t *testing.T) {
		ds := &appsv1.DaemonSet{Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 10,
			CurrentNumberScheduled: 20,
			UpdatedNumberScheduled: 30,
			NumberAvailable:        40,
			NumberReady:            50,
			NumberMisscheduled:     60,
			NumberUnavailable:      70,
		}}

		desired, current, updated, available := analyzeDsReplicas(ds)

		require.NotNil(t, desired)
		assert.Equal(t, 10, desired.Value)
		// the desired count has no expectation to compare against, so it carries no status
		assert.Equal(t, properties.PropertyStatus(""), desired.Status)
		require.NotNil(t, current)
		assert.Equal(t, 20, current.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, current.Status)
		require.NotNil(t, updated)
		assert.Equal(t, 30, updated.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, updated.Status)
		require.NotNil(t, available)
		assert.Equal(t, 40, available.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, available.Status)
	})

	t.Run("fully rolled out daemonset is success", func(t *testing.T) {
		ds := &appsv1.DaemonSet{Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 4,
			CurrentNumberScheduled: 4,
			UpdatedNumberScheduled: 4,
			NumberAvailable:        4,
		}}

		desired, current, updated, available := analyzeDsReplicas(ds)

		assert.Equal(t, 4, desired.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, current.Status)
		assert.Equal(t, properties.PropertyStatusSuccess, updated.Status)
		assert.Equal(t, properties.PropertyStatusSuccess, available.Status)
	})

	// the daemonset pods are scheduled on every node but are not ready yet
	t.Run("only the available count lags", func(t *testing.T) {
		ds := &appsv1.DaemonSet{Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 3,
			CurrentNumberScheduled: 3,
			UpdatedNumberScheduled: 3,
			NumberAvailable:        1,
		}}

		_, current, updated, available := analyzeDsReplicas(ds)

		assert.Equal(t, properties.PropertyStatusSuccess, current.Status)
		assert.Equal(t, properties.PropertyStatusSuccess, updated.Status)
		assert.Equal(t, 1, available.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, available.Status)
	})

	t.Run("mid rollout only the updated count lags", func(t *testing.T) {
		ds := &appsv1.DaemonSet{Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 3,
			CurrentNumberScheduled: 3,
			UpdatedNumberScheduled: 1,
			NumberAvailable:        3,
		}}

		_, current, updated, available := analyzeDsReplicas(ds)

		assert.Equal(t, properties.PropertyStatusSuccess, current.Status)
		assert.Equal(t, properties.PropertyStatusTransitioning, updated.Status)
		assert.Equal(t, properties.PropertyStatusSuccess, available.Status)
	})
}

func TestAnalyzePodsHealth(t *testing.T) {
	t.Run("no pod list", func(t *testing.T) {
		healthy, failed, reason := analyzePodsHealth(nil, 1)
		assert.Nil(t, healthy)
		assert.Nil(t, failed)
		assert.Nil(t, reason)
	})

	t.Run("all pods ready", func(t *testing.T) {
		pods := &corev1.PodList{Items: []corev1.Pod{
			podWithConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}),
			podWithConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}),
		}}

		healthy, failed, reason := analyzePodsHealth(pods, 2)

		require.NotNil(t, healthy)
		assert.Equal(t, 2, healthy.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, healthy.Status)
		require.NotNil(t, failed)
		assert.Equal(t, 0, failed.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, failed.Status)
		assert.Nil(t, reason)
	})

	t.Run("fewer ready pods than expected is transitioning", func(t *testing.T) {
		pods := &corev1.PodList{Items: []corev1.Pod{
			podWithConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}),
		}}

		healthy, failed, reason := analyzePodsHealth(pods, 2)

		assert.Equal(t, 1, healthy.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, healthy.Status)
		assert.Equal(t, 0, failed.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, failed.Status)
		assert.Nil(t, reason)
	})

	t.Run("pod without a ready condition counts as failed without a reason", func(t *testing.T) {
		pods := &corev1.PodList{Items: []corev1.Pod{podWithConditions()}}

		healthy, failed, reason := analyzePodsHealth(pods, 1)

		assert.Equal(t, 0, healthy.Value)
		assert.Equal(t, 1, failed.Value)
		assert.Equal(t, properties.PropertyStatusError, failed.Status)
		assert.Nil(t, reason)
	})

	// only the PodReady condition decides health, a ready ContainersReady is not enough
	t.Run("pod with only a containers ready condition counts as failed", func(t *testing.T) {
		pods := &corev1.PodList{Items: []corev1.Pod{
			podWithConditions(corev1.PodCondition{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}),
		}}

		healthy, failed, _ := analyzePodsHealth(pods, 1)

		assert.Equal(t, 0, healthy.Value)
		assert.Equal(t, 1, failed.Value)
	})

	t.Run("unready pod reports its message as the failure reason", func(t *testing.T) {
		pods := &corev1.PodList{Items: []corev1.Pod{
			podWithConditions(corev1.PodCondition{
				Type:    corev1.PodReady,
				Status:  corev1.ConditionFalse,
				Message: "containers with unready status: [gateway]",
			}),
		}}

		healthy, failed, reason := analyzePodsHealth(pods, 1)

		assert.Equal(t, 0, healthy.Value)
		assert.Equal(t, 1, failed.Value)
		require.NotNil(t, reason)
		assert.Equal(t, "containers with unready status: [gateway]", reason.Value)
		assert.Equal(t, properties.PropertyStatusError, reason.Status)
	})

	t.Run("the last unready pod message is the reported reason", func(t *testing.T) {
		pods := &corev1.PodList{Items: []corev1.Pod{
			podWithConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionFalse, Message: "first failure"}),
			podWithConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}),
			podWithConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionFalse, Message: "last failure"}),
		}}

		healthy, failed, reason := analyzePodsHealth(pods, 3)

		assert.Equal(t, 1, healthy.Value)
		assert.Equal(t, 2, failed.Value)
		require.NotNil(t, reason)
		assert.Equal(t, "last failure", reason.Value)
	})

	t.Run("unready pod with an empty message reports no reason", func(t *testing.T) {
		pods := &corev1.PodList{Items: []corev1.Pod{
			podWithConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionFalse}),
		}}

		_, failed, reason := analyzePodsHealth(pods, 1)

		assert.Equal(t, 1, failed.Value)
		assert.Nil(t, reason)
	})
}

func TestAnalyzeClusterCollector(t *testing.T) {
	t.Run("cluster collector is always enabled so a missing collectors group is transitioning", func(t *testing.T) {
		analyze := analyzeClusterCollector(&OdigosResources{})

		assert.Equal(t, true, analyze.Enabled.Value)
		assert.Equal(t, "not created", analyze.CollectorGroup.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, analyze.CollectorGroup.Status)
		assert.Nil(t, analyze.Deployed)
		assert.Nil(t, analyze.CollectorReady)
		assert.Equal(t, "not created", analyze.DeploymentCreated.Value)
		assert.Nil(t, analyze.ExpectedReplicas)
		assert.Nil(t, analyze.HealthyReplicas)
	})

	t.Run("healthy cluster collector", func(t *testing.T) {
		resources := &OdigosResources{
			ClusterCollector: ClusterCollectorResources{
				CollectorsGroup: collectorsGroupWithConditions(true,
					metav1.Condition{Type: "Deployed", Status: metav1.ConditionTrue}),
				Deployment: deploymentWithReplicas(2),
				LatestRevisionPods: &corev1.PodList{Items: []corev1.Pod{
					podWithConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}),
					podWithConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}),
				}},
			},
		}

		analyze := analyzeClusterCollector(resources)

		assert.Equal(t, properties.PropertyStatusSuccess, analyze.CollectorGroup.Status)
		require.NotNil(t, analyze.Deployed)
		assert.Equal(t, true, analyze.Deployed.Value)
		assert.Nil(t, analyze.DeployedError)
		require.NotNil(t, analyze.CollectorReady)
		assert.Equal(t, true, analyze.CollectorReady.Value)
		require.NotNil(t, analyze.ExpectedReplicas)
		assert.Equal(t, 2, analyze.ExpectedReplicas.Value)
		require.NotNil(t, analyze.HealthyReplicas)
		assert.Equal(t, 2, analyze.HealthyReplicas.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, analyze.HealthyReplicas.Status)
		require.NotNil(t, analyze.FailedReplicas)
		assert.Equal(t, 0, analyze.FailedReplicas.Value)
		assert.Nil(t, analyze.FailedReplicasReason)
	})

	// the healthy replica count is compared against the deployment spec, not against the number of pods found
	t.Run("healthy replicas are compared to the expected replicas of the deployment", func(t *testing.T) {
		resources := &OdigosResources{
			ClusterCollector: ClusterCollectorResources{
				Deployment: deploymentWithReplicas(3),
				LatestRevisionPods: &corev1.PodList{Items: []corev1.Pod{
					podWithConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}),
				}},
			},
		}

		analyze := analyzeClusterCollector(resources)

		assert.Equal(t, 1, analyze.HealthyReplicas.Value)
		assert.Equal(t, properties.PropertyStatusTransitioning, analyze.HealthyReplicas.Status)
	})
}

func TestAnalyzeNodeCollector(t *testing.T) {
	readyCg := func() *odigosv1.CollectorsGroup {
		return collectorsGroupWithConditions(true, metav1.Condition{Type: "Deployed", Status: metav1.ConditionTrue})
	}
	instrumentedSources := func() *odigosv1.InstrumentationConfigList {
		return &odigosv1.InstrumentationConfigList{Items: []odigosv1.InstrumentationConfig{{}}}
	}

	testCases := []struct {
		name            string
		collectorsGroup *odigosv1.CollectorsGroup
		sources         *odigosv1.InstrumentationConfigList
		expectedEnabled bool
	}{
		{
			name:            "no collectors group",
			sources:         instrumentedSources(),
			expectedEnabled: false,
		},
		{
			name:            "collectors group not ready",
			collectorsGroup: collectorsGroupWithConditions(false),
			sources:         instrumentedSources(),
			expectedEnabled: false,
		},
		{
			name:            "no instrumented sources",
			collectorsGroup: readyCg(),
			sources:         &odigosv1.InstrumentationConfigList{},
			expectedEnabled: false,
		},
		{
			name:            "ready collectors group with instrumented sources",
			collectorsGroup: readyCg(),
			sources:         instrumentedSources(),
			expectedEnabled: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resources := &OdigosResources{
				NodeCollector:          NodeCollectorResources{CollectorsGroup: tc.collectorsGroup},
				InstrumentationConfigs: tc.sources,
			}

			analyze := analyzeNodeCollector(resources)

			assert.Equal(t, tc.expectedEnabled, analyze.Enabled.Value)
		})
	}

	t.Run("daemonset and its counters are analyzed when the node collector is enabled", func(t *testing.T) {
		resources := &OdigosResources{
			NodeCollector: NodeCollectorResources{
				CollectorsGroup: readyCg(),
				DaemonSet: &appsv1.DaemonSet{Status: appsv1.DaemonSetStatus{
					DesiredNumberScheduled: 2,
					CurrentNumberScheduled: 2,
					UpdatedNumberScheduled: 2,
					NumberAvailable:        2,
				}},
			},
			InstrumentationConfigs: instrumentedSources(),
		}

		analyze := analyzeNodeCollector(resources)

		assert.Equal(t, properties.PropertyStatusSuccess, analyze.CollectorGroup.Status)
		assert.Equal(t, "created", analyze.DaemonSet.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, analyze.DaemonSet.Status)
		require.NotNil(t, analyze.DesiredNodes)
		assert.Equal(t, 2, analyze.DesiredNodes.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, analyze.AvailableNodes.Status)
	})

	// while the node collector is disabled odigos is not expected to create the daemonset,
	// so a missing daemonset is the settled state
	t.Run("missing daemonset is success while the node collector is disabled", func(t *testing.T) {
		analyze := analyzeNodeCollector(&OdigosResources{
			InstrumentationConfigs: &odigosv1.InstrumentationConfigList{},
		})

		assert.Equal(t, false, analyze.Enabled.Value)
		assert.Equal(t, "not created", analyze.DaemonSet.Value)
		assert.Equal(t, properties.PropertyStatusSuccess, analyze.DaemonSet.Status)
		assert.Equal(t, properties.PropertyStatusSuccess, analyze.CollectorGroup.Status)
		assert.Nil(t, analyze.DesiredNodes)
	})
}

func TestAnalyzePro(t *testing.T) {
	resources := &OdigosResources{
		OdigosDeployment: &corev1.ConfigMap{Data: map[string]string{
			k8sconsts.OdigosDeploymentConfigMapOnPremTokenAudKey:       "odigos-onprem",
			k8sconsts.OdigosDeploymentConfigMapOnPremTokenExpKey:       "2027-01-01T00:00:00Z",
			k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey: "kratos, full-payloads",
		}},
	}

	pro := analyzePro(resources)

	require.NotNil(t, pro)
	assert.Equal(t, "odigos-onprem", pro.OnpremTokenAud.Value)
	assert.Equal(t, "2027-01-01T00:00:00Z", pro.OnpremTokenExpiration.Value)
	assert.Equal(t, "kratos, full-payloads", pro.OdigosProfiles.Value)
}

func TestSummarizeStatus(t *testing.T) {
	// a fully deployed and healthy pipeline: every property either has no status or is successful
	settledAnalyze := func() (*ClusterCollectorAnalyze, *NodeCollectorAnalyze) {
		success := func() properties.EntityProperty {
			return properties.EntityProperty{Status: properties.PropertyStatusSuccess}
		}
		successPtr := func() *properties.EntityProperty {
			p := success()
			return &p
		}
		clusterCollector := &ClusterCollectorAnalyze{
			Enabled:              success(),
			CollectorGroup:       success(),
			Deployed:             successPtr(),
			DeployedError:        successPtr(),
			CollectorReady:       successPtr(),
			DeploymentCreated:    success(),
			ExpectedReplicas:     successPtr(),
			HealthyReplicas:      successPtr(),
			FailedReplicas:       successPtr(),
			FailedReplicasReason: successPtr(),
		}
		nodeCollector := &NodeCollectorAnalyze{
			Enabled:        success(),
			CollectorGroup: success(),
			Deployed:       successPtr(),
			DeployedError:  successPtr(),
			CollectorReady: successPtr(),
			DaemonSet:      success(),
			DesiredNodes:   successPtr(),
			CurrentNodes:   successPtr(),
			UpdatedNodes:   successPtr(),
			AvailableNodes: successPtr(),
		}
		return clusterCollector, nodeCollector
	}

	t.Run("all successful", func(t *testing.T) {
		clusterCollector, nodeCollector := settledAnalyze()

		isSettled, hasErrors := summarizeStatus(clusterCollector, nodeCollector)

		assert.True(t, isSettled)
		assert.False(t, hasErrors)
	})

	t.Run("nil optional properties are skipped", func(t *testing.T) {
		isSettled, hasErrors := summarizeStatus(&ClusterCollectorAnalyze{}, &NodeCollectorAnalyze{})

		assert.True(t, isSettled)
		assert.False(t, hasErrors)
	})

	// every property that the describe output shows must take part in the summary, otherwise
	// `odigos describe` reports a settled and error free pipeline while showing a red/yellow property
	propertySetters := map[string]func(cc *ClusterCollectorAnalyze, nc *NodeCollectorAnalyze, status properties.PropertyStatus){
		"clusterCollector.Enabled": func(cc *ClusterCollectorAnalyze, _ *NodeCollectorAnalyze, s properties.PropertyStatus) {
			cc.Enabled.Status = s
		},
		"clusterCollector.CollectorGroup": func(cc *ClusterCollectorAnalyze, _ *NodeCollectorAnalyze, s properties.PropertyStatus) {
			cc.CollectorGroup.Status = s
		},
		"clusterCollector.Deployed": func(cc *ClusterCollectorAnalyze, _ *NodeCollectorAnalyze, s properties.PropertyStatus) {
			cc.Deployed.Status = s
		},
		"clusterCollector.DeployedError": func(cc *ClusterCollectorAnalyze, _ *NodeCollectorAnalyze, s properties.PropertyStatus) {
			cc.DeployedError.Status = s
		},
		"clusterCollector.CollectorReady": func(cc *ClusterCollectorAnalyze, _ *NodeCollectorAnalyze, s properties.PropertyStatus) {
			cc.CollectorReady.Status = s
		},
		"clusterCollector.DeploymentCreated": func(cc *ClusterCollectorAnalyze, _ *NodeCollectorAnalyze, s properties.PropertyStatus) {
			cc.DeploymentCreated.Status = s
		},
		"clusterCollector.ExpectedReplicas": func(cc *ClusterCollectorAnalyze, _ *NodeCollectorAnalyze, s properties.PropertyStatus) {
			cc.ExpectedReplicas.Status = s
		},
		"clusterCollector.HealthyReplicas": func(cc *ClusterCollectorAnalyze, _ *NodeCollectorAnalyze, s properties.PropertyStatus) {
			cc.HealthyReplicas.Status = s
		},
		"clusterCollector.FailedReplicas": func(cc *ClusterCollectorAnalyze, _ *NodeCollectorAnalyze, s properties.PropertyStatus) {
			cc.FailedReplicas.Status = s
		},
		"clusterCollector.FailedReplicasReason": func(cc *ClusterCollectorAnalyze, _ *NodeCollectorAnalyze, s properties.PropertyStatus) {
			cc.FailedReplicasReason.Status = s
		},
		"nodeCollector.Enabled": func(_ *ClusterCollectorAnalyze, nc *NodeCollectorAnalyze, s properties.PropertyStatus) {
			nc.Enabled.Status = s
		},
		"nodeCollector.CollectorGroup": func(_ *ClusterCollectorAnalyze, nc *NodeCollectorAnalyze, s properties.PropertyStatus) {
			nc.CollectorGroup.Status = s
		},
		"nodeCollector.Deployed": func(_ *ClusterCollectorAnalyze, nc *NodeCollectorAnalyze, s properties.PropertyStatus) {
			nc.Deployed.Status = s
		},
		"nodeCollector.DeployedError": func(_ *ClusterCollectorAnalyze, nc *NodeCollectorAnalyze, s properties.PropertyStatus) {
			nc.DeployedError.Status = s
		},
		"nodeCollector.CollectorReady": func(_ *ClusterCollectorAnalyze, nc *NodeCollectorAnalyze, s properties.PropertyStatus) {
			nc.CollectorReady.Status = s
		},
		"nodeCollector.DaemonSet": func(_ *ClusterCollectorAnalyze, nc *NodeCollectorAnalyze, s properties.PropertyStatus) {
			nc.DaemonSet.Status = s
		},
		"nodeCollector.DesiredNodes": func(_ *ClusterCollectorAnalyze, nc *NodeCollectorAnalyze, s properties.PropertyStatus) {
			nc.DesiredNodes.Status = s
		},
		"nodeCollector.CurrentNodes": func(_ *ClusterCollectorAnalyze, nc *NodeCollectorAnalyze, s properties.PropertyStatus) {
			nc.CurrentNodes.Status = s
		},
		"nodeCollector.UpdatedNodes": func(_ *ClusterCollectorAnalyze, nc *NodeCollectorAnalyze, s properties.PropertyStatus) {
			nc.UpdatedNodes.Status = s
		},
		"nodeCollector.AvailableNodes": func(_ *ClusterCollectorAnalyze, nc *NodeCollectorAnalyze, s properties.PropertyStatus) {
			nc.AvailableNodes.Status = s
		},
	}

	for name, setStatus := range propertySetters {
		t.Run(name+" error is reported", func(t *testing.T) {
			clusterCollector, nodeCollector := settledAnalyze()
			setStatus(clusterCollector, nodeCollector, properties.PropertyStatusError)

			isSettled, hasErrors := summarizeStatus(clusterCollector, nodeCollector)

			assert.True(t, hasErrors)
			assert.True(t, isSettled)
		})

		t.Run(name+" transitioning is reported", func(t *testing.T) {
			clusterCollector, nodeCollector := settledAnalyze()
			setStatus(clusterCollector, nodeCollector, properties.PropertyStatusTransitioning)

			isSettled, hasErrors := summarizeStatus(clusterCollector, nodeCollector)

			assert.False(t, isSettled)
			assert.False(t, hasErrors)
		})
	}
}

func TestAnalyzeOdigos(t *testing.T) {
	t.Run("deployment metadata and counts", func(t *testing.T) {
		resources := &OdigosResources{
			OdigosDeployment: &corev1.ConfigMap{Data: map[string]string{
				k8sconsts.OdigosDeploymentConfigMapVersionKey:            "v1.22.0",
				k8sconsts.OdigosDeploymentConfigMapTierKey:               string(common.CommunityOdigosTier),
				k8sconsts.OdigosDeploymentConfigMapInstallationMethodKey: "helm",
				k8sconsts.OdigosDeploymentConfigMapKubernetesVersionKey:  "v1.31.2",
			}},
			Destinations: &odigosv1.DestinationList{Items: []odigosv1.Destination{{}, {}, {}}},
			InstrumentationConfigs: &odigosv1.InstrumentationConfigList{
				Items: []odigosv1.InstrumentationConfig{{}, {}},
			},
		}

		analyze := AnalyzeOdigos(resources)

		assert.Equal(t, "v1.22.0", analyze.OdigosVersion.Value)
		assert.Equal(t, "v1.31.2", analyze.KubernetesVersion.Value)
		assert.Equal(t, string(common.CommunityOdigosTier), analyze.Tier.Value)
		assert.Equal(t, "helm", analyze.InstallationMethod.Value)
		assert.Equal(t, 3, analyze.NumberOfDestinations)
		assert.Equal(t, 2, analyze.NumberOfSources)
	})

	t.Run("odigos pro is reported only for the onprem tier", func(t *testing.T) {
		for _, tier := range []common.OdigosTier{common.CommunityOdigosTier, common.CloudOdigosTier, common.OnPremOdigosTier} {
			resources := &OdigosResources{
				OdigosDeployment: &corev1.ConfigMap{Data: map[string]string{
					k8sconsts.OdigosDeploymentConfigMapTierKey:           string(tier),
					k8sconsts.OdigosDeploymentConfigMapOnPremTokenAudKey: "odigos-onprem",
				}},
				Destinations:           &odigosv1.DestinationList{},
				InstrumentationConfigs: &odigosv1.InstrumentationConfigList{},
			}

			analyze := AnalyzeOdigos(resources)

			if tier == common.OnPremOdigosTier {
				require.NotNil(t, analyze.OdigosPro, "tier %s", tier)
				assert.Equal(t, "odigos-onprem", analyze.OdigosPro.OnpremTokenAud.Value)
			} else {
				assert.Nil(t, analyze.OdigosPro, "tier %s", tier)
			}
		}
	})

	t.Run("a freshly installed pipeline is not settled", func(t *testing.T) {
		resources := &OdigosResources{
			OdigosDeployment:       &corev1.ConfigMap{},
			Destinations:           &odigosv1.DestinationList{},
			InstrumentationConfigs: &odigosv1.InstrumentationConfigList{},
		}

		analyze := AnalyzeOdigos(resources)

		assert.False(t, analyze.IsSettled)
		assert.False(t, analyze.HasErrors)
	})

	t.Run("a failed cluster collector deployment is reported as an error", func(t *testing.T) {
		resources := &OdigosResources{
			OdigosDeployment: &corev1.ConfigMap{},
			ClusterCollector: ClusterCollectorResources{
				CollectorsGroup: collectorsGroupWithConditions(false,
					metav1.Condition{Type: "Deployed", Status: metav1.ConditionFalse, Message: "no destinations configured"}),
				Deployment:         deploymentWithReplicas(1),
				LatestRevisionPods: &corev1.PodList{},
			},
			Destinations:           &odigosv1.DestinationList{},
			InstrumentationConfigs: &odigosv1.InstrumentationConfigList{},
		}

		analyze := AnalyzeOdigos(resources)

		assert.True(t, analyze.HasErrors)
		assert.False(t, analyze.IsSettled)
		require.NotNil(t, analyze.ClusterCollector.DeployedError)
		assert.Equal(t, "no destinations configured", analyze.ClusterCollector.DeployedError.Value)
	})

	t.Run("a fully healthy pipeline is settled without errors", func(t *testing.T) {
		resources := &OdigosResources{
			OdigosDeployment: &corev1.ConfigMap{},
			ClusterCollector: ClusterCollectorResources{
				CollectorsGroup: collectorsGroupWithConditions(true,
					metav1.Condition{Type: "Deployed", Status: metav1.ConditionTrue}),
				Deployment: deploymentWithReplicas(1),
				LatestRevisionPods: &corev1.PodList{Items: []corev1.Pod{
					podWithConditions(corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}),
				}},
			},
			NodeCollector: NodeCollectorResources{
				CollectorsGroup: collectorsGroupWithConditions(true,
					metav1.Condition{Type: "Deployed", Status: metav1.ConditionTrue}),
				DaemonSet: &appsv1.DaemonSet{Status: appsv1.DaemonSetStatus{
					DesiredNumberScheduled: 1,
					CurrentNumberScheduled: 1,
					UpdatedNumberScheduled: 1,
					NumberAvailable:        1,
				}},
			},
			Destinations: &odigosv1.DestinationList{Items: []odigosv1.Destination{{}}},
			InstrumentationConfigs: &odigosv1.InstrumentationConfigList{
				Items: []odigosv1.InstrumentationConfig{{}},
			},
		}

		analyze := AnalyzeOdigos(resources)

		assert.True(t, analyze.IsSettled)
		assert.False(t, analyze.HasErrors)
	})
}
