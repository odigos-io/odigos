/*
<<<<<<< HEAD
Copyright 2022.
=======
Copyright 2025.
>>>>>>> 9c5f87341 (Add actions controller suite tests)

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

package actions_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/actions"
	//+kubebuilder:scaffold:imports
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	testCtx   context.Context
	cancel    context.CancelFunc
	timeout   = time.Second * 10
	interval  = time.Millisecond * 250
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Actions Controllers Suite")
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
	})
	Expect(err).ToNot(HaveOccurred())

	err = actions.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(testCtx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

}, 60)

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func cleanupResources() {
	// Clean up all Actions
	actionList := &odigosv1.ActionList{}
	k8sClient.List(testCtx, actionList)
	for _, action := range actionList.Items {
		Eventually(func() bool {
			err := k8sClient.Delete(testCtx, &action)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}

	// Clean up all legacy ErrorSamplers
	errorSamplerList := &actionv1.ErrorSamplerList{}
	k8sClient.List(testCtx, errorSamplerList)
	for _, sampler := range errorSamplerList.Items {
		Eventually(func() bool {
			err := k8sClient.Delete(testCtx, &sampler)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}

	// Clean up all legacy LatencySamplers
	latencySamplerList := &actionv1.LatencySamplerList{}
	k8sClient.List(testCtx, latencySamplerList)
	for _, sampler := range latencySamplerList.Items {
		Eventually(func() bool {
			err := k8sClient.Delete(testCtx, &sampler)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}

	// Clean up all legacy ServiceNameSamplers
	serviceNameSamplerList := &actionv1.ServiceNameSamplerList{}
	k8sClient.List(testCtx, serviceNameSamplerList)
	for _, sampler := range serviceNameSamplerList.Items {
		Eventually(func() bool {
			err := k8sClient.Delete(testCtx, &sampler)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}

	// Clean up all legacy SpanAttributeSamplers
	spanAttributeSamplerList := &actionv1.SpanAttributeSamplerList{}
	k8sClient.List(testCtx, spanAttributeSamplerList)
	for _, sampler := range spanAttributeSamplerList.Items {
		Eventually(func() bool {
			err := k8sClient.Delete(testCtx, &sampler)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}

	// Clean up all legacy ProbabilisticSamplers
	probabilisticSamplerList := &actionv1.ProbabilisticSamplerList{}
	k8sClient.List(testCtx, probabilisticSamplerList)
	for _, sampler := range probabilisticSamplerList.Items {
		Eventually(func() bool {
			err := k8sClient.Delete(testCtx, &sampler)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}

<<<<<<< HEAD
	// Clean up all legacy K8sAttributesResolvers
	k8sAttributesResolverList := &actionv1.K8sAttributesResolverList{}
	k8sClient.List(testCtx, k8sAttributesResolverList)
	for _, resolver := range k8sAttributesResolverList.Items {
		Eventually(func() bool {
			err := k8sClient.Delete(testCtx, &resolver)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}

=======
>>>>>>> 9c5f87341 (Add actions controller suite tests)
	// Clean up all Processors
	processorList := &odigosv1.ProcessorList{}
	k8sClient.List(testCtx, processorList)
	for _, processor := range processorList.Items {
		Eventually(func() bool {
			err := k8sClient.Delete(testCtx, &processor)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}
}
