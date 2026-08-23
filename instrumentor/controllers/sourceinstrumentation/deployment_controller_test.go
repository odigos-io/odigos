package sourceinstrumentation_test

import (
	"context"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

		When("The containers of an instrumented workload change", func() {
			BeforeEach(func() {
				source = testutil.NewMockSource(deployment, false)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())
				Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
				testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig)
				assertContainerOverrideNames(ctx, instrumentationConfig, "test")
			})

			It("Records a container that was renamed in the pod template", func() {
				updateDeploymentContainers(ctx, deployment, corev1.Container{Name: "renamed", Image: "test"})

				assertContainerOverrideNames(ctx, instrumentationConfig, "renamed")
			})

			It("Records a container that was added to the pod template", func() {
				updateDeploymentContainers(ctx, deployment,
					corev1.Container{Name: "test", Image: "test"},
					corev1.Container{Name: "added", Image: "test"},
				)

				assertContainerOverrideNames(ctx, instrumentationConfig, "test", "added")
			})
		})
	})

})

func updateDeploymentContainers(ctx context.Context, deployment *appsv1.Deployment, containers ...corev1.Container) {
	Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)).Should(Succeed())
	deployment.Spec.Template.Spec.Containers = containers
	Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())
}

func assertContainerOverrideNames(ctx context.Context, instrumentationConfig *odigosv1.InstrumentationConfig, expected ...string) {
	key := client.ObjectKeyFromObject(instrumentationConfig)
	Eventually(func() []string {
		var ic odigosv1.InstrumentationConfig
		if err := k8sClient.Get(ctx, key, &ic); err != nil {
			return nil
		}
		names := make([]string, 0, len(ic.Spec.ContainersOverrides))
		for _, override := range ic.Spec.ContainersOverrides {
			names = append(names, override.ContainerName)
		}
		return names
	}, 10*time.Second, 250*time.Millisecond).Should(Equal(expected))
}
