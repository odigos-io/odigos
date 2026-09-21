package source

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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

const (
	sourceTestNs         = "shop"
	sourceTestDeployUID  = types.UID("deployment-uid")
	sourceTestWorkload   = "checkout"
	sourceTestPodLabel   = "checkout"
	sourceTestOtherLabel = "other"
)

func workloadObject(kind k8sconsts.WorkloadKind, labelSelector *metav1.LabelSelector) *K8sSourceObject {
	return &K8sSourceObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sourceTestWorkload,
			Namespace: sourceTestNs,
			UID:       sourceTestDeployUID,
		},
		Kind:          kind,
		LabelSelector: labelSelector,
	}
}

func appLabelSelector(app string) *metav1.LabelSelector {
	return &metav1.LabelSelector{MatchLabels: map[string]string{"app": app}}
}

func deploymentReplicaSet(name string, revision string, podTemplateHash string, owner metav1.OwnerReference) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       sourceTestNs,
			Annotations:     map[string]string{"deployment.kubernetes.io/revision": revision},
			Labels:          map[string]string{"app": sourceTestPodLabel},
			OwnerReferences: []metav1.OwnerReference{owner},
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app":               sourceTestPodLabel,
				"pod-template-hash": podTemplateHash,
			}},
		},
	}
}

func replicaSetPod(name string, podTemplateHash string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: sourceTestNs,
			Labels: map[string]string{
				"app":               sourceTestPodLabel,
				"pod-template-hash": podTemplateHash,
			},
		},
	}
}

func labeledPod(name string, app string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: sourceTestNs,
			Labels:    map[string]string{"app": app},
		},
	}
}

func sourceWithWorkloadLabels(name string, workloadName string, workloadKind k8sconsts.WorkloadKind, disable bool) *odigosv1.Source {
	return &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: sourceTestNs,
			Labels: map[string]string{
				k8sconsts.WorkloadNameLabel:      workloadName,
				k8sconsts.WorkloadNamespaceLabel: sourceTestNs,
				k8sconsts.WorkloadKindLabel:      string(workloadKind),
			},
		},
		Spec: odigosv1.SourceSpec{DisableInstrumentation: disable},
	}
}

func podRevisionAnnotations(pods *corev1.PodList) map[string]string {
	annotations := make(map[string]string, len(pods.Items))
	for i := range pods.Items {
		annotations[pods.Items[i].GetName()] = pods.Items[i].Annotations[OdigosRunningLatestWorkloadRevisionAnnotation]
	}
	return annotations
}

func TestGetSourcePodsDeployment(t *testing.T) {
	deploymentOwner := metav1.OwnerReference{Kind: "Deployment", Name: sourceTestWorkload, UID: sourceTestDeployUID}

	t.Run("pods of every owned replicaset are reported with their revision", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(
			deploymentReplicaSet("checkout-old", "1", "oldhash", deploymentOwner),
			deploymentReplicaSet("checkout-new", "2", "newhash", deploymentOwner),
			replicaSetPod("checkout-old-1", "oldhash"),
			replicaSetPod("checkout-new-1", "newhash"),
		)
		workloadObj := workloadObject(k8sconsts.WorkloadKindDeployment, appLabelSelector(sourceTestPodLabel))
		workloadObj.Annotations = map[string]string{"deployment.kubernetes.io/revision": "2"}

		pods, err := getSourcePods(context.Background(), kubeClient, workloadObj)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"checkout-old-1": "false",
			"checkout-new-1": "true",
		}, podRevisionAnnotations(pods))
	})

	t.Run("replicasets owned by another deployment are ignored", func(t *testing.T) {
		otherDeploymentOwner := metav1.OwnerReference{Kind: "Deployment", Name: "checkout-v2", UID: types.UID("other-uid")}
		kubeClient := fake.NewSimpleClientset(
			deploymentReplicaSet("checkout-other", "1", "otherhash", otherDeploymentOwner),
			replicaSetPod("checkout-other-1", "otherhash"),
		)
		workloadObj := workloadObject(k8sconsts.WorkloadKindDeployment, appLabelSelector(sourceTestPodLabel))
		workloadObj.Annotations = map[string]string{"deployment.kubernetes.io/revision": "1"}

		pods, err := getSourcePods(context.Background(), kubeClient, workloadObj)

		require.NoError(t, err)
		assert.Empty(t, pods.Items)
	})

	t.Run("replicasets owned by another kind are ignored", func(t *testing.T) {
		rolloutOwner := metav1.OwnerReference{Kind: "Rollout", Name: sourceTestWorkload, UID: sourceTestDeployUID}
		kubeClient := fake.NewSimpleClientset(
			deploymentReplicaSet("checkout-rollout", "1", "rollouthash", rolloutOwner),
			replicaSetPod("checkout-rollout-1", "rollouthash"),
		)
		workloadObj := workloadObject(k8sconsts.WorkloadKindDeployment, appLabelSelector(sourceTestPodLabel))
		workloadObj.Annotations = map[string]string{"deployment.kubernetes.io/revision": "1"}

		pods, err := getSourcePods(context.Background(), kubeClient, workloadObj)

		require.NoError(t, err)
		assert.Empty(t, pods.Items)
	})

	// without a revision annotation on the deployment no replicaset can be the active one
	t.Run("deployment without a revision annotation marks every pod as an older revision", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(
			deploymentReplicaSet("checkout-new", "1", "newhash", deploymentOwner),
			replicaSetPod("checkout-new-1", "newhash"),
		)
		workloadObj := workloadObject(k8sconsts.WorkloadKindDeployment, appLabelSelector(sourceTestPodLabel))

		pods, err := getSourcePods(context.Background(), kubeClient, workloadObj)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{"checkout-new-1": "false"}, podRevisionAnnotations(pods))
	})

	// a replicaset with no revision annotation must not be treated as the active one just
	// because the deployment has no revision annotation either
	t.Run("an empty revision on both the deployment and the replicaset is not a match", func(t *testing.T) {
		replicaSet := deploymentReplicaSet("checkout-new", "", "newhash", deploymentOwner)
		delete(replicaSet.Annotations, "deployment.kubernetes.io/revision")
		kubeClient := fake.NewSimpleClientset(replicaSet, replicaSetPod("checkout-new-1", "newhash"))
		workloadObj := workloadObject(k8sconsts.WorkloadKindDeployment, appLabelSelector(sourceTestPodLabel))

		pods, err := getSourcePods(context.Background(), kubeClient, workloadObj)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{"checkout-new-1": "false"}, podRevisionAnnotations(pods))
	})

	t.Run("existing pod annotations are preserved", func(t *testing.T) {
		pod := replicaSetPod("checkout-new-1", "newhash")
		pod.Annotations = map[string]string{"kubectl.kubernetes.io/restartedAt": "2026-08-28T00:00:00Z"}
		kubeClient := fake.NewSimpleClientset(
			deploymentReplicaSet("checkout-new", "2", "newhash", deploymentOwner),
			pod,
		)
		workloadObj := workloadObject(k8sconsts.WorkloadKindDeployment, appLabelSelector(sourceTestPodLabel))
		workloadObj.Annotations = map[string]string{"deployment.kubernetes.io/revision": "2"}

		pods, err := getSourcePods(context.Background(), kubeClient, workloadObj)

		require.NoError(t, err)
		require.Len(t, pods.Items, 1)
		assert.Equal(t, "2026-08-28T00:00:00Z", pods.Items[0].Annotations["kubectl.kubernetes.io/restartedAt"])
		assert.Equal(t, "true", pods.Items[0].Annotations[OdigosRunningLatestWorkloadRevisionAnnotation])
	})

	t.Run("replicaset list failure is propagated", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset()
		kubeClient.PrependReactor("list", "replicasets",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("replicasets is forbidden")
			})

		_, err := getSourcePods(context.Background(), kubeClient,
			workloadObject(k8sconsts.WorkloadKindDeployment, appLabelSelector(sourceTestPodLabel)))

		require.ErrorContains(t, err, "error listing replicasets")
		assert.ErrorContains(t, err, "replicasets is forbidden")
	})

	t.Run("pod list failure is propagated", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(deploymentReplicaSet("checkout-new", "2", "newhash", deploymentOwner))
		kubeClient.PrependReactor("list", "pods",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("pods is forbidden")
			})
		workloadObj := workloadObject(k8sconsts.WorkloadKindDeployment, appLabelSelector(sourceTestPodLabel))
		workloadObj.Annotations = map[string]string{"deployment.kubernetes.io/revision": "2"}

		_, err := getSourcePods(context.Background(), kubeClient, workloadObj)

		require.ErrorContains(t, err, "error listing pods for replicaset")
		assert.ErrorContains(t, err, "pods is forbidden")
	})
}

func TestGetSourcePodsOtherWorkloadKinds(t *testing.T) {
	t.Run("pods are selected by the workload label selector", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(
			labeledPod("checkout-1", sourceTestPodLabel),
			labeledPod("checkout-2", sourceTestPodLabel),
			labeledPod("unrelated-1", sourceTestOtherLabel),
		)

		pods, err := getSourcePods(context.Background(), kubeClient,
			workloadObject(k8sconsts.WorkloadKindDaemonSet, appLabelSelector(sourceTestPodLabel)))

		require.NoError(t, err)
		require.Len(t, pods.Items, 2)
		assert.ElementsMatch(t, []string{"checkout-1", "checkout-2"},
			[]string{pods.Items[0].GetName(), pods.Items[1].GetName()})
	})

	// a static pod has no selector, the workload name is the pod name
	t.Run("a workload without a label selector resolves the pod by name", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(labeledPod(sourceTestWorkload, sourceTestPodLabel))

		pods, err := getSourcePods(context.Background(), kubeClient,
			workloadObject(k8sconsts.WorkloadKindStaticPod, nil))

		require.NoError(t, err)
		require.Len(t, pods.Items, 1)
		assert.Equal(t, sourceTestWorkload, pods.Items[0].GetName())
	})

	t.Run("a missing pod for a workload without a label selector is an error", func(t *testing.T) {
		_, err := getSourcePods(context.Background(), fake.NewSimpleClientset(),
			workloadObject(k8sconsts.WorkloadKindStaticPod, nil))

		require.Error(t, err)
		assert.ErrorContains(t, err, sourceTestWorkload)
	})

	t.Run("pod list failure is propagated", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset()
		kubeClient.PrependReactor("list", "pods",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("pods is forbidden")
			})

		_, err := getSourcePods(context.Background(), kubeClient,
			workloadObject(k8sconsts.WorkloadKindStatefulSet, appLabelSelector(sourceTestPodLabel)))

		require.ErrorContains(t, err, "pods is forbidden")
	})
}

func TestGetSources(t *testing.T) {
	t.Run("nil workload object", func(t *testing.T) {
		_, err := getSources(context.Background(), odigosfake.NewSimpleClientset().OdigosV1alpha1(), nil)

		require.ErrorContains(t, err, "workload object is nil")
	})

	t.Run("no sources", func(t *testing.T) {
		sources, err := getSources(context.Background(), odigosfake.NewSimpleClientset().OdigosV1alpha1(),
			workloadObject(k8sconsts.WorkloadKindDeployment, nil))

		require.NoError(t, err)
		assert.Nil(t, sources.Workload)
		assert.Nil(t, sources.Namespace)
	})

	t.Run("the workload source and the namespace source are told apart by their labels", func(t *testing.T) {
		workloadSource := sourceWithWorkloadLabels("checkout-source", sourceTestWorkload, k8sconsts.WorkloadKindDeployment, false)
		namespaceSource := sourceWithWorkloadLabels("shop-source", sourceTestNs, k8sconsts.WorkloadKindNamespace, true)
		// a source for a different workload in the same namespace must not be picked up
		otherWorkloadSource := sourceWithWorkloadLabels("frontend-source", "frontend", k8sconsts.WorkloadKindDeployment, false)
		// a source for a workload of the same name but another kind must not be picked up either
		otherKindSource := sourceWithWorkloadLabels("checkout-ss-source", sourceTestWorkload, k8sconsts.WorkloadKindStatefulSet, false)

		odigosClient := odigosfake.NewSimpleClientset(workloadSource, namespaceSource, otherWorkloadSource, otherKindSource)

		sources, err := getSources(context.Background(), odigosClient.OdigosV1alpha1(),
			workloadObject(k8sconsts.WorkloadKindDeployment, nil))

		require.NoError(t, err)
		require.NotNil(t, sources.Workload)
		assert.Equal(t, "checkout-source", sources.Workload.GetName())
		require.NotNil(t, sources.Namespace)
		assert.Equal(t, "shop-source", sources.Namespace.GetName())
		assert.True(t, sources.Namespace.Spec.DisableInstrumentation)
	})

	t.Run("a namespace workload only looks up the namespace source", func(t *testing.T) {
		namespaceSource := sourceWithWorkloadLabels("shop-source", sourceTestNs, k8sconsts.WorkloadKindNamespace, false)
		// a namespace object carries its name in the name field and no namespace
		workloadObj := &K8sSourceObject{
			ObjectMeta: metav1.ObjectMeta{Name: sourceTestNs},
			Kind:       k8sconsts.WorkloadKindNamespace,
		}

		sources, err := getSources(context.Background(),
			odigosfake.NewSimpleClientset(namespaceSource).OdigosV1alpha1(), workloadObj)

		require.NoError(t, err)
		assert.Nil(t, sources.Workload)
		require.NotNil(t, sources.Namespace)
		assert.Equal(t, "shop-source", sources.Namespace.GetName())
	})

	t.Run("more than one workload source is an error", func(t *testing.T) {
		first := sourceWithWorkloadLabels("checkout-source", sourceTestWorkload, k8sconsts.WorkloadKindDeployment, false)
		second := sourceWithWorkloadLabels("checkout-source-duplicate", sourceTestWorkload, k8sconsts.WorkloadKindDeployment, true)

		_, err := getSources(context.Background(), odigosfake.NewSimpleClientset(first, second).OdigosV1alpha1(),
			workloadObject(k8sconsts.WorkloadKindDeployment, nil))

		require.ErrorIs(t, err, odigosv1.ErrorTooManySources)
	})

	t.Run("more than one namespace source is an error", func(t *testing.T) {
		first := sourceWithWorkloadLabels("shop-source", sourceTestNs, k8sconsts.WorkloadKindNamespace, false)
		second := sourceWithWorkloadLabels("shop-source-duplicate", sourceTestNs, k8sconsts.WorkloadKindNamespace, true)

		_, err := getSources(context.Background(), odigosfake.NewSimpleClientset(first, second).OdigosV1alpha1(),
			workloadObject(k8sconsts.WorkloadKindDeployment, nil))

		require.ErrorIs(t, err, odigosv1.ErrorTooManySources)
	})

	t.Run("workload source list failure is propagated", func(t *testing.T) {
		odigosClient := odigosfake.NewSimpleClientset()
		odigosClient.PrependReactor("list", "sources",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("sources is forbidden")
			})

		_, err := getSources(context.Background(), odigosClient.OdigosV1alpha1(),
			workloadObject(k8sconsts.WorkloadKindDeployment, nil))

		require.ErrorContains(t, err, "sources is forbidden")
	})

	// a namespace workload skips the workload source lookup, so this exercises the namespace list
	t.Run("namespace source list failure is propagated", func(t *testing.T) {
		odigosClient := odigosfake.NewSimpleClientset()
		odigosClient.PrependReactor("list", "sources",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("sources is forbidden")
			})

		_, err := getSources(context.Background(), odigosClient.OdigosV1alpha1(), &K8sSourceObject{
			ObjectMeta: metav1.ObjectMeta{Name: sourceTestNs},
			Kind:       k8sconsts.WorkloadKindNamespace,
		})

		require.ErrorContains(t, err, "sources is forbidden")
	})
}

func TestGetRelevantSourceResources(t *testing.T) {
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: sourceTestNs}}

	t.Run("the workload namespace must exist", func(t *testing.T) {
		_, err := GetRelevantSourceResources(context.Background(), fake.NewSimpleClientset(),
			odigosfake.NewSimpleClientset().OdigosV1alpha1(),
			workloadObject(k8sconsts.WorkloadKindStaticPod, nil))

		require.Error(t, err)
		assert.ErrorContains(t, err, sourceTestNs)
	})

	t.Run("a workload without an instrumentation config is not an error", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(namespace, labeledPod(sourceTestWorkload, sourceTestPodLabel))

		resources, err := GetRelevantSourceResources(context.Background(), kubeClient,
			odigosfake.NewSimpleClientset().OdigosV1alpha1(),
			workloadObject(k8sconsts.WorkloadKindStaticPod, nil))

		require.NoError(t, err)
		assert.Nil(t, resources.InstrumentationConfig)
		require.NotNil(t, resources.Sources)
		assert.Nil(t, resources.Sources.Workload)
		require.NotNil(t, resources.Namespace)
		assert.Equal(t, sourceTestNs, resources.Namespace.GetName())
		require.NotNil(t, resources.Pods)
		assert.Len(t, resources.Pods.Items, 1)
	})

	t.Run("the instrumentation config and instances of the workload are collected", func(t *testing.T) {
		runtimeObjectName := "statefulset-" + sourceTestWorkload
		instrumentationConfig := &odigosv1.InstrumentationConfig{
			ObjectMeta: metav1.ObjectMeta{Name: runtimeObjectName, Namespace: sourceTestNs},
		}
		// instrumentation instances are selected by the instrumented-app label, not by name
		instance := &odigosv1.InstrumentationInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "checkout-1-checkout",
				Namespace: sourceTestNs,
				Labels:    map[string]string{"instrumented-app": runtimeObjectName},
			},
		}
		otherWorkloadInstance := &odigosv1.InstrumentationInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "frontend-1-frontend",
				Namespace: sourceTestNs,
				Labels:    map[string]string{"instrumented-app": "statefulset-frontend"},
			},
		}
		kubeClient := fake.NewSimpleClientset(namespace, labeledPod("checkout-1", sourceTestPodLabel))

		resources, err := GetRelevantSourceResources(context.Background(), kubeClient,
			odigosfake.NewSimpleClientset(instrumentationConfig, instance, otherWorkloadInstance).OdigosV1alpha1(),
			workloadObject(k8sconsts.WorkloadKindStatefulSet, appLabelSelector(sourceTestPodLabel)))

		require.NoError(t, err)
		require.NotNil(t, resources.InstrumentationConfig)
		assert.Equal(t, runtimeObjectName, resources.InstrumentationConfig.GetName())
		require.NotNil(t, resources.InstrumentationInstances)
		require.Len(t, resources.InstrumentationInstances.Items, 1)
		assert.Equal(t, "checkout-1-checkout", resources.InstrumentationInstances.Items[0].GetName())
	})

	t.Run("instrumentation config get failure is propagated", func(t *testing.T) {
		odigosClient := odigosfake.NewSimpleClientset()
		odigosClient.PrependReactor("get", "instrumentationconfigs",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("instrumentationconfigs is forbidden")
			})

		_, err := GetRelevantSourceResources(context.Background(), fake.NewSimpleClientset(namespace),
			odigosClient.OdigosV1alpha1(), workloadObject(k8sconsts.WorkloadKindStaticPod, nil))

		require.ErrorContains(t, err, "instrumentationconfigs is forbidden")
	})

	t.Run("instrumentation instance list failure is propagated", func(t *testing.T) {
		odigosClient := odigosfake.NewSimpleClientset()
		odigosClient.PrependReactor("list", "instrumentationinstances",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("instrumentationinstances is forbidden")
			})

		_, err := GetRelevantSourceResources(context.Background(), fake.NewSimpleClientset(namespace),
			odigosClient.OdigosV1alpha1(), workloadObject(k8sconsts.WorkloadKindStaticPod, nil))

		require.ErrorContains(t, err, "instrumentationinstances is forbidden")
	})

	t.Run("source list failure is propagated", func(t *testing.T) {
		odigosClient := odigosfake.NewSimpleClientset()
		odigosClient.PrependReactor("list", "sources",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("sources is forbidden")
			})

		_, err := GetRelevantSourceResources(context.Background(), fake.NewSimpleClientset(namespace),
			odigosClient.OdigosV1alpha1(), workloadObject(k8sconsts.WorkloadKindStaticPod, nil))

		require.ErrorContains(t, err, "sources is forbidden")
	})

	t.Run("pod resolution failure is propagated", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(namespace)
		kubeClient.PrependReactor("list", "pods",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("pods is forbidden")
			})

		_, err := GetRelevantSourceResources(context.Background(), kubeClient,
			odigosfake.NewSimpleClientset().OdigosV1alpha1(),
			workloadObject(k8sconsts.WorkloadKindStatefulSet, appLabelSelector(sourceTestPodLabel)))

		require.ErrorContains(t, err, "pods is forbidden")
	})
}
