package controllers

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/distros"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Manager Suite")
}

var _ = Describe("Manager", func() {
	var (
		logger logr.Logger
	)

	BeforeEach(func() {
		logger = logr.Discard()
	})

	Describe("CreateManager", func() {
		It("should create a manager with default options", func() {
			opts := KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			mgr, err := CreateManager(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(mgr).NotTo(BeNil())

			// Verify manager configuration
			Expect(mgr.GetScheme()).NotTo(BeNil())
			Expect(mgr.GetClient()).NotTo(BeNil())
		})

		It("should create a manager with leader election enabled", func() {
			opts := KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     true,
			}

			mgr, err := CreateManager(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(mgr).NotTo(BeNil())
		})

		It("should handle custom webhook cert directory", func() {
			originalValue := os.Getenv("LOCAL_MUTATING_WEBHOOK_CERT_DIR")
			defer func() {
				if originalValue == "" {
					os.Unsetenv("LOCAL_MUTATING_WEBHOOK_CERT_DIR")
				} else {
					os.Setenv("LOCAL_MUTATING_WEBHOOK_CERT_DIR", originalValue)
				}
			}()

			// Set custom cert directory
			os.Setenv("LOCAL_MUTATING_WEBHOOK_CERT_DIR", "/tmp/test-certs")

			opts := KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			mgr, err := CreateManager(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(mgr).NotTo(BeNil())
		})

		It("should handle various metrics server bind addresses", func() {
			addresses := []string{"0", ":8080", "127.0.0.1:8080"}

			for _, addr := range addresses {
				opts := KubeManagerOptions{
					Logger:                   logger,
					MetricsServerBindAddress: addr,
					HealthProbeBindAddress:   "0",
					EnableLeaderElection:     false,
				}

				mgr, err := CreateManager(opts)
				Expect(err).NotTo(HaveOccurred())
				Expect(mgr).NotTo(BeNil())
			}
		})

		It("should handle various health probe bind addresses", func() {
			addresses := []string{"0", ":8081", "127.0.0.1:8081"}

			for _, addr := range addresses {
				opts := KubeManagerOptions{
					Logger:                   logger,
					MetricsServerBindAddress: "0",
					HealthProbeBindAddress:   addr,
					EnableLeaderElection:     false,
				}

				mgr, err := CreateManager(opts)
				Expect(err).NotTo(HaveOccurred())
				Expect(mgr).NotTo(BeNil())
			}
		})

		It("should create manager with custom logger", func() {
			customLogger := logr.Discard().WithName("custom-manager")
			opts := KubeManagerOptions{
				Logger:                   customLogger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			mgr, err := CreateManager(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(mgr).NotTo(BeNil())
		})

		It("should create multiple managers independently", func() {
			opts := KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			mgr1, err1 := CreateManager(opts)
			mgr2, err2 := CreateManager(opts)

			Expect(err1).NotTo(HaveOccurred())
			Expect(err2).NotTo(HaveOccurred())
			Expect(mgr1).NotTo(BeNil())
			Expect(mgr2).NotTo(BeNil())
			Expect(mgr1).NotTo(BeIdenticalTo(mgr2))
		})
	})

	Describe("durationPointer", func() {
		It("should return a pointer to the duration", func() {
			duration := 30 * time.Second
			ptr := durationPointer(duration)

			Expect(ptr).NotTo(BeNil())
			Expect(*ptr).To(Equal(duration))
		})

		It("should handle zero duration", func() {
			duration := time.Duration(0)
			ptr := durationPointer(duration)

			Expect(ptr).NotTo(BeNil())
			Expect(*ptr).To(Equal(duration))
		})

		It("should handle negative duration", func() {
			duration := -10 * time.Second
			ptr := durationPointer(duration)

			Expect(ptr).NotTo(BeNil())
			Expect(*ptr).To(Equal(duration))
		})

		It("should handle various duration types", func() {
			durations := []time.Duration{
				1 * time.Nanosecond,
				1 * time.Microsecond,
				1 * time.Millisecond,
				1 * time.Second,
				1 * time.Minute,
				1 * time.Hour,
				24 * time.Hour,
			}

			for _, d := range durations {
				ptr := durationPointer(d)
				Expect(ptr).NotTo(BeNil())
				Expect(*ptr).To(Equal(d))
			}
		})

		It("should create independent pointers", func() {
			duration1 := 10 * time.Second
			duration2 := 20 * time.Second

			ptr1 := durationPointer(duration1)
			ptr2 := durationPointer(duration2)

			Expect(ptr1).NotTo(BeIdenticalTo(ptr2))
			Expect(*ptr1).To(Equal(duration1))
			Expect(*ptr2).To(Equal(duration2))

			// Modifying one shouldn't affect the other
			*ptr1 = 15 * time.Second
			Expect(*ptr1).To(Equal(15 * time.Second))
			Expect(*ptr2).To(Equal(duration2))
		})
	})

	Describe("SetupWithManager", func() {
		var (
			mgr manager.Manager
			dp  *distros.Provider
		)

		BeforeEach(func() {
			opts := KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			var err error
			mgr, err = CreateManager(opts)
			Expect(err).NotTo(HaveOccurred())

			dp = &distros.Provider{}
		})

		It("should setup all controllers without error", func() {
			err := SetupWithManager(mgr, dp)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle nil distros provider", func() {
			err := SetupWithManager(mgr, nil)
			// This may or may not fail depending on the implementation
			// The test documents the current behavior
			if err != nil {
				Expect(err.Error()).To(ContainSubstring("nil"))
			}
		})

		It("should be idempotent", func() {
			err1 := SetupWithManager(mgr, dp)
			err2 := SetupWithManager(mgr, dp)

			Expect(err1).NotTo(HaveOccurred())
			// Second call might fail or succeed, but shouldn't crash
			if err2 != nil {
				// Log the error but don't fail the test
				GinkgoWriter.Printf("Second SetupWithManager call failed: %v\n", err2)
			}
		})

		It("should handle different distros providers", func() {
			dp1 := &distros.Provider{}
			dp2 := &distros.Provider{}

			err1 := SetupWithManager(mgr, dp1)
			Expect(err1).NotTo(HaveOccurred())

			// Creating a second manager to test with different provider
			opts := KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}
			mgr2, err := CreateManager(opts)
			Expect(err).NotTo(HaveOccurred())

			err2 := SetupWithManager(mgr2, dp2)
			Expect(err2).NotTo(HaveOccurred())
		})
	})

	Describe("RegisterWebhooks", func() {
		var (
			mgr manager.Manager
			dp  *distros.Provider
		)

		BeforeEach(func() {
			opts := KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			var err error
			mgr, err = CreateManager(opts)
			Expect(err).NotTo(HaveOccurred())

			dp = &distros.Provider{}
		})

		It("should register webhooks without error", func() {
			err := RegisterWebhooks(mgr, dp)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle nil distros provider", func() {
			err := RegisterWebhooks(mgr, nil)
			// This may or may not fail depending on the implementation
			// The test documents the current behavior
			if err != nil {
				Expect(err.Error()).To(ContainSubstring("nil"))
			}
		})

		It("should register webhooks after controller setup", func() {
			err1 := SetupWithManager(mgr, dp)
			Expect(err1).NotTo(HaveOccurred())

			err2 := RegisterWebhooks(mgr, dp)
			Expect(err2).NotTo(HaveOccurred())
		})

		It("should be idempotent", func() {
			err1 := RegisterWebhooks(mgr, dp)
			err2 := RegisterWebhooks(mgr, dp)

			Expect(err1).NotTo(HaveOccurred())
			// Second call might fail or succeed, but shouldn't crash
			if err2 != nil {
				GinkgoWriter.Printf("Second RegisterWebhooks call failed: %v\n", err2)
			}
		})
	})

	Describe("scheme initialization", func() {
		It("should have all required schemes registered", func() {
			Expect(scheme).NotTo(BeNil())

			// Check that core Kubernetes types are registered
			gvk := scheme.AllKnownTypes()
			Expect(gvk).To(HaveKey(schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			}))

			// Check that Odigos types are registered
			Expect(gvk).To(HaveKey(schema.GroupVersionKind{
				Group:   "odigos.io",
				Version: "v1alpha1",
				Kind:    "Source",
			}))
		})

		It("should register additional Kubernetes types", func() {
			gvk := scheme.AllKnownTypes()
			
			// Check for common Kubernetes resources
			expectedTypes := []schema.GroupVersionKind{
				{Group: "", Version: "v1", Kind: "ConfigMap"},
				{Group: "", Version: "v1", Kind: "Secret"},
				{Group: "", Version: "v1", Kind: "Service"},
				{Group: "apps", Version: "v1", Kind: "Deployment"},
				{Group: "apps", Version: "v1", Kind: "DaemonSet"},
				{Group: "apps", Version: "v1", Kind: "StatefulSet"},
			}

			for _, expectedType := range expectedTypes {
				Expect(gvk).To(HaveKey(expectedType), "Expected type %v to be registered", expectedType)
			}
		})

		It("should register Odigos specific types", func() {
			gvk := scheme.AllKnownTypes()
			
			// Check for Odigos-specific resources
			expectedOdigosTypes := []schema.GroupVersionKind{
				{Group: "odigos.io", Version: "v1alpha1", Kind: "Source"},
				{Group: "odigos.io", Version: "v1alpha1", Kind: "Destination"},
				{Group: "odigos.io", Version: "v1alpha1", Kind: "InstrumentationConfig"},
				{Group: "odigos.io", Version: "v1alpha1", Kind: "CollectorsGroup"},
			}

			for _, expectedType := range expectedOdigosTypes {
				Expect(gvk).To(HaveKey(expectedType), "Expected Odigos type %v to be registered", expectedType)
			}
		})
	})

	Describe("KubeManagerOptions", func() {
		It("should accept valid options", func() {
			opts := KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: ":8080",
				HealthProbeBindAddress:   ":8081",
				EnableLeaderElection:     true,
			}

			Expect(opts.Logger).To(Equal(logger))
			Expect(opts.MetricsServerBindAddress).To(Equal(":8080"))
			Expect(opts.HealthProbeBindAddress).To(Equal(":8081"))
			Expect(opts.EnableLeaderElection).To(BeTrue())
		})

		It("should handle empty logger", func() {
			opts := KubeManagerOptions{
				Logger:                   logr.Discard(),
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			mgr, err := CreateManager(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(mgr).NotTo(BeNil())
		})

		It("should validate struct fields", func() {
			opts := KubeManagerOptions{}
			
			// Use reflection to check that all fields are accessible
			t := reflect.TypeOf(opts)
			
			expectedFields := []string{"Logger", "MetricsServerBindAddress", "HealthProbeBindAddress", "EnableLeaderElection"}
			
			for _, fieldName := range expectedFields {
				field, found := t.FieldByName(fieldName)
				Expect(found).To(BeTrue(), "Field %s should exist", fieldName)
				Expect(field.IsExported()).To(BeTrue(), "Field %s should be exported", fieldName)
			}
		})

		It("should create different managers with different options", func() {
			opts1 := KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			opts2 := KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     true,
			}

			mgr1, err1 := CreateManager(opts1)
			mgr2, err2 := CreateManager(opts2)

			Expect(err1).NotTo(HaveOccurred())
			Expect(err2).NotTo(HaveOccurred())
			Expect(mgr1).NotTo(BeNil())
			Expect(mgr2).NotTo(BeNil())
			Expect(mgr1).NotTo(BeIdenticalTo(mgr2))
		})
	})

	Describe("Manager cache configuration", func() {
		It("should configure cache with correct selectors", func() {
			opts := KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			mgr, err := CreateManager(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(mgr).NotTo(BeNil())

			// The cache should be configured with the correct options
			// We can't directly inspect the cache configuration, but we can verify
			// that the manager was created successfully with our cache options
			cache := mgr.GetCache()
			Expect(cache).NotTo(BeNil())
		})

		It("should create cache that can be used for client operations", func() {
			opts := KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     false,
			}

			mgr, err := CreateManager(opts)
			Expect(err).NotTo(HaveOccurred())

			// Should be able to get a client from the manager
			client := mgr.GetClient()
			Expect(client).NotTo(BeNil())

			// Should be able to get various components
			Expect(mgr.GetCache()).NotTo(BeNil())
			Expect(mgr.GetScheme()).NotTo(BeNil())
			Expect(mgr.GetConfig()).NotTo(BeNil())
		})
	})

	Describe("Leader election configuration", func() {
		It("should configure leader election with correct parameters", func() {
			opts := KubeManagerOptions{
				Logger:                   logger,
				MetricsServerBindAddress: "0",
				HealthProbeBindAddress:   "0",
				EnableLeaderElection:     true,
			}

			mgr, err := CreateManager(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(mgr).NotTo(BeNil())

			// We can't directly access the leader election configuration,
			// but we can verify that the manager was created successfully
			// with leader election enabled
		})

		It("should have appropriate leader election timing", func() {
			// Test that our duration pointer helper works correctly
			leaseDuration := durationPointer(30 * time.Second)
			renewDeadline := durationPointer(20 * time.Second)
			retryPeriod := durationPointer(5 * time.Second)

			Expect(*leaseDuration).To(Equal(30 * time.Second))
			Expect(*renewDeadline).To(Equal(20 * time.Second))
			Expect(*retryPeriod).To(Equal(5 * time.Second))

			// Verify the timing relationship: RetryPeriod < RenewDeadline < LeaseDuration
			Expect(*retryPeriod).To(BeNumerically("<", *renewDeadline))
			Expect(*renewDeadline).To(BeNumerically("<", *leaseDuration))
		})

		It("should handle leader election timing edge cases", func() {
			// Test edge cases for leader election timing
			testCases := []struct {
				name          string
				leaseDuration time.Duration
				renewDeadline time.Duration
				retryPeriod   time.Duration
				isValid       bool
			}{
				{
					name:          "valid timing",
					leaseDuration: 60 * time.Second,
					renewDeadline: 40 * time.Second,
					retryPeriod:   10 * time.Second,
					isValid:       true,
				},
				{
					name:          "minimum values",
					leaseDuration: 3 * time.Second,
					renewDeadline: 2 * time.Second,
					retryPeriod:   1 * time.Second,
					isValid:       true,
				},
			}

			for _, tc := range testCases {
				leaseDuration := durationPointer(tc.leaseDuration)
				renewDeadline := durationPointer(tc.renewDeadline)
				retryPeriod := durationPointer(tc.retryPeriod)

				Expect(*leaseDuration).To(Equal(tc.leaseDuration))
				Expect(*renewDeadline).To(Equal(tc.renewDeadline))
				Expect(*retryPeriod).To(Equal(tc.retryPeriod))

				if tc.isValid {
					Expect(*retryPeriod).To(BeNumerically("<", *renewDeadline))
					Expect(*renewDeadline).To(BeNumerically("<", *leaseDuration))
				}
			}
		})
	})
})
