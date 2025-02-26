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

var _ = Describe("deleteInstrumentationConfig Deployment controller", func() {
	ctx := context.Background()
	var namespace *corev1.Namespace
	var deployment *appsv1.Deployment
	var source, nsSource, depSource *odigosv1.Source
	var instrumentationConfig *odigosv1.InstrumentationConfig

	Describe("Delete InstrumentationConfig", func() {

		When("Namespace is not instrumented", func() {

			BeforeEach(func() {
				namespace = testutil.NewMockNamespace()
				Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

				deployment = testutil.NewMockTestDeployment(namespace)
				Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

				source = testutil.NewMockSource(deployment)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())

				instrumentationConfig = testutil.NewMockInstrumentationConfig(deployment)
				//Expect(k8sClient.Create(ctx, instrumentationConfig)).Should(Succeed())
			})

			It("InstrumentationConfig deleted after removing instrumentation source from deployment", func() {
				Expect(k8sClient.Delete(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfig)
			})

			It("InstrumentationConfig deleted after setting instrumentation source to disabled", func() {
				source.Spec.DisableInstrumentation = true
				Expect(k8sClient.Update(ctx, source)).Should(Succeed())
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfig)
			})

		})

		When("Namespace is instrumented", func() {

			BeforeEach(func() {
				namespace = testutil.NewMockNamespace()
				Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

				nsSource = testutil.NewMockSource(namespace)
				Expect(k8sClient.Create(ctx, nsSource)).Should(Succeed())

				deployment = testutil.NewMockTestDeployment(namespace)
				Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

				depSource = testutil.NewMockSource(deployment)
				Expect(k8sClient.Create(ctx, depSource)).Should(Succeed())

				instrumentationConfig = testutil.NewMockInstrumentationConfig(deployment)
			})

			It("InstrumentationConfig retain when removing instrumentation source from deployment", func() {
				Expect(k8sClient.Delete(ctx, depSource)).Should(Succeed())
				testutil.AssertInstrumentationConfigRetained(ctx, k8sClient, instrumentationConfig)
			})

			It("InstrumentationConfig deleted after setting instrumentation source to disabled", func() {
				depSource.Spec.DisableInstrumentation = true
				Expect(k8sClient.Update(ctx, depSource)).Should(Succeed())
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfig)
			})
		})
	})

	Describe("Delete reported name annotation", func() {

		BeforeEach(func() {
			namespace = testutil.NewMockNamespace()
			Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

			deployment = testutil.NewMockTestDeployment(namespace)
			deployment = testutil.SetReportedNameAnnotation(deployment, "test")
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			source = testutil.NewMockSource(deployment)
			Expect(k8sClient.Create(ctx, source)).Should(Succeed())

			instrumentationConfig = testutil.NewMockInstrumentationConfig(deployment)
		})

		It("should delete the reported name annotation on instrumentation source disabled", func() {

			source.Spec.DisableInstrumentation = true
			Expect(k8sClient.Update(ctx, source)).Should(Succeed())
			testutil.AssertReportedNameAnnotationDeletedDeployment(ctx, k8sClient, deployment)
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
