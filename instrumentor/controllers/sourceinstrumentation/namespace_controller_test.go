package sourceinstrumentation_test

import (
	"context"

	k8sconsts "github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Namespace controller", func() {

	ctx := context.Background()
	var namespace *corev1.Namespace

	var deployment *appsv1.Deployment
	var daemonSet *appsv1.DaemonSet
	var statefulSet *appsv1.StatefulSet

	var sourceNamespace, sourceDeployment, sourceDaemonSet, sourceStatefulSet *odigosv1.Source

	var instrumentationConfigDeployment *odigosv1.InstrumentationConfig
	var instrumentationConfigDaemonSet *odigosv1.InstrumentationConfig
	var instrumentationConfigStatefulSet *odigosv1.InstrumentationConfig

	When("namespace instrumentation is disabled", func() {

		BeforeEach(func() {
			namespace = testutil.NewMockNamespace()
			Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())
			sourceNamespace = testutil.NewMockSource(namespace, false)
			sourceNamespace.Spec.DisableInstrumentation = false
			sourceNamespace.Finalizers = []string{k8sconsts.SourceInstrumentationFinalizer}
			Expect(k8sClient.Create(ctx, sourceNamespace)).Should(Succeed())
		})

		Context("workloads instrumentation source is not set (inherit from namespace)", func() {

			BeforeEach(func() {
				deployment = testutil.SetReportedNameAnnotation(testutil.NewMockTestDeployment(namespace, "test-deployment"), "foo")
				Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
				daemonSet = testutil.SetReportedNameAnnotation(testutil.NewMockTestDaemonSet(namespace), "foo")
				Expect(k8sClient.Create(ctx, daemonSet)).Should(Succeed())
				statefulSet = testutil.SetReportedNameAnnotation(testutil.NewMockTestStatefulSet(namespace), "foo")
				Expect(k8sClient.Create(ctx, statefulSet)).Should(Succeed())

				// these workloads has instrumentation application because the namespace has instrumentation enabled
				instrumentationConfigDeployment = testutil.NewMockInstrumentationConfig(deployment)
				instrumentationConfigDaemonSet = testutil.NewMockInstrumentationConfig(daemonSet)
				instrumentationConfigStatefulSet = testutil.NewMockInstrumentationConfig(statefulSet)
			})

			It("should delete InstrumentationConfig", func() {
				sourceNamespace.Spec.DisableInstrumentation = true
				sourceNamespace.Finalizers = []string{k8sconsts.SourceInstrumentationFinalizer}
				Expect(k8sClient.Update(ctx, sourceNamespace)).Should(Succeed())

				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfigDeployment)
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfigDaemonSet)
				testutil.AssertInstrumentationConfigDeleted(ctx, k8sClient, instrumentationConfigStatefulSet)
			})

		})

		Context("workloads instrumentation source enabled (override namespace)", func() {

			BeforeEach(func() {
				deployment = testutil.NewMockTestDeployment(namespace, "test-deployment")
				Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
				daemonSet = testutil.NewMockTestDaemonSet(namespace)
				Expect(k8sClient.Create(ctx, daemonSet)).Should(Succeed())
				statefulSet = testutil.NewMockTestStatefulSet(namespace)
				Expect(k8sClient.Create(ctx, statefulSet)).Should(Succeed())

				sourceDeployment = testutil.NewMockSource(deployment, false)
				Expect(k8sClient.Create(ctx, sourceDeployment)).Should(Succeed())
				sourceDaemonSet = testutil.NewMockSource(daemonSet, false)
				Expect(k8sClient.Create(ctx, sourceDaemonSet)).Should(Succeed())
				sourceStatefulSet = testutil.NewMockSource(statefulSet, false)
				Expect(k8sClient.Create(ctx, sourceStatefulSet)).Should(Succeed())

				instrumentationConfigDeployment = testutil.NewMockInstrumentationConfig(deployment)
				instrumentationConfigDaemonSet = testutil.NewMockInstrumentationConfig(daemonSet)
				instrumentationConfigStatefulSet = testutil.NewMockInstrumentationConfig(statefulSet)
			})

			It("should retain InstrumentationConfig", func() {
				sourceNamespace.Spec.DisableInstrumentation = true
				sourceNamespace.Finalizers = []string{k8sconsts.SourceInstrumentationFinalizer}
				Expect(k8sClient.Update(ctx, sourceNamespace)).Should(Succeed())

				testutil.AssertInstrumentationConfigRetained(ctx, k8sClient, instrumentationConfigDeployment)
				testutil.AssertInstrumentationConfigRetained(ctx, k8sClient, instrumentationConfigDaemonSet)
				testutil.AssertInstrumentationConfigRetained(ctx, k8sClient, instrumentationConfigStatefulSet)
			})
		})
	})
})
