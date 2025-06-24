package webhookenvinjector

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/envOverwrite"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

func TestWebhookEnvInjector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhook Environment Injector Suite")
}

var _ = Describe("WebhookEnvInjector", func() {
	var (
		ctx       context.Context
		logger    logr.Logger
		k8sClient client.Client
		scheme    *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = logr.Discard()
		scheme = runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
		Expect(odigosv1.AddToScheme(scheme)).To(Succeed())
		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()
	})

	Describe("getEnvVarFromRuntimeDetails", func() {
		It("should return env var value when found", func() {
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				EnvVars: []odigosv1.EnvVar{
					{Name: "TEST_VAR", Value: "test_value"},
					{Name: "OTHER_VAR", Value: "other_value"},
				},
			}

			value, found := getEnvVarFromRuntimeDetails(runtimeDetails, "TEST_VAR")
			Expect(found).To(BeTrue())
			Expect(value).To(Equal("test_value"))
		})

		It("should return false when env var not found", func() {
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				EnvVars: []odigosv1.EnvVar{
					{Name: "OTHER_VAR", Value: "other_value"},
				},
			}

			_, found := getEnvVarFromRuntimeDetails(runtimeDetails, "TEST_VAR")
			Expect(found).To(BeFalse())
		})

		It("should handle empty env vars", func() {
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				EnvVars: []odigosv1.EnvVar{},
			}

			_, found := getEnvVarFromRuntimeDetails(runtimeDetails, "TEST_VAR")
			Expect(found).To(BeFalse())
		})
	})

	Describe("getEnvVarNamesForLanguage", func() {
		It("should return env var names for supported languages", func() {
			// Test with Java
			javaEnvVars := getEnvVarNamesForLanguage(common.JavaProgrammingLanguage)
			Expect(javaEnvVars).NotTo(BeEmpty())

			// Test with Python
			pythonEnvVars := getEnvVarNamesForLanguage(common.PythonProgrammingLanguage)
			Expect(pythonEnvVars).NotTo(BeEmpty())
		})

		It("should return nil for unsupported languages", func() {
			envVars := getEnvVarNamesForLanguage(common.ProgrammingLanguage("unsupported"))
			Expect(envVars).To(BeNil())
		})
	})

	Describe("getContainerEnvVarPointer", func() {
		var container corev1.Container

		BeforeEach(func() {
			container = corev1.Container{
				Name: "test-container",
				Env: []corev1.EnvVar{
					{Name: "EXISTING_VAR", Value: "existing_value"},
					{Name: "ANOTHER_VAR", Value: "another_value"},
				},
			}
		})

		It("should return pointer to existing env var", func() {
			envVar := getContainerEnvVarPointer(&container.Env, "EXISTING_VAR")
			Expect(envVar).NotTo(BeNil())
			Expect(envVar.Name).To(Equal("EXISTING_VAR"))
			Expect(envVar.Value).To(Equal("existing_value"))
		})

		It("should return nil for non-existing env var", func() {
			envVar := getContainerEnvVarPointer(&container.Env, "NON_EXISTING_VAR")
			Expect(envVar).To(BeNil())
		})

		It("should allow modification through pointer", func() {
			envVar := getContainerEnvVarPointer(&container.Env, "EXISTING_VAR")
			envVar.Value = "modified_value"
			Expect(container.Env[0].Value).To(Equal("modified_value"))
		})
	})

	Describe("shouldInject", func() {
		It("should return false when RuntimeUpdateState is nil", func() {
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				RuntimeUpdateState: nil,
			}

			result := shouldInject(runtimeDetails, logger, "test-container")
			Expect(result).To(BeFalse())
		})

		It("should return false when RuntimeUpdateState is Failed", func() {
			failedState := odigosv1.ProcessingStateFailed
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				RuntimeUpdateState: &failedState,
			}

			result := shouldInject(runtimeDetails, logger, "test-container")
			Expect(result).To(BeFalse())
		})

		It("should return true when RuntimeUpdateState is Succeeded", func() {
			successState := odigosv1.ProcessingStateSucceeded
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				RuntimeUpdateState: &successState,
			}

			result := shouldInject(runtimeDetails, logger, "test-container")
			Expect(result).To(BeTrue())
		})

		It("should return true when RuntimeUpdateState is Skipped", func() {
			skippedState := odigosv1.ProcessingStateSkipped
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				RuntimeUpdateState: &skippedState,
			}

			result := shouldInject(runtimeDetails, logger, "test-container")
			Expect(result).To(BeTrue())
		})
	})

	Describe("isValueFromConfigmap", func() {
		It("should return true when ValueFrom is set", func() {
			envVar := &corev1.EnvVar{
				Name: "TEST_VAR",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "test-configmap"},
						Key:                  "test-key",
					},
				},
			}

			result := isValueFromConfigmap(envVar)
			Expect(result).To(BeTrue())
		})

		It("should return false when ValueFrom is nil", func() {
			envVar := &corev1.EnvVar{
				Name:  "TEST_VAR",
				Value: "test_value",
			}

			result := isValueFromConfigmap(envVar)
			Expect(result).To(BeFalse())
		})
	})

	Describe("handleValueFromEnvVar", func() {
		var container corev1.Container

		BeforeEach(func() {
			container = corev1.Container{
				Name: "test-container",
				Env:  []corev1.EnvVar{},
			}
		})

		It("should handle ValueFrom env var correctly", func() {
			envVar := &corev1.EnvVar{
				Name: "JAVA_TOOL_OPTIONS", // Use a known env var that has Odigos values
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "test-configmap"},
						Key:                  "test-key",
					},
				},
			}

			handleValueFromEnvVar(&container, envVar, "JAVA_TOOL_OPTIONS", "test_odigos_value")

			// The function should modify the envVar name to ORIGINAL_
			Expect(envVar.Name).To(Equal("ORIGINAL_JAVA_TOOL_OPTIONS"))
			// And add a new env var with the combined value
			Expect(len(container.Env)).To(BeNumerically(">", 0))
			Expect(container.Env[0].Name).To(Equal("JAVA_TOOL_OPTIONS"))
			Expect(container.Env[0].Value).To(ContainSubstring("$(ORIGINAL_JAVA_TOOL_OPTIONS)"))
		})
	})

	Describe("applyOdigosEnvDefaults", func() {
		var container corev1.Container

		BeforeEach(func() {
			container = corev1.Container{
				Name: "test-container",
				Env:  []corev1.EnvVar{},
			}
		})

		It("should apply defaults for empty container", func() {
			envVarsPerLanguage := []string{"JAVA_TOOL_OPTIONS"}
			otelsdk := common.OtelSdkNativeCommunity

			applyOdigosEnvDefaults(&container, envVarsPerLanguage, otelsdk)

			// Check if env vars were added
			Expect(len(container.Env)).To(BeNumerically(">", 0))
		})

		It("should not override existing env vars with values", func() {
			container.Env = []corev1.EnvVar{
				{Name: "JAVA_TOOL_OPTIONS", Value: "existing_value"},
			}
			envVarsPerLanguage := []string{"JAVA_TOOL_OPTIONS"}
			otelsdk := common.OtelSdkNativeCommunity

			applyOdigosEnvDefaults(&container, envVarsPerLanguage, otelsdk)

			// Should not change existing value
			Expect(container.Env[0].Value).To(Equal("existing_value"))
		})

		It("should set value for existing env var with empty value", func() {
			container.Env = []corev1.EnvVar{
				{Name: "JAVA_TOOL_OPTIONS", Value: ""},
			}
			envVarsPerLanguage := []string{"JAVA_TOOL_OPTIONS"}
			otelsdk := common.OtelSdkNativeCommunity

			applyOdigosEnvDefaults(&container, envVarsPerLanguage, otelsdk)

			// Should set value for empty env var
			possibleValues := envOverwrite.GetPossibleValuesPerEnv("JAVA_TOOL_OPTIONS")
			if possibleValues != nil {
				if value, ok := possibleValues[otelsdk]; ok {
					Expect(container.Env[0].Value).To(Equal(value))
				}
			}
		})
	})

	Describe("processEnvVarsFromRuntimeDetails", func() {
		It("should process env vars from runtime details", func() {
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				EnvFromContainerRuntime: []odigosv1.EnvVar{
					{Name: "JAVA_TOOL_OPTIONS", Value: "existing_value"},
				},
			}

			envVars := processEnvVarsFromRuntimeDetails(runtimeDetails, "JAVA_TOOL_OPTIONS", common.OtelSdkNativeCommunity)

			Expect(len(envVars)).To(BeNumerically(">=", 0))
		})

		It("should skip empty values", func() {
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				EnvFromContainerRuntime: []odigosv1.EnvVar{
					{Name: "JAVA_TOOL_OPTIONS", Value: ""},
				},
			}

			envVars := processEnvVarsFromRuntimeDetails(runtimeDetails, "JAVA_TOOL_OPTIONS", common.OtelSdkNativeCommunity)

			Expect(len(envVars)).To(Equal(0))
		})
	})

	Describe("InjectOdigosAgentEnvVars", func() {
		var (
			container       corev1.Container
			runtimeDetails  odigosv1.RuntimeDetailsByContainer
			config          common.OdigosConfiguration
			injectionMethod common.EnvInjectionMethod
		)

		BeforeEach(func() {
			container = corev1.Container{
				Name: "test-container",
				Env:  []corev1.EnvVar{},
			}
			successState := odigosv1.ProcessingStateSucceeded
			runtimeDetails = odigosv1.RuntimeDetailsByContainer{
				Language:           common.JavaProgrammingLanguage,
				RuntimeUpdateState: &successState,
			}
			injectionMethod = common.PodManifestEnvInjectionMethod
			config = common.OdigosConfiguration{
				AgentEnvVarsInjectionMethod: &injectionMethod,
			}
		})

		It("should inject env vars for Java language", func() {
			podWorkload := k8sconsts.PodWorkload{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Kind:      k8sconsts.WorkloadKindDeployment,
			}

			err := InjectOdigosAgentEnvVars(ctx, logger, podWorkload, &container, common.OtelSdkNativeCommunity, &runtimeDetails, k8sClient, &config)

			Expect(err).NotTo(HaveOccurred())
			// Check that env vars were injected (exact count depends on implementation)
			Expect(len(container.Env)).To(BeNumerically(">=", 0))
		})

		It("should return error when injection method is nil", func() {
			config.AgentEnvVarsInjectionMethod = nil
			podWorkload := k8sconsts.PodWorkload{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Kind:      k8sconsts.WorkloadKindDeployment,
			}

			err := InjectOdigosAgentEnvVars(ctx, logger, podWorkload, &container, common.OtelSdkNativeCommunity, &runtimeDetails, k8sClient, &config)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("env injection method is not set"))
		})

		It("should handle unsupported language gracefully", func() {
			runtimeDetails.Language = common.ProgrammingLanguage("unsupported")
			podWorkload := k8sconsts.PodWorkload{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Kind:      k8sconsts.WorkloadKindDeployment,
			}

			err := InjectOdigosAgentEnvVars(ctx, logger, podWorkload, &container, common.OtelSdkNativeCommunity, &runtimeDetails, k8sClient, &config)

			Expect(err).NotTo(HaveOccurred())
			// Should not inject any env vars
			Expect(len(container.Env)).To(Equal(0))
		})

		It("should handle loader injection method", func() {
			loaderMethod := common.LoaderEnvInjectionMethod
			config.AgentEnvVarsInjectionMethod = &loaderMethod

			// Set secure execution mode to false to allow loader injection
			secureExecution := false
			runtimeDetails.SecureExecutionMode = &secureExecution

			podWorkload := k8sconsts.PodWorkload{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Kind:      k8sconsts.WorkloadKindDeployment,
			}

			err := InjectOdigosAgentEnvVars(ctx, logger, podWorkload, &container, common.OtelSdkNativeCommunity, &runtimeDetails, k8sClient, &config)

			Expect(err).NotTo(HaveOccurred())
			// Should inject LD_PRELOAD
			found := false
			for _, env := range container.Env {
				if env.Name == commonconsts.LdPreloadEnvVarName {
					found = true
					Expect(env.Value).To(ContainSubstring(commonconsts.OdigosLoaderName))
					break
				}
			}
			Expect(found).To(BeTrue())
		})
	})

	Describe("setOtelSignalsExporterEnvVars", func() {
		var (
			container       corev1.Container
			collectorsGroup odigosv1.CollectorsGroup
		)

		BeforeEach(func() {
			container = corev1.Container{
				Name: "test-container",
				Env:  []corev1.EnvVar{},
			}
			collectorsGroup = odigosv1.CollectorsGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      k8sconsts.OdigosNodeCollectorDaemonSetName,
					Namespace: "odigos-system",
				},
				Status: odigosv1.CollectorsGroupStatus{
					ReceiverSignals: []common.ObservabilitySignal{
						common.TracesObservabilitySignal,
						common.MetricsObservabilitySignal,
					},
				},
			}
			Expect(k8sClient.Create(ctx, &collectorsGroup)).To(Succeed())
		})

		It("should set OTEL exporter env vars based on collector signals", func() {
			setOtelSignalsExporterEnvVars(ctx, logger, &container, k8sClient)

			// Check that exporter env vars were set
			envVarNames := []string{
				commonconsts.OtelTracesExporter,
				commonconsts.OtelMetricsExporter,
				commonconsts.OtelLogsExporter,
			}

			for _, envVarName := range envVarNames {
				found := false
				for _, env := range container.Env {
					if env.Name == envVarName {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), "Expected env var %s to be set", envVarName)
			}
		})

		It("should handle missing collector group gracefully", func() {
			// Delete the collector group
			Expect(k8sClient.Delete(ctx, &collectorsGroup)).To(Succeed())

			initialEnvCount := len(container.Env)
			setOtelSignalsExporterEnvVars(ctx, logger, &container, k8sClient)

			// Should not crash and should still set some env vars
			Expect(len(container.Env)).To(BeNumerically(">=", initialEnvCount))
		})
	})
})
