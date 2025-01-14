package deleteinstrumentationconfig_test

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

var _ = Describe("deleteInstrumentationConfig Deployment controller", func() {
	ctx := context.Background()
	var namespace *corev1.Namespace
	var deployment *appsv1.Deployment
	var source *odigosv1.Source
	var instrumentationConfig *odigosv1.InstrumentationConfig

	Describe("Delete InstrumentationConfig", func() {

		When("Namespace is not instrumented", func() {

			BeforeEach(func() {
				namespace = testutil.NewMockNamespace()
				Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

				deployment = testutil.SetOdigosInstrumentationEnabled(testutil.NewMockTestDeployment(namespace))
				Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

				source = testutil.NewMockSource(deployment)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())

				instrumentationConfig = testutil.NewMockInstrumentationConfig(deployment)
				Expect(k8sClient.Create(ctx, instrumentationConfig)).Should(Succeed())
			})

			It("InstrumentationConfig retained after removing instrumentation label from deployment", func() {
				deployment = testutil.DeleteOdigosInstrumentationLabel(deployment)
				Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())
				testutil.AssertInstrumentationConfigRetained(ctx, k8sClient, instrumentationConfig)
			})

			It("InstrumentationConfig deleted after setting instrumentation label to disabled", func() {
				deployment = testutil.SetOdigosInstrumentationDisabled(deployment)
				Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfig)
			})

		})

		When("Namespace is instrumented", func() {

			BeforeEach(func() {
				namespace = testutil.SetOdigosInstrumentationEnabled(testutil.NewMockNamespace())
				Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

				source = testutil.NewMockSource(namespace)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())

				deployment = testutil.SetOdigosInstrumentationEnabled(testutil.NewMockTestDeployment(namespace))
				Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

				source = testutil.NewMockSource(deployment)
				Expect(k8sClient.Create(ctx, source)).Should(Succeed())

				instrumentationConfig = testutil.NewMockInstrumentationConfig(deployment)
				Expect(k8sClient.Create(ctx, instrumentationConfig)).Should(Succeed())
			})

			It("InstrumentationConfig retain when removing instrumentation label from deployment", func() {
				deployment = testutil.DeleteOdigosInstrumentationLabel(deployment)
				Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())
				testutil.AssertInstrumentationConfigRetained(ctx, k8sClient, instrumentationConfig)
			})

			It("InstrumentationConfig deleted after setting instrumentation label to disabled", func() {
				deployment = testutil.SetOdigosInstrumentationDisabled(deployment)
				Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfig)
			})
		})
	})

	Describe("Delete reported name annotation", func() {

		BeforeEach(func() {
			namespace = testutil.NewMockNamespace()
			Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

			deployment = testutil.SetOdigosInstrumentationEnabled(testutil.NewMockTestDeployment(namespace))
			deployment = testutil.SetReportedNameAnnotation(deployment, "test")
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			source = testutil.NewMockSource(deployment)
			Expect(k8sClient.Create(ctx, source)).Should(Succeed())

			instrumentationConfig = testutil.NewMockInstrumentationConfig(deployment)
			Expect(k8sClient.Create(ctx, instrumentationConfig)).Should(Succeed())
		})

		It("should delete the reported name annotation on instrumentation label disabled", func() {

			Eventually(func() error {
				k8sClient.Get(ctx, client.ObjectKey{Namespace: deployment.GetNamespace(), Name: deployment.GetName()}, deployment)
				deployment = testutil.SetOdigosInstrumentationDisabled(deployment)
				return k8sClient.Update(ctx, deployment)
			}, time.Second*2, time.Millisecond*250).Should(Succeed())
			testutil.AssertReportedNameAnnotationDeletedDeployment(ctx, k8sClient, deployment)
		})

		It("should retain other annotations on instrumentation label deleted", func() {

			annotationKey := "test"
			annotationValue := "test"

			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: deployment.GetNamespace(), Name: deployment.GetName()}, deployment)).Should(Succeed())
			if len(deployment.Annotations) == 0 {
				deployment.Annotations = make(map[string]string)
			}
			deployment.Annotations[annotationKey] = annotationValue
			Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())

			deployment = testutil.SetOdigosInstrumentationDisabled(deployment)
			Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())

			testutil.AssertDeploymentAnnotationRetained(ctx, k8sClient, deployment, annotationKey, annotationValue)
		})
	})

})
