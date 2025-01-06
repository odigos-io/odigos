package deleteinstrumentationconfig_test

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("DeleteInstrumentationConfig Namespace controller", func() {

	ctx := context.Background()
	var namespace *corev1.Namespace

	var deployment *appsv1.Deployment
	var daemonSet *appsv1.DaemonSet
	var statefulSet *appsv1.StatefulSet

	var instrumentationConfigDeployment *odigosv1.InstrumentationConfig
	var instrumentationConfigDaemonSet *odigosv1.InstrumentationConfig
	var instrumentationConfigStatefulSet *odigosv1.InstrumentationConfig

	When("namespace instrumentation is disabled", func() {

		BeforeEach(func() {
			namespace = testutil.SetOdigosInstrumentationEnabled(testutil.NewMockNamespace())
			Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())
		})

		Context("workloads instrumentation label is not set (inherit from namespace)", func() {

			BeforeEach(func() {
				deployment = testutil.SetReportedNameAnnotation(testutil.NewMockTestDeployment(namespace), "foo")
				Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
				daemonSet = testutil.SetReportedNameAnnotation(testutil.NewMockTestDaemonSet(namespace), "foo")
				Expect(k8sClient.Create(ctx, daemonSet)).Should(Succeed())
				statefulSet = testutil.SetReportedNameAnnotation(testutil.NewMockTestStatefulSet(namespace), "foo")
				Expect(k8sClient.Create(ctx, statefulSet)).Should(Succeed())

				// these workloads has instrumentation application because the namespace has instrumentation enabled
				instrumentationConfigDeployment = testutil.NewMockInstrumentationConfig(deployment)
				Expect(k8sClient.Create(ctx, instrumentationConfigDeployment)).Should(Succeed())
				instrumentationConfigDaemonSet = testutil.NewMockInstrumentationConfig(daemonSet)
				Expect(k8sClient.Create(ctx, instrumentationConfigDaemonSet)).Should(Succeed())
				instrumentationConfigStatefulSet = testutil.NewMockInstrumentationConfig(statefulSet)
				Expect(k8sClient.Create(ctx, instrumentationConfigStatefulSet)).Should(Succeed())
			})

			It("should delete instrumented application", func() {

				namespace = testutil.SetOdigosInstrumentationDisabled(namespace)
				Expect(k8sClient.Update(ctx, namespace)).Should(Succeed())

				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfigDeployment)
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfigDaemonSet)
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfigStatefulSet)
			})

			It("should delete reported name annotation", func() {

				namespace = testutil.SetOdigosInstrumentationDisabled(namespace)
				Expect(k8sClient.Update(ctx, namespace)).Should(Succeed())

				testutil.AssertReportedNameAnnotationDeletedDeployment(ctx, k8sClient, deployment)
				testutil.AssertReportedNameAnnotationDeletedDaemonSet(ctx, k8sClient, daemonSet)
				testutil.AssertReportedNameAnnotationDeletedStatefulSet(ctx, k8sClient, statefulSet)
			})
		})

		Context("workloads instrumentation label enabled (override namespace)", func() {

			BeforeEach(func() {
				deployment = testutil.SetOdigosInstrumentationEnabled(testutil.NewMockTestDeployment(namespace))
				Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
				daemonSet = testutil.SetOdigosInstrumentationEnabled(testutil.NewMockTestDaemonSet(namespace))
				Expect(k8sClient.Create(ctx, daemonSet)).Should(Succeed())
				statefulSet = testutil.SetOdigosInstrumentationEnabled(testutil.NewMockTestStatefulSet(namespace))
				Expect(k8sClient.Create(ctx, statefulSet)).Should(Succeed())

				// these workloads has instrumentation application because the namespace has instrumentation enabled
				instrumentationConfigDeployment = testutil.NewMockInstrumentationConfig(deployment)
				Expect(k8sClient.Create(ctx, instrumentationConfigDeployment)).Should(Succeed())
				instrumentationConfigDaemonSet = testutil.NewMockInstrumentationConfig(daemonSet)
				Expect(k8sClient.Create(ctx, instrumentationConfigDaemonSet)).Should(Succeed())
				instrumentationConfigStatefulSet = testutil.NewMockInstrumentationConfig(statefulSet)
				Expect(k8sClient.Create(ctx, instrumentationConfigStatefulSet)).Should(Succeed())
			})

			It("should retain instrumented application", func() {
				namespace = testutil.SetOdigosInstrumentationDisabled(namespace)
				Expect(k8sClient.Update(ctx, namespace)).Should(Succeed())

				testutil.AssertInstrumentationConfigRetained(ctx, k8sClient, instrumentationConfigDeployment)
				testutil.AssertInstrumentationConfigRetained(ctx, k8sClient, instrumentationConfigDaemonSet)
				testutil.AssertInstrumentationConfigRetained(ctx, k8sClient, instrumentationConfigStatefulSet)
			})
		})
	})
})
