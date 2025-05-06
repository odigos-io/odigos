package sourceinstrumentation_test

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Source controller", func() {
	ctx := context.Background()
	var namespace *corev1.Namespace
	var deployment *appsv1.Deployment
	var source *odigosv1.Source
	var instrumentationConfig *odigosv1.InstrumentationConfig

	Describe("Workload Instrumentation", func() {
		BeforeEach(func() {
			namespace = testutil.NewMockNamespace()
			Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

			deployment = testutil.NewMockTestDeployment(namespace, "test-deployment")
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			instrumentationConfig = testutil.NewMockInstrumentationConfig(deployment)

			source = testutil.NewMockSource(deployment, false)
			Expect(k8sClient.Create(ctx, source)).Should(Succeed())
			testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig)
		})

		When("Sources are instrumented", func() {
			It("Creates an InstrumentationConfig for the instrumented workload", func() {
				testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig)
			})

		})

		When("Sources are uninstrumented", func() {
			It("Deletes the InstrumentationConfig for the uninstrumented workload", func() {
				Expect(k8sClient.Delete(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfig)
			})

			It("Deletes the InstrumentationConfig when a Workload Source is updated to disableInstrumentation=true", func() {
				source.Spec.DisableInstrumentation = true
				Expect(k8sClient.Update(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfig)
			})

		})
	})

	Describe("Namespace instrumentation", func() {
		BeforeEach(func() {
			namespace = testutil.NewMockNamespace()
			Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

			deployment = testutil.NewMockTestDeployment(namespace, "test-deployment")
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			instrumentationConfig = testutil.NewMockInstrumentationConfig(deployment)
		})

		When("Namespaces are instrumented", func() {
			It("Creates an InstrumentationConfig for each workload in the namespace", func() {
				source = testutil.NewMockSource(namespace, false)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig)
			})

			It("Creates an InstrumentationConfig for workloads in the namespace which are not already instrumented", func() {
				deployment2 := testutil.NewMockTestDeployment(namespace, "test-deployment-2")
				Expect(k8sClient.Create(ctx, deployment2)).Should(Succeed())
				source2 := testutil.NewMockSource(deployment2, false)
				Expect(k8sClient.Create(ctx, source2)).Should(Succeed())
				instrumentationConfig2 := testutil.NewMockInstrumentationConfig(deployment2)
				testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig2)

				source = testutil.NewMockSource(namespace, false)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig)
			})

			It("Does not create an InstrumentationConfig for Excluded Workloads", func() {
				deployment2 := testutil.NewMockTestDeployment(namespace, "test-deployment-2")
				Expect(k8sClient.Create(ctx, deployment2)).Should(Succeed())
				source2 := testutil.NewMockSource(deployment2, true)
				Expect(k8sClient.Create(ctx, source2)).Should(Succeed())
				instrumentationConfig2 := testutil.NewMockInstrumentationConfig(deployment2)

				source = testutil.NewMockSource(namespace, false)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig)
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfig2)
			})

			It("Creates an InstrumentationConfig when an Excluded Workload Source is deleted", func() {
				deployment2 := testutil.NewMockTestDeployment(namespace, "test-deployment-2")
				Expect(k8sClient.Create(ctx, deployment2)).Should(Succeed())
				source2 := testutil.NewMockSource(deployment2, true)
				Expect(k8sClient.Create(ctx, source2)).Should(Succeed())
				instrumentationConfig2 := testutil.NewMockInstrumentationConfig(deployment2)

				source = testutil.NewMockSource(namespace, false)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig)
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfig2)

				Expect(k8sClient.Delete(ctx, source2)).Should(Succeed())
				testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig2)
			})

			It("Creates an InstrumentationConfig when an Excluded Workload Source is updated to disableInstrumentation=false", func() {
				deployment2 := testutil.NewMockTestDeployment(namespace, "test-deployment-2")
				Expect(k8sClient.Create(ctx, deployment2)).Should(Succeed())
				source2 := testutil.NewMockSource(deployment2, true)
				Expect(k8sClient.Create(ctx, source2)).Should(Succeed())
				instrumentationConfig2 := testutil.NewMockInstrumentationConfig(deployment2)

				source = testutil.NewMockSource(namespace, false)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig)
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfig2)

				source2.Spec.DisableInstrumentation = false
				Expect(k8sClient.Update(ctx, source2)).Should(Succeed())
				testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig2)
			})

		})

		When("Namespaces are uninstrumented", func() {
			It("Deletes the InstrumentationConfig for each workload in the namespace", func() {
				source = testutil.NewMockSource(namespace, false)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig)

				Expect(k8sClient.Delete(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfig)
			})

			It("Does not delete the InstrumentationConfig for individual Workloads", func() {
				source = testutil.NewMockSource(deployment, false)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigCreated(ctx, k8sClient, instrumentationConfig)

				source2 := testutil.NewMockSource(namespace, false)
				Expect(k8sClient.Create(ctx, source2)).Should(Succeed())
				Expect(k8sClient.Delete(ctx, source2)).Should(Succeed())
				testutil.AssertInstrumentationConfigRetained(ctx, k8sClient, instrumentationConfig)
			})

		})
	})

	Describe("Retain annotations", func() {

		BeforeEach(func() {
			namespace = testutil.NewMockNamespace()
			Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

			deployment = testutil.NewMockTestDeployment(namespace, "test-deployment")
			deployment = testutil.SetReportedNameAnnotation(deployment, "test")
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			source = testutil.NewMockSource(deployment, false)
			Expect(k8sClient.Create(ctx, source)).Should(Succeed())

			instrumentationConfig = testutil.NewMockInstrumentationConfig(deployment)
		})

		It("should retain other annotations on instrumentation source deleted", func() {

			annotationKey := "test"
			annotationValue := "test"

			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: deployment.GetNamespace(), Name: deployment.GetName()}, deployment)).Should(Succeed())
			if len(deployment.Annotations) == 0 {
				deployment.Annotations = make(map[string]string)
			}
			deployment.Annotations[annotationKey] = annotationValue
			Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())

			Expect(k8sClient.Delete(ctx, source)).Should(Succeed())

			testutil.AssertDeploymentAnnotationRetained(ctx, k8sClient, deployment, annotationKey, annotationValue)
		})
	})
})
