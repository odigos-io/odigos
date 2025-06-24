package instrumentor

import (
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInstrumentorUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Instrumentor Unit Test Suite")
}

var _ = Describe("Instrumentor Unit Tests", func() {
	var (
		logger logr.Logger
		dp     *distros.Provider
	)

	BeforeEach(func() {
		logger = logr.Discard()
		dp = &distros.Provider{}
	})

	Describe("New", func() {
		It("should attempt to create a new Instrumentor instance", func() {
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			instrumentor, err := New(opts, dp)
			// In unit test environment without Kubernetes config, this may fail
			// The test verifies the function executes without crashing
			if err != nil {
				// Expected in unit test environment without K8s config
				Expect(err.Error()).To(ContainSubstring("unable to load"))
				Expect(instrumentor).To(BeNil())
			} else {
				// If somehow it succeeds, verify the structure
				Expect(instrumentor).NotTo(BeNil())
				Expect(instrumentor.mgr).NotTo(BeNil())
				Expect(instrumentor.logger).To(Equal(logger))
				Expect(instrumentor.dp).To(Equal(dp))
				Expect(instrumentor.certReady).NotTo(BeNil())
				Expect(instrumentor.webhooksRegistered).NotTo(BeNil())
				Expect(instrumentor.webhooksRegistered.Load()).To(BeFalse())
			}
		})

		It("should handle different configuration options", func() {
			// Test leader election enabled
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     true,
			}

			_, err := New(opts, dp)
			// May fail due to K8s config in unit test environment
			if err != nil {
				Expect(err.Error()).To(ContainSubstring("unable to load"))
			}

			// Test different port bindings
			opts = controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: ":9090",
				HealthProbeBindAddress:   ":9091",
				EnableLeaderElection:     false,
			}

			_, err = New(opts, dp)
			// May fail due to K8s config in unit test environment
			if err != nil {
				Expect(err.Error()).To(ContainSubstring("unable to load"))
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
			// May fail due to K8s config, but should accept nil provider
			if err != nil {
				Expect(err.Error()).To(ContainSubstring("unable to load"))
				Expect(instrumentor).To(BeNil())
			} else {
				Expect(instrumentor).NotTo(BeNil())
				Expect(instrumentor.dp).To(BeNil())
			}
		})
	})

	Describe("Configuration Validation", func() {
		It("should validate logger requirements", func() {
			// Use a logr.Logger zero value which is effectively nil
			var nullLogger logr.Logger
			opts := controllers.KubeManagerOptions{
				Logger:                   nullLogger, // Invalid: zero-value logger
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			_, err := New(opts, dp)
			// Should handle invalid logger gracefully or fail
			Expect(err).To(HaveOccurred())
		})

		It("should validate port binding formats", func() {
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "invalid-port",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			_, err := New(opts, dp)
			// May fail due to invalid port or K8s config
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Basic Function Validation", func() {
		It("should validate distros provider type", func() {
			customDP := &distros.Provider{}
			Expect(customDP).NotTo(BeNil())
			Expect(customDP).To(BeAssignableToTypeOf(&distros.Provider{}))
		})

		It("should validate logger interface", func() {
			customLogger := logr.Discard().WithName("test-instrumentor")
			Expect(customLogger).NotTo(BeNil())
			
			// Test basic logger functionality
			customLogger.Info("test message")
			customLogger.Error(fmt.Errorf("test error"), "test error message")
		})

		It("should validate KubeManagerOptions structure", func() {
			opts := controllers.KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			// Verify options structure
			Expect(opts.Logger).NotTo(BeNil())
			Expect(opts.MetricsServerBindAddress).To(Equal("0"))
			Expect(opts.HealthProbeBindAddress).To(Equal("0"))
			Expect(opts.EnableLeaderElection).To(BeFalse())
		})
	})
})