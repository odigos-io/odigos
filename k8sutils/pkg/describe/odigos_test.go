package describe

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	odigosdescribe "github.com/odigos-io/odigos/k8sutils/pkg/describe/odigos"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
)

const describeTestNs = "odigos-system"

func odigosAnalyzeProperty(name string, value interface{}, status properties.PropertyStatus) properties.EntityProperty {
	return properties.EntityProperty{Name: name, Value: value, Status: status}
}

func TestDescribeOdigosToText(t *testing.T) {
	analyze := &odigosdescribe.OdigosAnalyze{
		OdigosVersion:        odigosAnalyzeProperty("Odigos Version", "v1.22.0", ""),
		KubernetesVersion:    odigosAnalyzeProperty("Kubernetes Version", "v1.31.2", ""),
		Tier:                 odigosAnalyzeProperty("Tier", "onprem", ""),
		InstallationMethod:   odigosAnalyzeProperty("Installation Method", "helm", ""),
		NumberOfDestinations: 2,
		NumberOfSources:      3,
		ClusterCollector: odigosdescribe.ClusterCollectorAnalyze{
			Enabled:           odigosAnalyzeProperty("Enabled", true, ""),
			CollectorGroup:    odigosAnalyzeProperty("Collector Group", "created", properties.PropertyStatusSuccess),
			DeploymentCreated: odigosAnalyzeProperty("Deployment", "created", properties.PropertyStatusSuccess),
		},
		NodeCollector: odigosdescribe.NodeCollectorAnalyze{
			Enabled:        odigosAnalyzeProperty("Enabled", false, ""),
			CollectorGroup: odigosAnalyzeProperty("Collector Group", "not created", properties.PropertyStatusSuccess),
			DaemonSet:      odigosAnalyzeProperty("DaemonSet", "not created", properties.PropertyStatusSuccess),
		},
	}

	text := DescribeOdigosToText(analyze)

	assert.Contains(t, text, "Odigos Version: v1.22.0")
	assert.Contains(t, text, "Kubernetes Version: v1.31.2")
	assert.Contains(t, text, "Tier: onprem")
	assert.Contains(t, text, "Installation Method: helm")
	assert.Contains(t, text, "Odigos Pipeline:")
	assert.Contains(t, text, "Status: there are 3 sources and 2 destinations")
	assert.Contains(t, text, "Cluster Collector:")
	assert.Contains(t, text, "Node Collector:")
	assert.Contains(t, text, greenPrefix+"Collector Group: created"+colorSuffix)
	assert.Contains(t, text, greenPrefix+"DaemonSet: not created"+colorSuffix)
	// the optional properties of a pipeline that was not reconciled yet are omitted
	assert.NotContains(t, text, "Deployed")
	assert.NotContains(t, text, "Expected Replicas")
	assert.NotContains(t, text, "Desired Nodes")
	// odigos pro is printed only when the analysis holds it
	assert.NotContains(t, text, "Odigos Pro:")

	// the cluster collector section comes before the node collector section
	assert.Less(t, strings.Index(text, "Cluster Collector:"), strings.Index(text, "Node Collector:"))
}

func TestDescribeOdigosToTextWithOptionalProperties(t *testing.T) {
	expectedReplicas := odigosAnalyzeProperty("Expected Replicas", 1, "")
	failedReplicas := odigosAnalyzeProperty("Failed Replicas", 1, properties.PropertyStatusError)
	failedReason := odigosAnalyzeProperty("Failed Replicas Reason", "back-off restarting failed container", properties.PropertyStatusError)
	desiredNodes := odigosAnalyzeProperty("Desired Nodes", 3, "")
	currentNodes := odigosAnalyzeProperty("Current Nodes", 2, properties.PropertyStatusTransitioning)
	analyze := &odigosdescribe.OdigosAnalyze{
		ClusterCollector: odigosdescribe.ClusterCollectorAnalyze{
			ExpectedReplicas:     &expectedReplicas,
			FailedReplicas:       &failedReplicas,
			FailedReplicasReason: &failedReason,
		},
		NodeCollector: odigosdescribe.NodeCollectorAnalyze{
			DesiredNodes: &desiredNodes,
			CurrentNodes: &currentNodes,
		},
		OdigosPro: &odigosdescribe.OdigosProAnalyze{
			OnpremTokenAud:        odigosAnalyzeProperty("OnPrem Token Audience", "odigos-onprem", ""),
			OnpremTokenExpiration: odigosAnalyzeProperty("OnPrem Token Expiration Date", "2027-01-01", ""),
			OdigosProfiles:        odigosAnalyzeProperty("OnPrem Client Profiles", "kratos", ""),
		},
	}

	text := DescribeOdigosToText(analyze)

	assert.Contains(t, text, "Odigos Pro:")
	assert.Contains(t, text, "OnPrem Token Audience: odigos-onprem")
	assert.Contains(t, text, "OnPrem Token Expiration Date: 2027-01-01")
	assert.Contains(t, text, "OnPrem Client Profiles: kratos")
	assert.Contains(t, text, "Expected Replicas: 1")
	assert.Contains(t, text, redPrefix+"Failed Replicas: 1"+colorSuffix)
	assert.Contains(t, text, redPrefix+"Failed Replicas Reason: back-off restarting failed container"+colorSuffix)
	assert.Contains(t, text, "Desired Nodes: 3")
	assert.Contains(t, text, yellowPrefix+"Current Nodes: 2"+colorSuffix)
	// the pro section is printed before the pipeline section
	assert.Less(t, strings.Index(text, "Odigos Pro:"), strings.Index(text, "Odigos Pipeline:"))
}

func TestDescribeOdigos(t *testing.T) {
	odigosDeployment := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosDeploymentConfigMapName,
			Namespace: describeTestNs,
		},
		Data: map[string]string{
			k8sconsts.OdigosDeploymentConfigMapVersionKey: "v1.22.0",
			k8sconsts.OdigosDeploymentConfigMapTierKey:    string(common.CommunityOdigosTier),
		},
	}

	t.Run("a healthy installation", func(t *testing.T) {
		clusterCollectorGroup := &odigosv1.CollectorsGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      k8sconsts.OdigosClusterCollectorCollectorGroupName,
				Namespace: describeTestNs,
			},
			Status: odigosv1.CollectorsGroupStatus{
				Ready:      true,
				Conditions: []metav1.Condition{{Type: "Deployed", Status: metav1.ConditionTrue}},
			},
		}
		replicas := int32(1)
		gateway := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      k8sconsts.OdigosClusterCollectorDeploymentName,
				Namespace: describeTestNs,
			},
			Spec: appsv1.DeploymentSpec{Replicas: &replicas},
		}

		analyze, err := DescribeOdigos(context.Background(),
			fake.NewSimpleClientset(odigosDeployment, gateway),
			odigosfake.NewSimpleClientset(clusterCollectorGroup).OdigosV1alpha1(), describeTestNs)

		require.NoError(t, err)
		assert.Equal(t, "v1.22.0", analyze.OdigosVersion.Value)
		assert.Equal(t, string(common.CommunityOdigosTier), analyze.Tier.Value)
		assert.Nil(t, analyze.OdigosPro)
		require.NotNil(t, analyze.ClusterCollector.Deployed)
		assert.Equal(t, true, analyze.ClusterCollector.Deployed.Value)
		assert.Equal(t, "created", analyze.ClusterCollector.DeploymentCreated.Value)
		assert.False(t, analyze.HasErrors)

		text := DescribeOdigosToText(analyze)
		assert.Contains(t, text, "Odigos Version: v1.22.0")
		assert.Contains(t, text, greenPrefix+"Deployed: true"+colorSuffix)
	})

	t.Run("a missing odigos deployment config map is an error", func(t *testing.T) {
		_, err := DescribeOdigos(context.Background(), fake.NewSimpleClientset(),
			odigosfake.NewSimpleClientset().OdigosV1alpha1(), describeTestNs)

		require.Error(t, err)
		assert.ErrorContains(t, err, k8sconsts.OdigosDeploymentConfigMapName)
	})
}
