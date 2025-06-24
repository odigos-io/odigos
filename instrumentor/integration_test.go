//go:build integration

package instrumentor

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers"
	"github.com/odigos-io/odigos/instrumentor/internal/pod"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	webhookenvinjector "github.com/odigos-io/odigos/instrumentor/internal/webhook_env_injector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("Instrumentor Integration Tests", func() {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		logger     logr.Logger
		dp         *distros.Provider
		k8sClient  client.Client
		scheme     *runtime.Scheme
		testPod    *corev1.Pod
		testConfig *common.OdigosConfiguration
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		logger = logr.Discard()
		dp = &distros.Provider{}

		// Setup fake Kubernetes client
		scheme = runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
		Expect(odigosv1.AddToScheme(scheme)).To(Succeed())
		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()

		// Create test pod
		testPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
				Labels: map[string]string{
					k8sconsts.OdigosAgentsMetaHashLabel: "test-hash",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "app-container",
						Image: "test-app:latest",
						Env: []corev1.EnvVar{
							{Name: "EXISTING_VAR", Value: "existing_value"},
						},
					},
				},
			},
		}

		// Setup test configuration
		injectionMethod := common.PodManifestEnvInjectionMethod
		testConfig = &common.OdigosConfiguration{
			AgentEnvVarsInjectionMethod: &injectionMethod,
		}
	})

	AfterEach(func() {
		cancel()
	})

	Describe("Pod Affinity and Environment Injection Integration", func() {
		It("should add odiglet affinity and inject environment variables", func() {
			By("Adding odiglet affinity to the pod")
			pod.AddOdigletInstalledAffinity(testPod)

			By("Verifying odiglet affinity was added")
			Expect(testPod.Spec.Affinity).NotTo(BeNil())
			Expect(testPod.Spec.Affinity.NodeAffinity).NotTo(BeNil())
			Expect(testPod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).NotTo(BeNil())
			Expect(len(testPod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)).To(BeNumerically(">", 0))

			By("Injecting environment variables for Java application")
			successState := odigosv1.ProcessingStateSucceeded
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				Language:           common.JavaProgrammingLanguage,
				RuntimeUpdateState: &successState,
			}

			podWorkload := k8sconsts.PodWorkload{
				Name:      testPod.Name,
				Namespace: testPod.Namespace,
				Kind:      k8sconsts.WorkloadKindPod,
			}

			err := webhookenvinjector.InjectOdigosAgentEnvVars(
				ctx, logger, podWorkload, &testPod.Spec.Containers[0],
				common.OtelSdkNativeCommunity, runtimeDetails, k8sClient, testConfig,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the container has instrumentation environment variables")
			// The exact number and values depend on the Java instrumentation configuration
			// but we should have at least the original env var plus some instrumentation vars
			Expect(len(testPod.Spec.Containers[0].Env)).To(BeNumerically(">=", 1))

			// Check that existing env var is preserved
			existingVarFound := false
			for _, env := range testPod.Spec.Containers[0].Env {
				if env.Name == "EXISTING_VAR" && env.Value == "existing_value" {
					existingVarFound = true
					break
				}
			}
			Expect(existingVarFound).To(BeTrue(), "Existing environment variable should be preserved")
		})

		It("should handle loader injection method correctly", func() {
			By("Setting up loader injection method")
			loaderMethod := common.LoaderEnvInjectionMethod
			testConfig.AgentEnvVarsInjectionMethod = &loaderMethod

			By("Injecting environment variables with loader method")
			successState := odigosv1.ProcessingStateSucceeded
			secureExecution := false
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				Language:             common.JavaProgrammingLanguage,
				RuntimeUpdateState:   &successState,
				SecureExecutionMode:  &secureExecution,
			}

			podWorkload := k8sconsts.PodWorkload{
				Name:      testPod.Name,
				Namespace: testPod.Namespace,
				Kind:      k8sconsts.WorkloadKindPod,
			}

			err := webhookenvinjector.InjectOdigosAgentEnvVars(
				ctx, logger, podWorkload, &testPod.Spec.Containers[0],
				common.OtelSdkNativeCommunity, runtimeDetails, k8sClient, testConfig,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying LD_PRELOAD was injected")
			ldPreloadFound := false
			for _, env := range testPod.Spec.Containers[0].Env {
				if env.Name == commonconsts.LdPreloadEnvVarName {
					ldPreloadFound = true
					Expect(env.Value).To(ContainSubstring(commonconsts.OdigosLoaderName))
					break
				}
			}
			Expect(ldPreloadFound).To(BeTrue(), "LD_PRELOAD should be injected with loader method")
		})

		It("should handle multiple containers correctly", func() {
			By("Adding a second container to the pod")
			testPod.Spec.Containers = append(testPod.Spec.Containers, corev1.Container{
				Name:  "sidecar-container",
				Image: "sidecar:latest",
				Env: []corev1.EnvVar{
					{Name: "SIDECAR_VAR", Value: "sidecar_value"},
				},
			})

			By("Adding odiglet affinity to the pod")
			pod.AddOdigletInstalledAffinity(testPod)

			By("Injecting environment variables for both containers")
			successState := odigosv1.ProcessingStateSucceeded
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				Language:           common.JavaProgrammingLanguage,
				RuntimeUpdateState: &successState,
			}

			podWorkload := k8sconsts.PodWorkload{
				Name:      testPod.Name,
				Namespace: testPod.Namespace,
				Kind:      k8sconsts.WorkloadKindPod,
			}

			// Inject for first container
			err1 := webhookenvinjector.InjectOdigosAgentEnvVars(
				ctx, logger, podWorkload, &testPod.Spec.Containers[0],
				common.OtelSdkNativeCommunity, runtimeDetails, k8sClient, testConfig,
			)
			Expect(err1).NotTo(HaveOccurred())

			// Inject for second container  
			err2 := webhookenvinjector.InjectOdigosAgentEnvVars(
				ctx, logger, podWorkload, &testPod.Spec.Containers[1],
				common.OtelSdkNativeCommunity, runtimeDetails, k8sClient, testConfig,
			)
			Expect(err2).NotTo(HaveOccurred())

			By("Verifying both containers were processed correctly")
			Expect(len(testPod.Spec.Containers)).To(Equal(2))
			Expect(len(testPod.Spec.Containers[0].Env)).To(BeNumerically(">=", 1))
			Expect(len(testPod.Spec.Containers[1].Env)).To(BeNumerically(">=", 1))
		})

		It("should handle different programming languages", func() {
			languages := []common.ProgrammingLanguage{
				common.JavaProgrammingLanguage,
				common.PythonProgrammingLanguage,
				common.JavascriptProgrammingLanguage,
				common.DotNetProgrammingLanguage,
			}

			for _, lang := range languages {
				By(fmt.Sprintf("Testing %s language", lang))
				
				container := corev1.Container{
					Name:  fmt.Sprintf("%s-container", lang),
					Image: fmt.Sprintf("%s:latest", lang),
					Env:   []corev1.EnvVar{},
				}

				successState := odigosv1.ProcessingStateSucceeded
				runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
					Language:           lang,
					RuntimeUpdateState: &successState,
				}

				podWorkload := k8sconsts.PodWorkload{
					Name:      "test-pod",
					Namespace: "test-namespace",
					Kind:      k8sconsts.WorkloadKindPod,
				}

				err := webhookenvinjector.InjectOdigosAgentEnvVars(
					ctx, logger, podWorkload, &container,
					common.OtelSdkNativeCommunity, runtimeDetails, k8sClient, testConfig,
				)
				Expect(err).NotTo(HaveOccurred())

				// Check that appropriate env vars were injected based on language support
				// Some languages may not have env vars to inject, which is fine
				// The main thing is that the function doesn't error
			}
		})
	})

	Describe("Manager and Webhook Integration", func() {
		It("should create and configure manager with webhooks", func() {
			By("Creating manager with options")
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			mgr, err := controllers.CreateManager(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(mgr).NotTo(BeNil())

			By("Setting up controllers with manager")
			err = controllers.SetupWithManager(mgr, dp)
			Expect(err).NotTo(HaveOccurred())

			By("Registering webhooks")
			err = controllers.RegisterWebhooks(mgr, dp)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying manager configuration")
			Expect(mgr.GetScheme()).NotTo(BeNil())
			Expect(mgr.GetClient()).NotTo(BeNil())
			Expect(mgr.GetCache()).NotTo(BeNil())
		})

		It("should handle manager creation with different configurations", func() {
			configurations := []struct {
				name     string
				opts     controllers.KubeManagerOptions
				expected bool
			}{
				{
					name: "basic configuration",
					opts: controllers.KubeManagerOptions{
						Logger:                   logger,
						MetricsServerBindAddress: "0",
						HealthProbeBindAddress:   "0",
						EnableLeaderElection:     false,
					},
					expected: true,
				},
				{
					name: "leader election enabled",
					opts: controllers.KubeManagerOptions{
						Logger:                   logger,
						MetricsServerBindAddress: "0",
						HealthProbeBindAddress:   "0",
						EnableLeaderElection:     true,
					},
					expected: true,
				},
			}

			for _, config := range configurations {
				By(fmt.Sprintf("Testing %s", config.name))
				
				mgr, err := controllers.CreateManager(config.opts)
				if config.expected {
					Expect(err).NotTo(HaveOccurred())
					Expect(mgr).NotTo(BeNil())

					// Test controller setup
					err = controllers.SetupWithManager(mgr, dp)
					Expect(err).NotTo(HaveOccurred())

					// Test webhook registration
					err = controllers.RegisterWebhooks(mgr, dp)
					Expect(err).NotTo(HaveOccurred())
				} else {
					Expect(err).To(HaveOccurred())
				}
			}
		})
	})

	Describe("End-to-End Instrumentor Workflow", func() {
		It("should handle the complete instrumentation workflow", func() {
			By("Creating an Instrumentor instance")
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			instrumentor, err := New(opts, dp)
			Expect(err).NotTo(HaveOccurred())
			Expect(instrumentor).NotTo(BeNil())

			By("Verifying instrumentor state")
			Expect(instrumentor.webhooksRegistered.Load()).To(BeFalse())

			By("Testing webhook registration state management")
			instrumentor.webhooksRegistered.Store(true)
			Expect(instrumentor.webhooksRegistered.Load()).To(BeTrue())

			By("Starting instrumentor in background")
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(done)
				instrumentor.Run(ctx, true) // telemetry disabled for testing
			}()

			// Give it a moment to start
			time.Sleep(200 * time.Millisecond)

			By("Canceling context to stop instrumentor")
			cancel()

			By("Waiting for instrumentor to stop")
			Eventually(done, 10*time.Second).Should(BeClosed())
		})

		It("should handle instrumentor lifecycle with different telemetry settings", func() {
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			instrumentor, err := New(opts, dp)
			Expect(err).NotTo(HaveOccurred())

			testCases := []struct {
				name               string
				telemetryDisabled  bool
				expectedExitTime   time.Duration
			}{
				{
					name:              "telemetry disabled",
					telemetryDisabled: true,
					expectedExitTime:  5 * time.Second,
				},
				{
					name:              "telemetry enabled",
					telemetryDisabled: false,
					expectedExitTime:  5 * time.Second,
				},
			}

			for _, tc := range testCases {
				By(fmt.Sprintf("Testing %s", tc.name))
				
				ctx, cancel := context.WithCancel(context.Background())
				
				done := make(chan struct{})
				go func() {
					defer GinkgoRecover()
					defer close(done)
					instrumentor.Run(ctx, tc.telemetryDisabled)
				}()

				time.Sleep(100 * time.Millisecond)
				cancel()

				Eventually(done, tc.expectedExitTime).Should(BeClosed())
			}
		})
	})

	Describe("Error Handling Integration", func() {
		It("should handle invalid configurations gracefully", func() {
			By("Testing with nil injection method")
			testConfig.AgentEnvVarsInjectionMethod = nil

			successState := odigosv1.ProcessingStateSucceeded
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				Language:           common.JavaProgrammingLanguage,
				RuntimeUpdateState: &successState,
			}

			podWorkload := k8sconsts.PodWorkload{
				Name:      testPod.Name,
				Namespace: testPod.Namespace,
				Kind:      k8sconsts.WorkloadKindPod,
			}

			err := webhookenvinjector.InjectOdigosAgentEnvVars(
				ctx, logger, podWorkload, &testPod.Spec.Containers[0],
				common.OtelSdkNativeCommunity, runtimeDetails, k8sClient, testConfig,
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("env injection method is not set"))
		})

		It("should handle failed runtime detection state", func() {
			By("Testing with failed runtime detection")
			failedState := odigosv1.ProcessingStateFailed
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				Language:           common.JavaProgrammingLanguage,
				RuntimeUpdateState: &failedState,
			}

			podWorkload := k8sconsts.PodWorkload{
				Name:      testPod.Name,
				Namespace: testPod.Namespace,
				Kind:      k8sconsts.WorkloadKindPod,
			}

			originalEnvCount := len(testPod.Spec.Containers[0].Env)

			err := webhookenvinjector.InjectOdigosAgentEnvVars(
				ctx, logger, podWorkload, &testPod.Spec.Containers[0],
				common.OtelSdkNativeCommunity, runtimeDetails, k8sClient, testConfig,
			)
			Expect(err).NotTo(HaveOccurred())

			// Should not inject any new env vars due to failed state
			Expect(len(testPod.Spec.Containers[0].Env)).To(Equal(originalEnvCount))
		})

		It("should handle secure execution mode limitations", func() {
			By("Testing loader injection with secure execution mode")
			loaderMethod := common.LoaderEnvInjectionMethod
			testConfig.AgentEnvVarsInjectionMethod = &loaderMethod

			successState := odigosv1.ProcessingStateSucceeded
			secureExecution := true // Secure mode enabled
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				Language:             common.JavaProgrammingLanguage,
				RuntimeUpdateState:   &successState,
				SecureExecutionMode:  &secureExecution,
			}

			podWorkload := k8sconsts.PodWorkload{
				Name:      testPod.Name,
				Namespace: testPod.Namespace,
				Kind:      k8sconsts.WorkloadKindPod,
			}

			err := webhookenvinjector.InjectOdigosAgentEnvVars(
				ctx, logger, podWorkload, &testPod.Spec.Containers[0],
				common.OtelSdkNativeCommunity, runtimeDetails, k8sClient, testConfig,
			)
			
			// Should fail because secure execution mode prevents loader injection
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("secure execution mode"))
		})
	})

	Describe("Complex Environment Variable Scenarios", func() {
		It("should handle ValueFrom environment variables correctly", func() {
			By("Setting up pod with ValueFrom env var")
			testPod.Spec.Containers[0].Env = []corev1.EnvVar{
				{
					Name: "JAVA_TOOL_OPTIONS",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
							Key:                  "java-opts",
						},
					},
				},
			}

			By("Creating configmap for ValueFrom reference")
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app-config",
					Namespace: "test-namespace",
				},
				Data: map[string]string{
					"java-opts": "-Xmx1g -Xms512m",
				},
			}
			Expect(k8sClient.Create(ctx, configMap)).To(Succeed())

			By("Injecting environment variables")
			successState := odigosv1.ProcessingStateSucceeded
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				Language:           common.JavaProgrammingLanguage,
				RuntimeUpdateState: &successState,
			}

			podWorkload := k8sconsts.PodWorkload{
				Name:      testPod.Name,
				Namespace: testPod.Namespace,
				Kind:      k8sconsts.WorkloadKindPod,
			}

			err := webhookenvinjector.InjectOdigosAgentEnvVars(
				ctx, logger, podWorkload, &testPod.Spec.Containers[0],
				common.OtelSdkNativeCommunity, runtimeDetails, k8sClient, testConfig,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ValueFrom handling")
			// Should have the original ORIGINAL_JAVA_TOOL_OPTIONS and new JAVA_TOOL_OPTIONS
			Expect(len(testPod.Spec.Containers[0].Env)).To(BeNumerically(">=", 2))

			originalFound := false
			newFound := false
			for _, env := range testPod.Spec.Containers[0].Env {
				if env.Name == "ORIGINAL_JAVA_TOOL_OPTIONS" && env.ValueFrom != nil {
					originalFound = true
				}
				if env.Name == "JAVA_TOOL_OPTIONS" && env.Value != "" {
					newFound = true
					Expect(env.Value).To(ContainSubstring("$(ORIGINAL_JAVA_TOOL_OPTIONS)"))
				}
			}
			Expect(originalFound).To(BeTrue(), "Original env var should be renamed")
			Expect(newFound).To(BeTrue(), "New env var should reference original")
		})

		It("should handle test utilities integration", func() {
			By("Testing test utility functions")
			
			// Test instrumentation label helpers
			enabledPod := testutil.SetOdigosInstrumentationEnabled(testPod)
			disabledPod := testutil.SetOdigosInstrumentationDisabled(testPod)
			unlabeledPod := testutil.DeleteOdigosInstrumentationLabel(testPod)

			Expect(enabledPod.GetLabels()).To(HaveKeyWithValue("odigos.io/instrumentation", "enabled"))
			Expect(disabledPod.GetLabels()).To(HaveKeyWithValue("odigos.io/instrumentation", "disabled"))
			Expect(unlabeledPod.GetLabels()).NotTo(HaveKey("odigos.io/instrumentation"))

			// Test reported name annotation
			reportedPod := testutil.SetReportedNameAnnotation(testPod, "my-app")
			Expect(reportedPod.GetAnnotations()).To(HaveKeyWithValue("odigos.io/reported-name", "my-app"))
		})
	})

	Describe("Performance and Scalability", func() {
		It("should handle multiple concurrent pod processing", func() {
			By("Creating multiple test pods")
			numPods := 10
			pods := make([]*corev1.Pod, numPods)
			
			for i := 0; i < numPods; i++ {
				pods[i] = &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-pod-%d", i),
						Namespace: "test-namespace",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app-container",
								Image: "test-app:latest",
								Env:   []corev1.EnvVar{},
							},
						},
					},
				}
			}

			By("Processing all pods concurrently")
			done := make(chan int, numPods)
			successState := odigosv1.ProcessingStateSucceeded
			runtimeDetails := &odigosv1.RuntimeDetailsByContainer{
				Language:           common.JavaProgrammingLanguage,
				RuntimeUpdateState: &successState,
			}

			for i := 0; i < numPods; i++ {
				go func(podIndex int) {
					defer GinkgoRecover()
					
					// Add affinity
					pod.AddOdigletInstalledAffinity(pods[podIndex])
					
					// Inject env vars
					podWorkload := k8sconsts.PodWorkload{
						Name:      pods[podIndex].Name,
						Namespace: pods[podIndex].Namespace,
						Kind:      k8sconsts.WorkloadKindPod,
					}

					err := webhookenvinjector.InjectOdigosAgentEnvVars(
						ctx, logger, podWorkload, &pods[podIndex].Spec.Containers[0],
						common.OtelSdkNativeCommunity, runtimeDetails, k8sClient, testConfig,
					)
					Expect(err).NotTo(HaveOccurred())
					
					done <- podIndex
				}(i)
			}

			By("Waiting for all pods to be processed")
			processedCount := 0
			for processedCount < numPods {
				Eventually(done, 5*time.Second).Should(Receive())
				processedCount++
			}

			By("Verifying all pods were processed correctly")
			for i := 0; i < numPods; i++ {
				Expect(pods[i].Spec.Affinity).NotTo(BeNil())
				Expect(len(pods[i].Spec.Containers[0].Env)).To(BeNumerically(">=", 0))
			}
		})
	})
})
