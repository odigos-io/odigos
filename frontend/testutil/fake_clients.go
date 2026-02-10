package testutil

import (
	"time"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/kube"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	versionutil "k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8stesting "k8s.io/client-go/testing"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	crfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func SlowReactor(latency time.Duration) k8stesting.ReactionFunc {
	return func(action k8stesting.Action) (bool, runtime.Object, error) {
		time.Sleep(latency)
		return false, nil, nil
	}
}

func SlowFakeClient(latency time.Duration, k8sObjects []runtime.Object, odigosObjects []runtime.Object) *kube.Client {
	k8sFake := kubefake.NewSimpleClientset(k8sObjects...)
	k8sFake.PrependReactor("*", "*", SlowReactor(latency))
	k8sFake.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &versionutil.Info{
		GitVersion: "v1.28.0",
	}

	odigosFake := odigosfake.NewSimpleClientset(odigosObjects...)
	odigosFake.PrependReactor("*", "*", SlowReactor(latency))

	return &kube.Client{
		Interface:     k8sFake,
		OdigosClient:  odigosFake.OdigosV1alpha1(),
		DynamicClient: FakeDynamicClient(),
	}
}

func FakeCacheClient(objects ...ctrlclient.Object) ctrlclient.Client {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = odigosv1alpha1.AddToScheme(scheme)
	return crfake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func FakeDynamicClient() *fakedynamic.FakeDynamicClient {
	dynScheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(dynScheme)
	return fakedynamic.NewSimpleDynamicClientWithCustomListKinds(dynScheme,
		map[schema.GroupVersionResource]string{
			{Group: "apps.openshift.io", Version: "v1", Resource: "deploymentconfigs"}: "DeploymentConfigList",
			{Group: "argoproj.io", Version: "v1alpha1", Resource: "rollouts"}:           "RolloutList",
		},
	)
}
