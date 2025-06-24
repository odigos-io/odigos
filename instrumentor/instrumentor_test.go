package instrumentor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	testEnv    *envtest.Environment
	k8sClient  client.Client
	testScheme *runtime.Scheme
)

func TestInstrumentor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Instrumentor Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{}

	var err error
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	testScheme = runtime.NewScheme()
	err = clientgoscheme.AddToScheme(testScheme)
	Expect(err).NotTo(HaveOccurred())
	err = odigosv1.AddToScheme(testScheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: testScheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Instrumentor", func() {
	var (
		logger logr.Logger
		dp     *distros.Provider
	)

	BeforeEach(func() {
		logger = logr.Discard()
		dp = &distros.Provider{}
	})

	Describe("New", func() {
		It("should create a new Instrumentor instance successfully", func() {
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			instrumentor, err := New(opts, dp)
			Expect(err).NotTo(HaveOccurred())
			Expect(instrumentor).NotTo(BeNil())
			Expect(instrumentor.mgr).NotTo(BeNil())
			Expect(instrumentor.logger).To(Equal(logger))
			Expect(instrumentor.dp).To(Equal(dp))
			Expect(instrumentor.certReady).NotTo(BeNil())
			Expect(instrumentor.webhooksRegistered).NotTo(BeNil())
			Expect(instrumentor.webhooksRegistered.Load()).To(BeFalse())
		})

		It("should create instrumentor with different configurations", func() {
			testCases := []struct {
				name          string
				opts          controllers.KubeManagerOptions
				shouldSucceed bool
			}{
				{
					name: "default config",
					opts: controllers.KubeManagerOptions{
						Logger:                   logger,
						MetricsServerBindAddress: "0",
						HealthProbeBindAddress:   "0",
						EnableLeaderElection:     false,
					},
					shouldSucceed: true,
				},
				{
					name: "leader election enabled",
					opts: controllers.KubeManagerOptions{
						Logger:                   logger,
						MetricsServerBindAddress: "0",
						HealthProbeBindAddress:   "0",
						EnableLeaderElection:     true,
					},
					shouldSucceed: true,
				},
				{
					name: "different port bindings",
					opts: controllers.KubeManagerOptions{
						Logger:                   logger,
						MetricsServerBindAddress: ":9090",
						HealthProbeBindAddress:   ":9091",
						EnableLeaderElection:     false,
					},
					shouldSucceed: true,
				},
			}

			for _, tc := range testCases {
				It(fmt.Sprintf("should handle %s", tc.name), func() {
					instrumentor, err := New(tc.opts, dp)
					if tc.shouldSucceed {
						Expect(err).NotTo(HaveOccurred())
						Expect(instrumentor).NotTo(BeNil())
					} else {
						Expect(err).To(HaveOccurred())
						Expect(instrumentor).To(BeNil())
					}
				})
			}
		})

		It("should handle nil distros provider", func() {
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			instrumentor, err := New(opts, nil)
			// Should still create successfully, but with nil dp
			Expect(err).NotTo(HaveOccurred())
			Expect(instrumentor).NotTo(BeNil())
			Expect(instrumentor.dp).To(BeNil())
		})
	})

	Describe("Run", func() {
		var (
			instrumentor *Instrumentor
			ctx          context.Context
			cancel       context.CancelFunc
		)

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.Background())
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			var err error
			instrumentor, err = New(opts, dp)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			cancel()
		})

		It("should start and stop gracefully with telemetry disabled", func() {
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				instrumentor.Run(ctx, true) // telemetry disabled
				close(done)
			}()

			// Give it a moment to start
			time.Sleep(100 * time.Millisecond)

			// Cancel the context to stop the instrumentor
			cancel()

			// Wait for it to finish
			Eventually(done, 5*time.Second).Should(BeClosed())
		})

		It("should start and stop gracefully with telemetry enabled", func() {
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				instrumentor.Run(ctx, false) // telemetry enabled
				close(done)
			}()

			// Give it a moment to start
			time.Sleep(100 * time.Millisecond)

			// Cancel the context to stop the instrumentor
			cancel()

			// Wait for it to finish
			Eventually(done, 5*time.Second).Should(BeClosed())
		})

		It("should handle context cancellation immediately", func() {
			// Cancel context before starting
			cancel()

			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				instrumentor.Run(ctx, true)
				close(done)
			}()

			// Should exit quickly since context is already cancelled
			Eventually(done, 2*time.Second).Should(BeClosed())
		})

		It("should handle multiple concurrent runs", func() {
			ctx1, cancel1 := context.WithCancel(context.Background())
			ctx2, cancel2 := context.WithCancel(context.Background())
			defer cancel1()
			defer cancel2()

			done1 := make(chan struct{})
			done2 := make(chan struct{})

			go func() {
				defer GinkgoRecover()
				instrumentor.Run(ctx1, true)
				close(done1)
			}()

			go func() {
				defer GinkgoRecover()
				instrumentor.Run(ctx2, false)
				close(done2)
			}()

			time.Sleep(100 * time.Millisecond)

			cancel1()
			cancel2()

			Eventually(done1, 5*time.Second).Should(BeClosed())
			Eventually(done2, 5*time.Second).Should(BeClosed())
		})
	})

	Describe("webhooksRegistered atomic bool", func() {
		It("should initialize as false and be settable", func() {
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			instrumentor, err := New(opts, dp)
			Expect(err).NotTo(HaveOccurred())

			Expect(instrumentor.webhooksRegistered.Load()).To(BeFalse())

			instrumentor.webhooksRegistered.Store(true)
			Expect(instrumentor.webhooksRegistered.Load()).To(BeTrue())

			instrumentor.webhooksRegistered.Store(false)
			Expect(instrumentor.webhooksRegistered.Load()).To(BeFalse())
		})

		It("should be thread-safe", func() {
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			instrumentor, err := New(opts, dp)
			Expect(err).NotTo(HaveOccurred())

			// Test concurrent access
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				for i := 0; i < 100; i++ {
					instrumentor.webhooksRegistered.Store(true)
					instrumentor.webhooksRegistered.Store(false)
				}
				close(done)
			}()

			go func() {
				defer GinkgoRecover()
				for i := 0; i < 100; i++ {
					_ = instrumentor.webhooksRegistered.Load()
				}
			}()

			Eventually(done, 5*time.Second).Should(BeClosed())
		})
	})

	Describe("Instrumentor fields validation", func() {
		It("should have all required fields set after creation", func() {
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			instrumentor, err := New(opts, dp)
			Expect(err).NotTo(HaveOccurred())

			// Validate all fields are properly initialized
			Expect(instrumentor.mgr).NotTo(BeNil(), "Manager should not be nil")
			Expect(instrumentor.logger).NotTo(BeNil(), "Logger should not be nil")
			Expect(instrumentor.certReady).NotTo(BeNil(), "CertReady channel should not be nil")
			Expect(instrumentor.dp).NotTo(BeNil(), "Distros provider should not be nil")
			Expect(instrumentor.webhooksRegistered).NotTo(BeNil(), "WebhooksRegistered should not be nil")

			// Validate initial state
			Expect(instrumentor.webhooksRegistered.Load()).To(BeFalse(), "WebhooksRegistered should be false initially")
		})

		It("should preserve logger instance", func() {
			customLogger := logr.Discard().WithName("test-instrumentor")
			opts := controllers.KubeManagerOptions{
				Logger:                   customLogger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			instrumentor, err := New(opts, dp)
			Expect(err).NotTo(HaveOccurred())
			Expect(instrumentor.logger).To(BeIdenticalTo(customLogger))
		})

		It("should preserve distros provider instance", func() {
			customDP := &distros.Provider{}
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			instrumentor, err := New(opts, customDP)
			Expect(err).NotTo(HaveOccurred())
			Expect(instrumentor.dp).To(BeIdenticalTo(customDP))
		})
	})
})
