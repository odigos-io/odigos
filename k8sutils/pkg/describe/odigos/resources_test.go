package odigos

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

const describeTestNs = "odigos-system"

func odigosDeploymentConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosDeploymentConfigMapName,
			Namespace: describeTestNs,
		},
		Data: map[string]string{
			k8sconsts.OdigosDeploymentConfigMapVersionKey: "v1.22.0",
		},
	}
}

func gatewayDeployment(revision string) *appsv1.Deployment {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorDeploymentName,
			Namespace: describeTestNs,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": k8sconsts.OdigosClusterCollectorDeploymentName},
			},
		},
	}
	if revision != "" {
		dep.Annotations = map[string]string{"deployment.kubernetes.io/revision": revision}
	}
	return dep
}

func gatewayReplicaSet(name string, revision string, podTemplateHash string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   describeTestNs,
			Annotations: map[string]string{"deployment.kubernetes.io/revision": revision},
			Labels: map[string]string{
				"app":               k8sconsts.OdigosClusterCollectorDeploymentName,
				"pod-template-hash": podTemplateHash,
			},
		},
	}
}

func gatewayPod(name string, podTemplateHash string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: describeTestNs,
			Labels: map[string]string{
				"app":               k8sconsts.OdigosClusterCollectorDeploymentName,
				"pod-template-hash": podTemplateHash,
			},
		},
	}
}

func podNames(pods *corev1.PodList) []string {
	names := make([]string, 0, len(pods.Items))
	for i := range pods.Items {
		names = append(names, pods.Items[i].GetName())
	}
	return names
}

func TestGetClusterCollectorResources(t *testing.T) {
	t.Run("nothing deployed yet", func(t *testing.T) {
		clusterCollector, err := getClusterCollectorResources(context.Background(),
			fake.NewSimpleClientset(), odigosfake.NewSimpleClientset().OdigosV1alpha1(), describeTestNs)

		require.NoError(t, err)
		assert.Nil(t, clusterCollector.CollectorsGroup)
		assert.Nil(t, clusterCollector.Deployment)
		assert.Nil(t, clusterCollector.LatestRevisionPods)
	})

	t.Run("collectors group and deployment are resolved by their well known names", func(t *testing.T) {
		cg := &odigosv1.CollectorsGroup{ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorCollectorGroupName,
			Namespace: describeTestNs,
		}}
		// the node collector objects live in the same namespace and must not be picked up here
		nodeCg := &odigosv1.CollectorsGroup{ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorCollectorGroupName,
			Namespace: describeTestNs,
		}}

		clusterCollector, err := getClusterCollectorResources(context.Background(),
			fake.NewSimpleClientset(gatewayDeployment("")),
			odigosfake.NewSimpleClientset(cg, nodeCg).OdigosV1alpha1(), describeTestNs)

		require.NoError(t, err)
		require.NotNil(t, clusterCollector.CollectorsGroup)
		assert.Equal(t, k8sconsts.OdigosClusterCollectorCollectorGroupName, clusterCollector.CollectorsGroup.GetName())
		require.NotNil(t, clusterCollector.Deployment)
		assert.Equal(t, k8sconsts.OdigosClusterCollectorDeploymentName, clusterCollector.Deployment.GetName())
		// without a revision annotation there is no way to tell which replicaset is the latest
		assert.Nil(t, clusterCollector.LatestRevisionPods)
	})

	t.Run("only the pods of the latest revision replicaset are collected", func(t *testing.T) {
		// another odigos component in the same namespace, also at revision 2, must not be
		// mistaken for the cluster collector replicaset
		otherComponentReplicaSet := gatewayReplicaSet("a-other-component", "2", "otherhash")
		otherComponentReplicaSet.Labels["app"] = "odigos-ui"
		otherComponentPod := gatewayPod("a-other-component-1", "otherhash")
		otherComponentPod.Labels["app"] = "odigos-ui"

		kubeClient := fake.NewSimpleClientset(
			gatewayDeployment("2"),
			otherComponentReplicaSet,
			otherComponentPod,
			gatewayReplicaSet("odigos-gateway-old", "1", "oldhash"),
			gatewayReplicaSet("odigos-gateway-new", "2", "newhash"),
			gatewayPod("odigos-gateway-old-1", "oldhash"),
			gatewayPod("odigos-gateway-new-1", "newhash"),
			gatewayPod("odigos-gateway-new-2", "newhash"),
		)

		clusterCollector, err := getClusterCollectorResources(context.Background(),
			kubeClient, odigosfake.NewSimpleClientset().OdigosV1alpha1(), describeTestNs)

		require.NoError(t, err)
		require.NotNil(t, clusterCollector.LatestRevisionPods)
		assert.ElementsMatch(t, []string{"odigos-gateway-new-1", "odigos-gateway-new-2"},
			podNames(clusterCollector.LatestRevisionPods))
	})

	t.Run("no replicaset matches the deployment revision", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(
			gatewayDeployment("3"),
			gatewayReplicaSet("odigos-gateway-old", "1", "oldhash"),
			gatewayPod("odigos-gateway-old-1", "oldhash"),
		)

		clusterCollector, err := getClusterCollectorResources(context.Background(),
			kubeClient, odigosfake.NewSimpleClientset().OdigosV1alpha1(), describeTestNs)

		require.NoError(t, err)
		assert.Nil(t, clusterCollector.LatestRevisionPods)
	})

	t.Run("collectors group get failure is propagated", func(t *testing.T) {
		odigosClient := odigosfake.NewSimpleClientset()
		odigosClient.PrependReactor("get", "collectorsgroups",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("collectorsgroups is forbidden")
			})

		_, err := getClusterCollectorResources(context.Background(),
			fake.NewSimpleClientset(), odigosClient.OdigosV1alpha1(), describeTestNs)

		require.ErrorContains(t, err, "collectorsgroups is forbidden")
	})

	t.Run("deployment get failure is propagated", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset()
		kubeClient.PrependReactor("get", "deployments",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("deployments is forbidden")
			})

		_, err := getClusterCollectorResources(context.Background(),
			kubeClient, odigosfake.NewSimpleClientset().OdigosV1alpha1(), describeTestNs)

		require.ErrorContains(t, err, "deployments is forbidden")
	})

	t.Run("replicaset list failure is propagated", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(gatewayDeployment("2"))
		kubeClient.PrependReactor("list", "replicasets",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("replicasets is forbidden")
			})

		_, err := getClusterCollectorResources(context.Background(),
			kubeClient, odigosfake.NewSimpleClientset().OdigosV1alpha1(), describeTestNs)

		require.ErrorContains(t, err, "replicasets is forbidden")
	})

	t.Run("latest revision pod list failure is propagated", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(
			gatewayDeployment("2"),
			gatewayReplicaSet("odigos-gateway-new", "2", "newhash"),
		)
		kubeClient.PrependReactor("list", "pods",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("pods is forbidden")
			})

		_, err := getClusterCollectorResources(context.Background(),
			kubeClient, odigosfake.NewSimpleClientset().OdigosV1alpha1(), describeTestNs)

		require.ErrorContains(t, err, "pods is forbidden")
	})
}

func TestGetNodeCollectorResources(t *testing.T) {
	t.Run("nothing deployed yet", func(t *testing.T) {
		nodeCollector, err := getNodeCollectorResources(context.Background(),
			fake.NewSimpleClientset(), odigosfake.NewSimpleClientset().OdigosV1alpha1(), describeTestNs)

		require.NoError(t, err)
		assert.Nil(t, nodeCollector.CollectorsGroup)
		assert.Nil(t, nodeCollector.DaemonSet)
	})

	t.Run("collectors group and daemonset are resolved by their well known names", func(t *testing.T) {
		cg := &odigosv1.CollectorsGroup{ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorCollectorGroupName,
			Namespace: describeTestNs,
		}}
		clusterCg := &odigosv1.CollectorsGroup{ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorCollectorGroupName,
			Namespace: describeTestNs,
		}}
		ds := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorDaemonSetName,
			Namespace: describeTestNs,
		}}

		nodeCollector, err := getNodeCollectorResources(context.Background(),
			fake.NewSimpleClientset(ds), odigosfake.NewSimpleClientset(cg, clusterCg).OdigosV1alpha1(), describeTestNs)

		require.NoError(t, err)
		require.NotNil(t, nodeCollector.CollectorsGroup)
		assert.Equal(t, k8sconsts.OdigosNodeCollectorCollectorGroupName, nodeCollector.CollectorsGroup.GetName())
		require.NotNil(t, nodeCollector.DaemonSet)
		assert.Equal(t, k8sconsts.OdigosNodeCollectorDaemonSetName, nodeCollector.DaemonSet.GetName())
	})

	t.Run("daemonset get failure is propagated", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset()
		kubeClient.PrependReactor("get", "daemonsets",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("daemonsets is forbidden")
			})

		_, err := getNodeCollectorResources(context.Background(),
			kubeClient, odigosfake.NewSimpleClientset().OdigosV1alpha1(), describeTestNs)

		require.ErrorContains(t, err, "daemonsets is forbidden")
	})

	t.Run("collectors group get failure is propagated", func(t *testing.T) {
		odigosClient := odigosfake.NewSimpleClientset()
		odigosClient.PrependReactor("get", "collectorsgroups",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("collectorsgroups is forbidden")
			})

		_, err := getNodeCollectorResources(context.Background(),
			fake.NewSimpleClientset(), odigosClient.OdigosV1alpha1(), describeTestNs)

		require.ErrorContains(t, err, "collectorsgroups is forbidden")
	})
}

func TestGetRelevantOdigosResources(t *testing.T) {
	t.Run("the odigos deployment config map is mandatory", func(t *testing.T) {
		_, err := GetRelevantOdigosResources(context.Background(),
			fake.NewSimpleClientset(), odigosfake.NewSimpleClientset().OdigosV1alpha1(), describeTestNs)

		require.Error(t, err)
		assert.ErrorContains(t, err, k8sconsts.OdigosDeploymentConfigMapName)
	})

	t.Run("destinations are namespaced while instrumentation configs are cluster wide", func(t *testing.T) {
		destinationInOdigosNs := &odigosv1.Destination{ObjectMeta: metav1.ObjectMeta{
			Name: "jaeger", Namespace: describeTestNs,
		}}
		destinationInOtherNs := &odigosv1.Destination{ObjectMeta: metav1.ObjectMeta{
			Name: "tempo", Namespace: "other",
		}}
		icInDefault := &odigosv1.InstrumentationConfig{ObjectMeta: metav1.ObjectMeta{
			Name: "deployment-frontend", Namespace: "default",
		}}
		icInOther := &odigosv1.InstrumentationConfig{ObjectMeta: metav1.ObjectMeta{
			Name: "deployment-backend", Namespace: "other",
		}}

		resources, err := GetRelevantOdigosResources(context.Background(),
			fake.NewSimpleClientset(odigosDeploymentConfigMap()),
			odigosfake.NewSimpleClientset(destinationInOdigosNs, destinationInOtherNs, icInDefault, icInOther).OdigosV1alpha1(),
			describeTestNs)

		require.NoError(t, err)
		require.NotNil(t, resources.OdigosDeployment)
		assert.Equal(t, "v1.22.0", resources.OdigosDeployment.Data[k8sconsts.OdigosDeploymentConfigMapVersionKey])
		assert.Len(t, resources.Destinations.Items, 1)
		assert.Equal(t, "jaeger", resources.Destinations.Items[0].GetName())
		assert.Len(t, resources.InstrumentationConfigs.Items, 2)
	})

	t.Run("cluster collector resolution failure is propagated", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(odigosDeploymentConfigMap())
		kubeClient.PrependReactor("get", "deployments",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("deployments is forbidden")
			})

		_, err := GetRelevantOdigosResources(context.Background(),
			kubeClient, odigosfake.NewSimpleClientset().OdigosV1alpha1(), describeTestNs)

		require.ErrorContains(t, err, "deployments is forbidden")
	})

	t.Run("node collector resolution failure is propagated", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(odigosDeploymentConfigMap())
		kubeClient.PrependReactor("get", "daemonsets",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("daemonsets is forbidden")
			})

		_, err := GetRelevantOdigosResources(context.Background(),
			kubeClient, odigosfake.NewSimpleClientset().OdigosV1alpha1(), describeTestNs)

		require.ErrorContains(t, err, "daemonsets is forbidden")
	})

	t.Run("destination list failure is propagated", func(t *testing.T) {
		odigosClient := odigosfake.NewSimpleClientset()
		odigosClient.PrependReactor("list", "destinations",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("destinations is forbidden")
			})

		_, err := GetRelevantOdigosResources(context.Background(),
			fake.NewSimpleClientset(odigosDeploymentConfigMap()), odigosClient.OdigosV1alpha1(), describeTestNs)

		require.ErrorContains(t, err, "destinations is forbidden")
	})

	t.Run("instrumentation config list failure is propagated", func(t *testing.T) {
		odigosClient := odigosfake.NewSimpleClientset()
		odigosClient.PrependReactor("list", "instrumentationconfigs",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("instrumentationconfigs is forbidden")
			})

		_, err := GetRelevantOdigosResources(context.Background(),
			fake.NewSimpleClientset(odigosDeploymentConfigMap()), odigosClient.OdigosV1alpha1(), describeTestNs)

		require.ErrorContains(t, err, "instrumentationconfigs is forbidden")
	})
}
