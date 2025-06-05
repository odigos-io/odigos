package sourceinstrumentation_test

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Workload controllers", func() {
	ctx := context.Background()
	var namespace *corev1.Namespace
	var deployment *appsv1.Deployment
	var source *odigosv1.Source
	var instrumentationConfig *odigosv1.InstrumentationConfig

	Describe("Workload-Source decoupling", func() {
		BeforeEach(func() {
			namespace = testutil.NewMockNamespace()
			Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

			deployment = testutil.NewMockTestDeployment(namespace, "test-deployment")
			instrumentationConfig = testutil.NewMockInstrumentationConfig(deployment)
		})

		When("Workload is created after Source", func() {
			It("Creates an InstrumentationConfig for an instrumented workload", func() {
				source = testutil.NewMockSource(deployment, false)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigNotCreated(ctx, k8sClient, instrumentationConfig)

				Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
				testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig)
			})

			It("Does not create an InstrumentationConfig for a disabled workload", func() {
				source = testutil.NewMockSource(deployment, true)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigNotCreated(ctx, k8sClient, instrumentationConfig)

				Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
				testutil.AssertInstrumentationConfigNotCreated(ctx, k8sClient, instrumentationConfig)
			})
		})
	})

})
