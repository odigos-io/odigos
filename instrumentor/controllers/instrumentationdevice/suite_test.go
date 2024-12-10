/*
Copyright 2022.

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

package instrumentationdevice_test

import (
	"context"
	"path/filepath"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/controllers/instrumentationdevice"
	//+kubebuilder:scaffold:imports
)

var (
	cfg                *rest.Config
	k8sClient          client.Client
	testEnv            *envtest.Environment
	testCtx            context.Context
	cancel             context.CancelFunc
	origGetDefaultSDKs func() map[common.ProgrammingLanguage]common.OtelSdk
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "InstrumentationDevice Controllers Suite")
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

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// create the odigos system namespace
	odigosSystemNamespace := testutil.NewOdigosSystemNamespace()
	Expect(k8sClient.Create(testCtx, odigosSystemNamespace)).Should(Succeed())

	configmap := testutil.NewMockOdigosConfig()
	Expect(k8sClient.Create(testCtx, configmap)).Should(Succeed())

	// report the node collector is ready
	datacollection := testutil.NewMockDataCollection()
	Expect(k8sClient.Create(testCtx, datacollection)).Should(Succeed())
	k8sClient.Get(testCtx, types.NamespacedName{Name: datacollection.GetName(), Namespace: datacollection.GetNamespace()}, datacollection)
	datacollection.Status.Ready = true
	Expect(k8sClient.Status().Update(testCtx, datacollection)).Should(Succeed())

	// create odigos configuration with default sdks
	origGetDefaultSDKs = instrumentationdevice.GetDefaultSDKs
	instrumentationdevice.GetDefaultSDKs = testutil.MockGetDefaultSDKs

	webhookInstallOptions := &testEnv.WebhookInstallOptions
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
	})
	Expect(err).ToNot(HaveOccurred())

	err = instrumentationdevice.SetupWithManager(k8sManager)
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
	instrumentationdevice.GetDefaultSDKs = origGetDefaultSDKs
})
