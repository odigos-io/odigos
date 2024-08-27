package instrumentationdevice_test

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func GetOdigosConfig(configMap *corev1.ConfigMap, odigosConfig *common.OdigosConfiguration) error {
	if err := yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), &odigosConfig); err != nil {
		return err
	}
	return nil
}

func SetOdigosConfig(configMap *corev1.ConfigMap, odigosConfig *common.OdigosConfiguration) error {
	data, err := yaml.Marshal(odigosConfig)
	if err != nil {
		return err
	}
	configMap.Data[consts.OdigosConfigurationFileName] = string(data)
	return nil
}

var _ = Describe("envoverwrite", func() {
	ctx := context.Background()
	var namespace *corev1.Namespace
	var deployment *appsv1.Deployment
	var instrumentedApplication *odigosv1.InstrumentedApplication

	testProgrammingLanguagePython := common.PythonProgrammingLanguage
	deploymentSdk := common.OtelSdkNativeCommunity
	testEnvVarPythonPath := "PYTHONPATH"
	// the following is the value that odigos will append to the user's env
	var testEnvOdigosValue string

	BeforeEach(func() {
		// create a new namespace for each test to prevent conflict
		namespace = testutil.NewMockNamespace()
		Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

		sdkEnvVal, found := envOverwrite.ValToAppend(testEnvVarPythonPath, deploymentSdk)
		Expect(found).Should(BeTrue())
		testEnvOdigosValue = sdkEnvVal
	})

	AfterEach(func() {
		// restore odigos config to it's original state
		var cm corev1.ConfigMap
		var odigosConfig common.OdigosConfiguration
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: consts.DefaultOdigosNamespace, Name: consts.OdigosConfigurationName}, &cm)).Should(Succeed())
		GetOdigosConfig(&cm, &odigosConfig)
		odigosConfig.DefaultSDKs[testProgrammingLanguagePython] = common.OtelSdkNativeCommunity
		SetOdigosConfig(&cm, &odigosConfig)
		Expect(k8sClient.Update(ctx, &cm)).Should(Succeed())
	})

	Describe("User did not set env in manifest or docker image", func() {

		BeforeEach(func() {
			// create a deployment with no env
			deployment = testutil.SetOdigosInstrumentationEnabled(testutil.NewMockTestDeployment(namespace))
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
		})

		It("should not add env vars to deployment", func() {
			// initial state - no env varas in manifest or dockerfile
			// and odigos haven't yet injected it's env, so the deployment should have no env vars
			instrumentedApplication = testutil.NewMockInstrumentedApplication(deployment)
			Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())

			// odigos env is the only one, so no need to inject anything to the manifest
			testutil.AssertDepContainerEnvRemainEmpty(ctx, k8sClient, deployment)

			// now the pods restarts, and odigos detects the env var it injected
			// via the instrumentation device.
			// instrumented application should be updated with the odigos env
			k8sClient.Get(ctx, client.ObjectKeyFromObject(instrumentedApplication), instrumentedApplication)
			instrumentedApplication = testutil.SetInstrumentedApplicationContainer(instrumentedApplication, &testEnvVarPythonPath, &testEnvOdigosValue, testProgrammingLanguagePython)
			Expect(k8sClient.Update(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerEnvRemainEmpty(ctx, k8sClient, deployment)

			// uninstrument the deployment, env var in deployment should remain empty
			Expect(k8sClient.Delete(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerEnvRemainEmpty(ctx, k8sClient, deployment)
		})
	})

	Describe("User set env var via dockerfile and not in manifest", func() {
		userEnvValue := "/from_dockerfile"
		var mergedEnvValue string

		BeforeEach(func() {
			mergedEnvValue = userEnvValue + ":" + testEnvOdigosValue
			// the env var is in dockerfile, thus the manifest should start empty of env vars
			deployment = testutil.SetOdigosInstrumentationEnabled(testutil.NewMockTestDeployment(namespace))
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
		})

		It("Should add the dockerfile env and odigos env to manifest and successfully revert", func() {
			// initial state - should capture the env var from dockerfile only
			instrumentedApplication = testutil.SetInstrumentedApplicationContainer(testutil.NewMockInstrumentedApplication(deployment), &testEnvVarPythonPath, &userEnvValue, testProgrammingLanguagePython)
			Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())

			// odigos should merge the value from dockerfile and odigos env
			testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVarPythonPath, mergedEnvValue)

			// after instrumentation is applied, now the value in the pod should be the merged value
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(instrumentedApplication), instrumentedApplication)).Should(Succeed())
			instrumentedApplication.Spec.RuntimeDetails[0].EnvVars[0].Value = mergedEnvValue
			Expect(k8sClient.Update(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnvRemainsSame(ctx, k8sClient, deployment, testEnvVarPythonPath, mergedEnvValue)

			// when uninstrumented, the value should be reverted to the original value which was empty in manifest
			Expect(k8sClient.Delete(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnvBecomesEmpty(ctx, k8sClient, deployment)
		})

	})

	Describe("Container is run with env variables of other language than what it runs", func() {

		userEnvValue := "/from_dockerfile"

		BeforeEach(func() {
			// the env var is in dockerfile, thus the manifest should start empty of env vars
			deployment = testutil.SetOdigosInstrumentationEnabled(testutil.NewMockTestDeployment(namespace))
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
		})

		It("Should not add the unrelated env vars to the manifest", func() {
			instrumentedApplication = testutil.SetInstrumentedApplicationContainer(testutil.NewMockInstrumentedApplication(deployment), &testEnvVarPythonPath, &userEnvValue, common.JavaProgrammingLanguage)
			Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())

			// odigos found a relevant env var for python, but it should not be injected to the manifest
			// as the language is not Python
			testutil.AssertDepContainerEnvRemainEmpty(ctx, k8sClient, deployment)
		})

		It("Should not add the unrelated env vars with different otel SDKs", func() {
			// make the SDK for python and java different
			var cm corev1.ConfigMap
			var odigosConfig common.OdigosConfiguration
			Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: consts.DefaultOdigosNamespace, Name: consts.OdigosConfigurationName}, &cm)).Should(Succeed())
			GetOdigosConfig(&cm, &odigosConfig)
			odigosConfig.DefaultSDKs[common.JavaProgrammingLanguage] = common.OtelSdkEbpfEnterprise
			SetOdigosConfig(&cm, &odigosConfig)
			Expect(k8sClient.Update(ctx, &cm)).Should(Succeed())

			instrumentedApplication = testutil.SetInstrumentedApplicationContainer(testutil.NewMockInstrumentedApplication(deployment), &testEnvVarPythonPath, &userEnvValue, common.JavaProgrammingLanguage)
			Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())

			// odigos found a relevant env var for python, but it should not be injected to the manifest
			// as the language is not Python
			testutil.AssertDepContainerEnvRemainEmpty(ctx, k8sClient, deployment)
		})
	})

	Describe("User set env var via manifest and not in dockerfile", func() {
		userEnvValue := "/from_manifest"
		var mergedEnvValue string

		BeforeEach(func() {
			mergedEnvValue = userEnvValue + ":" + testEnvOdigosValue
			// the env var is in manifest, thus the deployment should contain it at the start
			deployment = testutil.SetDeploymentContainerEnv(
				testutil.SetOdigosInstrumentationEnabled(
					testutil.NewMockTestDeployment(namespace),
				),
				testEnvVarPythonPath, userEnvValue)
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
		})

		It("Should update the manifest with merged value, and revet when uninstrumenting", func() {
			// initial state - should capture the env var from manifest only
			instrumentedApplication = testutil.SetInstrumentedApplicationContainer(testutil.NewMockInstrumentedApplication(deployment), &testEnvVarPythonPath, &userEnvValue, testProgrammingLanguagePython)
			Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())

			// odigos should merge the value from manifest and odigos env
			testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVarPythonPath, mergedEnvValue)

			// after instrumentation is applied, now the value in the pod should be the merged value
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(instrumentedApplication), instrumentedApplication)).Should(Succeed())
			instrumentedApplication.Spec.RuntimeDetails[0].EnvVars[0].Value = mergedEnvValue
			Expect(k8sClient.Update(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnvRemainsSame(ctx, k8sClient, deployment, testEnvVarPythonPath, mergedEnvValue)

			// when uninstrumented, the value should be reverted to the original value which was in the manifest
			Expect(k8sClient.Delete(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVarPythonPath, userEnvValue)
		})
	})

	When("Default SDK changes after env var is injected", func() {

		userEnvValue := "/from_manifest"

		BeforeEach(func() {
			// the env var is in manifest, thus the deployment should contain it at the start
			deployment = testutil.SetDeploymentContainerEnv(
				testutil.SetOdigosInstrumentationEnabled(
					testutil.NewMockTestDeployment(namespace),
				),
				testEnvVarPythonPath, userEnvValue)
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			// initial state - should capture the env var from manifest only
			mergedEnvValue := userEnvValue + ":" + testEnvOdigosValue
			instrumentedApplication = testutil.SetInstrumentedApplicationContainer(testutil.NewMockInstrumentedApplication(deployment), &testEnvVarPythonPath, &userEnvValue, testProgrammingLanguagePython)
			Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVarPythonPath, mergedEnvValue)

			// after instrumentation is applied, now the value in the pod should be the merged value
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(instrumentedApplication), instrumentedApplication)).Should(Succeed())
			instrumentedApplication.Spec.RuntimeDetails[0].EnvVars[0].Value = mergedEnvValue
			Expect(k8sClient.Update(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnvRemainsSame(ctx, k8sClient, deployment, testEnvVarPythonPath, mergedEnvValue)
		})

		When("Default SDK changes to another SDK", func() {
			newSdk := common.OtelSdkEbpfEnterprise

			BeforeEach(func() {
				var cm corev1.ConfigMap
				var odigosConfig common.OdigosConfiguration
				Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: consts.DefaultOdigosNamespace, Name: consts.OdigosConfigurationName}, &cm)).Should(Succeed())
				GetOdigosConfig(&cm, &odigosConfig)
				odigosConfig.DefaultSDKs[testProgrammingLanguagePython] = newSdk
				SetOdigosConfig(&cm, &odigosConfig)
				Expect(k8sClient.Update(ctx, &cm)).Should(Succeed())
			})

			It("Should update the manifest with new odigos env value", func() {
				newOdigosValue, found := envOverwrite.ValToAppend(testEnvVarPythonPath, newSdk)
				Expect(found).Should(BeTrue())
				newMergedEnvValue := userEnvValue + ":" + newOdigosValue

				// after the odigos config is updated, the deployment should be updated with the new odigos value
				testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVarPythonPath, newMergedEnvValue)

				// when uninstrumented, the value should be reverted to the original value which was in the manifest
				Expect(k8sClient.Delete(ctx, instrumentedApplication)).Should(Succeed())
				testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVarPythonPath, userEnvValue)
			})
		})
	})

	When("Apply to workload restores the value to it's original state", func() {

		userEnvValue := "/orig_in_manifest"
		var mergedEnvValue string

		BeforeEach(func() {
			// the env var is in manifest, thus the deployment should contain it at the start
			deployment = testutil.SetDeploymentContainerEnv(
				testutil.SetOdigosInstrumentationEnabled(
					testutil.NewMockTestDeployment(namespace),
				),
				testEnvVarPythonPath, userEnvValue)
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			// initial state - should capture the env var from manifest only
			mergedEnvValue = userEnvValue + ":" + testEnvOdigosValue
			instrumentedApplication = testutil.SetInstrumentedApplicationContainer(testutil.NewMockInstrumentedApplication(deployment), &testEnvVarPythonPath, &userEnvValue, testProgrammingLanguagePython)
			Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVarPythonPath, mergedEnvValue)

			// after instrumentation is applied, now the value in the pod should be the merged value
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(instrumentedApplication), instrumentedApplication)).Should(Succeed())
			instrumentedApplication.Spec.RuntimeDetails[0].EnvVars[0].Value = mergedEnvValue
			Expect(k8sClient.Update(ctx, instrumentedApplication)).Should(Succeed())
			testutil.AssertDepContainerSingleEnvRemainsSame(ctx, k8sClient, deployment, testEnvVarPythonPath, mergedEnvValue)
		})

		It("Should reapply odigos value to the manifest", func() {
			// when the deployment is updated, the odigos value should be reapplied
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)).Should(Succeed())
			// restore the value to the original state
			deployment = testutil.SetDeploymentContainerEnv(deployment, testEnvVarPythonPath, userEnvValue)
			Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())

			// the odigos value should be reapplied
			testutil.AssertDepContainerSingleEnv(ctx, k8sClient, deployment, testEnvVarPythonPath, mergedEnvValue)
		})
	})

})
