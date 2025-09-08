/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clustercollector_test

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/clustercollector"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var testCtx context.Context
var cancel context.CancelFunc

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "ClusterCollector Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	testCtx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "api", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = odigosv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = actionv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	Expect(err).ToNot(HaveOccurred())

	err = clustercollector.SetupWithManager(k8sManager, nil, "")
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(testCtx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func cleanupResources() {
	// Clean up all Destinations
	destinationList := &odigosv1.DestinationList{}
	k8sClient.List(testCtx, destinationList)
	for _, destination := range destinationList.Items {
		Eventually(func() bool {
			err := k8sClient.Delete(testCtx, &destination)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}

	// Clean up all CollectorsGroups
	collectorsGroupList := &odigosv1.CollectorsGroupList{}
	k8sClient.List(testCtx, collectorsGroupList)
	for _, collectorsGroup := range collectorsGroupList.Items {
		Eventually(func() bool {
			err := k8sClient.Delete(testCtx, &collectorsGroup)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}

	secretList := &corev1.SecretList{}
	k8sClient.List(testCtx, secretList)
	for _, secret := range secretList.Items {
		Eventually(func() bool {
			err := k8sClient.Delete(testCtx, &secret)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}

	deploymentList := &appsv1.DeploymentList{}
	k8sClient.List(testCtx, deploymentList)
	for _, deployment := range deploymentList.Items {
		Eventually(func() bool {
			err := k8sClient.Delete(testCtx, &deployment)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}

	namespaceList := &corev1.NamespaceList{}
	k8sClient.List(testCtx, namespaceList)
	for _, namespace := range namespaceList.Items {
		Eventually(func() bool {
			err := k8sClient.Delete(testCtx, &namespace)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}
}
