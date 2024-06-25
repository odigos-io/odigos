package instrumentationdevice_test

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("envoverwrite", func() {
	ctx := context.Background()
	var namespace *corev1.Namespace
	var deployment *appsv1.Deployment
	var instrumentedApplication *odigosv1.InstrumentedApplication

	var deploymentSdk common.OtelSdk = common.OtelSdkNativeCommunity
	var testEnvVar = "PYTHONPATH"
	// the following is the value that odigos will append to the user's env
	testEnvOdigosValue, found := envOverwrite.ValToAppend(testEnvVar, deploymentSdk)
	Expect(found).Should(BeTrue())

	BeforeEach(func() {
		namespace = testutil.NewMockNamespace()
		Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())
	})

	Describe("User did not set any env", func() {

		BeforeEach(func() {
			deployment = testutil.SetOdigosInstrumentationEnabled(testutil.NewMockTestDeployment(namespace))
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
		})

		When("When odigos first detects empty env variables", func() {

			BeforeEach(func() {
				instrumentedApplication = testutil.NewMockInstrumentedApplication(deployment)
				Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())
			})

			It("should not inject odigos env", func() {
				testutil.AssertDepContainerEnvRemainEmpty(ctx, k8sClient, deployment)
			})
		})

		When("Odigos detects the odigos env variable inject by k8s via device", func() {

			BeforeEach(func() {
				instrumentedApplication = testutil.SetInstrumentedApplicationContainer(testutil.NewMockInstrumentedApplication(deployment), &testEnvVar, &testEnvOdigosValue, common.PythonProgrammingLanguage)
				Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())
			})

			It("should not inject odigos env", func() {
				testutil.AssertDepContainerEnvRemainEmpty(ctx, k8sClient, deployment)
			})
		})
	})

	Describe("User set env var via dockerfile and not in manifest", func() {
		userEnvValue := "/foo"
		mergedEnvValue := userEnvValue + ":" + testEnvOdigosValue

		BeforeEach(func() {
			deployment = testutil.SetOdigosInstrumentationEnabled(testutil.NewMockTestDeployment(namespace))
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
		})

		When("The observed value is just the env", func() {

			It("Should add the merged environment to manifest", func() {
				instrumentedApplication = testutil.SetInstrumentedApplicationContainer(testutil.NewMockInstrumentedApplication(deployment), &testEnvVar, &userEnvValue, common.PythonProgrammingLanguage)
				Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())
				testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVar, mergedEnvValue)

				// simulate the observed value in the pod is updated with the merged value in the manifest
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: instrumentedApplication.Namespace, Name: instrumentedApplication.Name}, instrumentedApplication)).Should(Succeed())
				instrumentedApplication.Spec.RuntimeDetails[0].EnvVars[0].Value = mergedEnvValue
				Expect(k8sClient.Update(ctx, instrumentedApplication)).Should(Succeed())

				// manifest value should still contain the merged value
				testutil.AssertDepContainerSingleEnvRemainsSame(ctx, k8sClient, deployment, testEnvVar, mergedEnvValue)

				// when uninstrumented, the value should be reverted to the original value which was empty in manifest
				Expect(k8sClient.Delete(ctx, instrumentedApplication)).Should(Succeed())
				testutil.AssertDepContainerSingleEnvBecomesEmpty(ctx, k8sClient, deployment)
			})
		})

		When("The observed value is both user value and odigos value", func() {

			It("Should add the merged environment to manifest", func() {
				instrumentedApplication = testutil.SetInstrumentedApplicationContainer(testutil.NewMockInstrumentedApplication(deployment), &testEnvVar, &mergedEnvValue, common.PythonProgrammingLanguage)
				Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())
				testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVar, mergedEnvValue)
			})
		})
	})

	Describe("User set env var via manifest and not in dockerfile", func() {
		userEnvValue := "/bar"
		mergedEnvValue := userEnvValue + ":" + testEnvOdigosValue

		BeforeEach(func() {
			deployment = testutil.SetDeploymentContainerEnv(
				testutil.SetOdigosInstrumentationEnabled(
					testutil.NewMockTestDeployment(namespace),
				),
				testEnvVar, userEnvValue)
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
		})

		It("Should add the merged environment to manifest", func() {
			instrumentedApplication = testutil.SetInstrumentedApplicationContainer(testutil.NewMockInstrumentedApplication(deployment), &testEnvVar, &userEnvValue, common.PythonProgrammingLanguage)
			Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVar, mergedEnvValue)

			// simulate the observed value in the pod is updated with the merged value in the manifest
			Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: instrumentedApplication.Namespace, Name: instrumentedApplication.Name}, instrumentedApplication)).Should(Succeed())
			instrumentedApplication.Spec.RuntimeDetails[0].EnvVars[0].Value = mergedEnvValue
			Expect(k8sClient.Update(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVar, mergedEnvValue)

			// uninstrument the deployment, value should be reverted to the user value
			Expect(k8sClient.Delete(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVar, userEnvValue)
		})
	})

	Describe("Env is set in both dockerfile and manifest", func() {
		dockerEnvValue := "/bar"
		manifestEnvValue := "/foo"
		mergedEnvValue := dockerEnvValue + ":" + testEnvOdigosValue

		BeforeEach(func() {
			deployment = testutil.SetDeploymentContainerEnv(
				testutil.SetOdigosInstrumentationEnabled(
					testutil.NewMockTestDeployment(namespace),
				),
				testEnvVar, manifestEnvValue)
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
		})

		It("Should add the merged environment to manifest", func() {
			instrumentedApplication = testutil.SetInstrumentedApplicationContainer(testutil.NewMockInstrumentedApplication(deployment), &testEnvVar, &dockerEnvValue, common.PythonProgrammingLanguage)
			Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVar, mergedEnvValue)

			// simulate the observed value in the pod is updated with the merged value in the manifest
			Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: instrumentedApplication.Namespace, Name: instrumentedApplication.Name}, instrumentedApplication)).Should(Succeed())
			instrumentedApplication.Spec.RuntimeDetails[0].EnvVars[0].Value = mergedEnvValue
			Expect(k8sClient.Update(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVar, mergedEnvValue)

			// uninstrument the deployment, value should be reverted to the user value
			Expect(k8sClient.Delete(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVar, manifestEnvValue)
		})
	})

})
