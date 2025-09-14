/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package actions_test

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	sampling "github.com/odigos-io/odigos/autoscaler/controllers/actions/sampling"
	"github.com/odigos-io/odigos/common"
)

// Helper function to extract rule details from raw JSON
// If ruleIndex is -1, returns all rule details; otherwise returns the rule at the specified index
func getRuleDetailsFromRawJSON(rawJSON []byte, ruleType string, ruleIndex int) ([]map[string]interface{}, error) {
	var rawConfig map[string]interface{}
	if err := json.Unmarshal(rawJSON, &rawConfig); err != nil {
		return nil, err
	}

	// Access the appropriate rules array based on rule type
	var rules []interface{}
	var ok bool

	switch ruleType {
	case "global":
		rules, ok = rawConfig["global_rules"].([]interface{})
	case "service":
		rules, ok = rawConfig["service_rules"].([]interface{})
	case "endpoint":
		rules, ok = rawConfig["endpoint_rules"].([]interface{})
	default:
		return nil, fmt.Errorf("unknown rule type: %s", ruleType)
	}

	if !ok {
		return nil, fmt.Errorf("no rules found for type %s", ruleType)
	}

	if ruleIndex == -1 {
		// Return all rule details
		var ruleDetailsList []map[string]interface{}
		for _, ruleInterface := range rules {
			ruleMap, ok := ruleInterface.(map[string]interface{})
			if !ok {
				continue
			}

			ruleDetails, ok := ruleMap["rule_details"].(map[string]interface{})
			if !ok {
				continue
			}

			ruleDetailsList = append(ruleDetailsList, ruleDetails)
		}
		return ruleDetailsList, nil
	} else {
		// Return specific rule details
		if ruleIndex >= len(rules) {
			return nil, fmt.Errorf("rule index %d out of range", ruleIndex)
		}

		rule, ok := rules[ruleIndex].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid rule format")
		}

		ruleDetails, ok := rule["rule_details"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("no rule_details found")
		}

		return []map[string]interface{}{ruleDetails}, nil
	}
}

var _ = Describe("Action Controller", func() {
	const (
		ActionName      = "test-action"
		ActionNamespace = "default"
	)

	AfterEach(func() {
		cleanupResources()
	})

	Context("When creating an Action with Samplers", func() {
		It("Should create a Processor for probabilistic sampler", func() {
			By("Creating an Action with ProbabilisticSampler")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName,
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-probabilistic-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						ProbabilisticSampler: &actionv1.ProbabilisticSamplerConfig{
							SamplingPercentage: "50",
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that a Processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName,
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("probabilistic_sampler"))
			Expect(processor.Spec.OrderHint).Should(Equal(1))
		})

		It("Should create a Processor for other samplers", func() {
			By("Creating an Action with ErrorSampler")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-error",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-error-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						ErrorSampler: &actionv1.ErrorSamplerConfig{
							FallbackSamplingRatio: 10.0,
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that a Processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "sampling-processor",
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("odigossampling"))
			Expect(processor.Spec.OrderHint).Should(Equal(-24))
			ownerRefs := processor.GetOwnerReferences()
			Expect(len(ownerRefs)).Should(Equal(1))
			Expect(ownerRefs[0].Name).Should(Equal(ActionName + "-error"))
			Expect(ownerRefs[0].Kind).Should(Equal("Action"))
		})

		It("Should create a Processor for latency sampler", func() {
			By("Creating an Action with LatencySampler")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-latency",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-latency-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						LatencySampler: &actionv1.LatencySamplerConfig{
							EndpointsFilters: []actionv1.HttpRouteFilter{
								{
									ServiceName:             "test-service",
									HttpRoute:               "/api/test",
									MinimumLatencyThreshold: 100,
									FallbackSamplingRatio:   5.0,
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that a Processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "sampling-processor",
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("odigossampling"))
			Expect(processor.Spec.OrderHint).Should(Equal(-24))
			ownerRefs := processor.GetOwnerReferences()
			Expect(len(ownerRefs)).Should(Equal(1))
			Expect(ownerRefs[0].Name).Should(Equal(ActionName + "-latency"))
			Expect(ownerRefs[0].Kind).Should(Equal("Action"))
		})

		It("Should create a Processor for service name sampler", func() {
			By("Creating an Action with ServiceNameSampler")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-service",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-service-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						ServiceNameSampler: &actionv1.ServiceNameSamplerConfig{
							ServicesNameFilters: []actionv1.ServiceNameFilter{
								{
									ServiceName:           "test-service",
									SamplingRatio:         20.0,
									FallbackSamplingRatio: 5.0,
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that a Processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "sampling-processor",
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("odigossampling"))
			Expect(processor.Spec.OrderHint).Should(Equal(-24))
			ownerRefs := processor.GetOwnerReferences()
			Expect(len(ownerRefs)).Should(Equal(1))
			Expect(ownerRefs[0].Name).Should(Equal(ActionName + "-service"))
			Expect(ownerRefs[0].Kind).Should(Equal("Action"))
		})

		It("Should create a Processor for span attribute sampler", func() {
			By("Creating an Action with SpanAttributeSampler")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-span",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-span-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						SpanAttributeSampler: &actionv1.SpanAttributeSamplerConfig{
							AttributeFilters: []actionv1.SpanAttributeFilter{
								{
									ServiceName:           "test-service",
									AttributeKey:          "http.status_code",
									SamplingRatio:         15.0,
									FallbackSamplingRatio: 5.0,
									Condition: actionv1.AttributeCondition{
										StringCondition: &actionv1.StringAttributeCondition{
											Operation:     "equals",
											ExpectedValue: "500",
										},
									},
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that a Processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "sampling-processor",
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("odigossampling"))
			Expect(processor.Spec.OrderHint).Should(Equal(-24))
			ownerRefs := processor.GetOwnerReferences()
			Expect(len(ownerRefs)).Should(Equal(1))
			Expect(ownerRefs[0].Name).Should(Equal(ActionName + "-span"))
			Expect(ownerRefs[0].Kind).Should(Equal("Action"))
		})
	})

	Context("When updating an Action", func() {
		It("Should update the corresponding Processor", func() {
			By("Creating an Action with ErrorSampler")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-update",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-update-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						ErrorSampler: &actionv1.ErrorSamplerConfig{
							FallbackSamplingRatio: 10.0,
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Waiting for Processor to be created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "sampling-processor",
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Updating the Action")
			// Get the action again because it had its status updated after the first time
			Expect(k8sClient.Get(testCtx, types.NamespacedName{
				Name:      ActionName + "-update",
				Namespace: ActionNamespace,
			}, action)).Should(Succeed())
			action.Spec.Samplers.ErrorSampler.FallbackSamplingRatio = 20.0
			Expect(k8sClient.Update(testCtx, action)).Should(Succeed())

			By("Checking that the Processor is updated")
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "sampling-processor",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}
				// Use helper function to get rule details
				ruleDetailsList, err := getRuleDetailsFromRawJSON(processor.Spec.ProcessorConfig.Raw, "global", 0)
				if err != nil || len(ruleDetailsList) == 0 {
					return false
				}

				// Check the fallback_sampling_ratio
				fallbackRatio, ok := ruleDetailsList[0]["fallback_sampling_ratio"].(float64)
				if !ok {
					return false
				}

				return fallbackRatio == 20.0
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When creating an Action with legacy sampler objects", func() {
		It("Should merge all legacy sampler objects into a single processor with owner references", func() {
			By("Creating multiple legacy ErrorSampler objects")
			legacyErrorSampler1 := &actionv1.ErrorSampler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-legacy-error-1",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.ErrorSamplerSpec{
					ActionName:            "legacy-error-sampler-1",
					Signals:               []common.ObservabilitySignal{common.TracesObservabilitySignal},
					FallbackSamplingRatio: 10.0,
				},
			}
			Expect(k8sClient.Create(testCtx, legacyErrorSampler1)).Should(Succeed())

			legacyErrorSampler2 := &actionv1.ErrorSampler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-legacy-error-2",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.ErrorSamplerSpec{
					ActionName:            "legacy-error-sampler-2",
					Signals:               []common.ObservabilitySignal{common.TracesObservabilitySignal},
					FallbackSamplingRatio: 15.0,
				},
			}
			Expect(k8sClient.Create(testCtx, legacyErrorSampler2)).Should(Succeed())

			By("Creating a new Action with ErrorSampler")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-merge-test",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-merge-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						ErrorSampler: &actionv1.ErrorSamplerConfig{
							FallbackSamplingRatio: 25.0,
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that a single odigossampling Processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "sampling-processor",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}
				ownerRefs := processor.GetOwnerReferences()
				return len(ownerRefs) == 3
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("odigossampling"))
			Expect(processor.Spec.OrderHint).Should(Equal(-24))

			actualSamplingConfig := sampling.SamplingConfig{}
			json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &actualSamplingConfig)
			Expect(len(actualSamplingConfig.GlobalRules)).Should(Equal(3), "Expected 3 rules in the processor config", actualSamplingConfig.GlobalRules)

			rules := []float64{10.0, 15.0, 25.0}
			ruleDetailsList, err := getRuleDetailsFromRawJSON(processor.Spec.ProcessorConfig.Raw, "global", -1)
			Expect(err).ShouldNot(HaveOccurred())

			for _, rule := range rules {
				found := false
				for _, ruleDetails := range ruleDetailsList {
					fallbackRatio, ok := ruleDetails["fallback_sampling_ratio"].(float64)
					if !ok {
						continue
					}

					if fallbackRatio == rule {
						found = true
						break
					}
				}
				Expect(found).Should(BeTrue(), "Rule with fallback sampling ratio %f should be present", rule)
			}

			By("Verifying that the processor does not have owner references to all legacy objects")
			// The processor should not have owner references to all the legacy ErrorSampler objects

			// Verify that the owner references do not include the legacy ErrorSampler objects
			legacyObjectNames := []string{
				ActionName + "-legacy-error-1",
				ActionName + "-legacy-error-2",
			}

			ownerRefs := processor.GetOwnerReferences()
			for _, legacyName := range legacyObjectNames {
				found := false
				for _, ownerRef := range ownerRefs {
					if ownerRef.Name == legacyName && ownerRef.Kind == "ErrorSampler" {
						found = true
						break
					}
				}
				Expect(found).Should(BeFalse(), "Owner reference for legacy %s should not be present", legacyName)
			}
			objectNames := []string{
				odigosv1.ActionMigratedLegacyPrefix + ActionName + "-legacy-error-1",
				odigosv1.ActionMigratedLegacyPrefix + ActionName + "-legacy-error-2",
			}
			for _, objectName := range objectNames {
				found := false
				for _, ownerRef := range ownerRefs {
					if ownerRef.Name == objectName && ownerRef.Kind == "Action" {
						found = true
						break
					}
				}
				Expect(found).Should(BeTrue(), "Owner reference for %s should be present", objectName)
			}
		})

		It("Should merge legacy LatencySampler objects into a single processor", func() {
			testRules := []actionv1.HttpRouteFilter{
				{
					ServiceName:             "service-1",
					HttpRoute:               "/api/service1",
					MinimumLatencyThreshold: 100,
					FallbackSamplingRatio:   5.0,
				},
				{
					ServiceName:             "service-2",
					HttpRoute:               "/api/service2",
					MinimumLatencyThreshold: 200,
					FallbackSamplingRatio:   10.0,
				},
				{
					ServiceName:             "new-service",
					HttpRoute:               "/api/new",
					MinimumLatencyThreshold: 150,
					FallbackSamplingRatio:   8.0,
				},
			}

			By("Creating multiple legacy LatencySampler objects")
			legacyLatencySampler1 := &actionv1.LatencySampler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-legacy-latency-1",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.LatencySamplerSpec{
					ActionName:       "legacy-latency-sampler-1",
					Signals:          []common.ObservabilitySignal{common.TracesObservabilitySignal},
					EndpointsFilters: []actionv1.HttpRouteFilter{testRules[0]},
				},
			}
			Expect(k8sClient.Create(testCtx, legacyLatencySampler1)).Should(Succeed())

			legacyLatencySampler2 := &actionv1.LatencySampler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-legacy-latency-2",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.LatencySamplerSpec{
					ActionName:       "legacy-latency-sampler-2",
					Signals:          []common.ObservabilitySignal{common.TracesObservabilitySignal},
					EndpointsFilters: []actionv1.HttpRouteFilter{testRules[1]},
				},
			}
			Expect(k8sClient.Create(testCtx, legacyLatencySampler2)).Should(Succeed())

			By("Creating a new Action with LatencySampler")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-merge-latency-test",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-merge-latency-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						LatencySampler: &actionv1.LatencySamplerConfig{
							EndpointsFilters: []actionv1.HttpRouteFilter{testRules[2]},
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that a single odigossampling Processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "sampling-processor",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}
				ownerRefs := processor.GetOwnerReferences()
				return len(ownerRefs) == 3
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("odigossampling"))
			Expect(processor.Spec.OrderHint).Should(Equal(-24))

			actualSamplingConfig := sampling.SamplingConfig{}
			json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &actualSamplingConfig)
			Expect(len(actualSamplingConfig.EndpointRules)).Should(Equal(3), "Expected 3 rules in the processor config", actualSamplingConfig.EndpointRules)
			ruleDetailsList, err := getRuleDetailsFromRawJSON(processor.Spec.ProcessorConfig.Raw, "endpoint", -1)
			Expect(err).ShouldNot(HaveOccurred())

			for _, rule := range testRules {
				found := false
				for _, ruleDetails := range ruleDetailsList {
					// Extract fields from rule_details
					fallbackRatio, ok1 := ruleDetails["fallback_sampling_ratio"].(float64)
					httpRoute, ok2 := ruleDetails["http_route"].(string)
					serviceName, ok3 := ruleDetails["service_name"].(string)
					thresholdMs, ok4 := ruleDetails["threshold"].(float64)

					if !ok1 || !ok2 || !ok3 || !ok4 {
						continue
					}

					if fallbackRatio == rule.FallbackSamplingRatio &&
						httpRoute == rule.HttpRoute &&
						serviceName == rule.ServiceName &&
						int(thresholdMs) == rule.MinimumLatencyThreshold {
						found = true
						break
					}
				}
				Expect(found).Should(BeTrue(), "Rule with fallback sampling ratio %f, http route %s, service name %s, minimum latency threshold %d should be present", rule.FallbackSamplingRatio, rule.HttpRoute, rule.ServiceName, rule.MinimumLatencyThreshold)
			}

			By("Verifying that the processor does not have owner references to all legacy LatencySampler objects")
			ownerRefs := processor.GetOwnerReferences()

			// Verify that the owner references do not include the legacy LatencySampler objects
			legacyObjectNames := []string{
				ActionName + "-legacy-latency-1",
				ActionName + "-legacy-latency-2",
			}

			for _, legacyName := range legacyObjectNames {
				found := false
				for _, ownerRef := range ownerRefs {
					if ownerRef.Name == legacyName && ownerRef.Kind == "LatencySampler" {
						found = true
						break
					}
				}
				Expect(found).Should(BeFalse(), "Owner reference for %s should not be present", legacyName)
			}

			objectNames := []string{
				odigosv1.ActionMigratedLegacyPrefix + ActionName + "-legacy-latency-1",
				odigosv1.ActionMigratedLegacyPrefix + ActionName + "-legacy-latency-2",
			}
			for _, objectName := range objectNames {
				found := false
				for _, ownerRef := range ownerRefs {
					if ownerRef.Name == objectName && ownerRef.Kind == "Action" {
						found = true
						break
					}
				}
				Expect(found).Should(BeTrue(), "Owner reference for %s should be present", objectName)
			}
		})

		It("Should merge legacy ServiceNameSampler objects into a single processor", func() {
			testRules := []actionv1.ServiceNameFilter{
				{
					ServiceName:           "service-a",
					SamplingRatio:         10.0,
					FallbackSamplingRatio: 5.0,
				},
				{
					ServiceName:           "service-b",
					SamplingRatio:         15.0,
					FallbackSamplingRatio: 8.0,
				},
				{
					ServiceName:           "new-service",
					SamplingRatio:         20.0,
					FallbackSamplingRatio: 10.0,
				},
			}

			By("Creating multiple legacy ServiceNameSampler objects")
			legacyServiceSampler1 := &actionv1.ServiceNameSampler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-legacy-service-1",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.ServiceNameSamplerSpec{
					ActionName:          "legacy-service-sampler-1",
					Signals:             []common.ObservabilitySignal{common.TracesObservabilitySignal},
					ServicesNameFilters: []actionv1.ServiceNameFilter{testRules[0]},
				},
			}
			Expect(k8sClient.Create(testCtx, legacyServiceSampler1)).Should(Succeed())

			legacyServiceSampler2 := &actionv1.ServiceNameSampler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-legacy-service-2",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.ServiceNameSamplerSpec{
					ActionName:          "legacy-service-sampler-2",
					Signals:             []common.ObservabilitySignal{common.TracesObservabilitySignal},
					ServicesNameFilters: []actionv1.ServiceNameFilter{testRules[1]},
				},
			}
			Expect(k8sClient.Create(testCtx, legacyServiceSampler2)).Should(Succeed())

			By("Creating a new Action with ServiceNameSampler")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-merge-service-test",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-merge-service-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						ServiceNameSampler: &actionv1.ServiceNameSamplerConfig{
							ServicesNameFilters: []actionv1.ServiceNameFilter{testRules[2]},
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that a single odigossampling Processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "sampling-processor",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}
				ownerRefs := processor.GetOwnerReferences()
				return len(ownerRefs) == 3
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("odigossampling"))
			Expect(processor.Spec.OrderHint).Should(Equal(-24))

			actualSamplingConfig := sampling.SamplingConfig{}
			json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &actualSamplingConfig)
			Expect(len(actualSamplingConfig.ServiceRules)).Should(Equal(3), "Expected 3 rules in the processor config", actualSamplingConfig.ServiceRules)
			ruleDetailsList, err := getRuleDetailsFromRawJSON(processor.Spec.ProcessorConfig.Raw, "service", -1)
			Expect(err).ShouldNot(HaveOccurred())

			for _, rule := range testRules {
				found := false
				for _, ruleDetails := range ruleDetailsList {
					// Extract fields from rule_details
					fallbackRatio, ok1 := ruleDetails["fallback_sampling_ratio"].(float64)
					samplingRatio, ok2 := ruleDetails["sampling_ratio"].(float64)
					serviceName, ok3 := ruleDetails["service_name"].(string)

					if !ok1 || !ok2 || !ok3 {
						continue
					}

					if fallbackRatio == rule.FallbackSamplingRatio &&
						samplingRatio == rule.SamplingRatio &&
						serviceName == rule.ServiceName {
						found = true
						break
					}
				}
				Expect(found).Should(BeTrue(), "Rule with fallback sampling ratio %f, sampling ratio %f, service name %s should be present", rule.FallbackSamplingRatio, rule.SamplingRatio, rule.ServiceName)
			}

			By("Verifying that the processor does not have owner references to all legacy ServiceNameSampler objects")
			ownerRefs := processor.GetOwnerReferences()

			// Verify that the owner references do not include the legacy ServiceNameSampler objects
			legacyObjectNames := []string{
				ActionName + "-legacy-service-1",
				ActionName + "-legacy-service-2",
			}

			for _, legacyName := range legacyObjectNames {
				found := false
				for _, ownerRef := range ownerRefs {
					if ownerRef.Name == legacyName && ownerRef.Kind == "ServiceNameSampler" {
						found = true
						break
					}
				}
				Expect(found).Should(BeFalse(), "Owner reference for %s should not be present", legacyName)
			}

			objectNames := []string{
				odigosv1.ActionMigratedLegacyPrefix + ActionName + "-legacy-service-1",
				odigosv1.ActionMigratedLegacyPrefix + ActionName + "-legacy-service-2",
			}
			for _, objectName := range objectNames {
				found := false
				for _, ownerRef := range ownerRefs {
					if ownerRef.Name == objectName && ownerRef.Kind == "Action" {
						found = true
						break
					}
				}
				Expect(found).Should(BeTrue(), "Owner reference for %s should be present", objectName)
			}
		})

		It("Should merge legacy SpanAttributeSampler objects into a single processor", func() {
			testRules := []actionv1.SpanAttributeFilter{
				{
					ServiceName:           "service-x",
					AttributeKey:          "http.status_code",
					SamplingRatio:         10.0,
					FallbackSamplingRatio: 5.0,
					Condition: actionv1.AttributeCondition{
						StringCondition: &actionv1.StringAttributeCondition{
							Operation:     "equals",
							ExpectedValue: "500",
						},
					},
				},
				{
					ServiceName:           "service-y",
					AttributeKey:          "error",
					SamplingRatio:         15.0,
					FallbackSamplingRatio: 8.0,
					Condition: actionv1.AttributeCondition{
						StringCondition: &actionv1.StringAttributeCondition{
							Operation:     "exists",
							ExpectedValue: "",
						},
					},
				},
				{
					ServiceName:           "new-service",
					AttributeKey:          "custom.attribute",
					SamplingRatio:         20.0,
					FallbackSamplingRatio: 10.0,
					Condition: actionv1.AttributeCondition{
						StringCondition: &actionv1.StringAttributeCondition{
							Operation:     "contains",
							ExpectedValue: "test",
						},
					},
				},
			}
			By("Creating multiple legacy SpanAttributeSampler objects")
			legacySpanSampler1 := &actionv1.SpanAttributeSampler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-legacy-span-1",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.SpanAttributeSamplerSpec{
					ActionName:       "legacy-span-sampler-1",
					Signals:          []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AttributeFilters: []actionv1.SpanAttributeFilter{testRules[0]},
				},
			}
			Expect(k8sClient.Create(testCtx, legacySpanSampler1)).Should(Succeed())

			legacySpanSampler2 := &actionv1.SpanAttributeSampler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-legacy-span-2",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.SpanAttributeSamplerSpec{
					ActionName:       "legacy-span-sampler-2",
					Signals:          []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AttributeFilters: []actionv1.SpanAttributeFilter{testRules[1]},
				},
			}
			Expect(k8sClient.Create(testCtx, legacySpanSampler2)).Should(Succeed())

			By("Creating a new Action with SpanAttributeSampler")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-merge-span-test",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-merge-span-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						SpanAttributeSampler: &actionv1.SpanAttributeSamplerConfig{
							AttributeFilters: []actionv1.SpanAttributeFilter{testRules[2]},
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that a single odigossampling Processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "sampling-processor",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}
				ownerRefs := processor.GetOwnerReferences()
				return len(ownerRefs) == 3
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("odigossampling"))
			Expect(processor.Spec.OrderHint).Should(Equal(-24))

			By("Verifying that the processor does not have owner references to all legacy SpanAttributeSampler objects")
			ownerRefs := processor.GetOwnerReferences()

			// Verify that the owner references do not include the legacy SpanAttributeSampler objects
			legacyObjectNames := []string{
				ActionName + "-legacy-span-1",
				ActionName + "-legacy-span-2",
			}

			for _, legacyName := range legacyObjectNames {
				found := false
				for _, ownerRef := range ownerRefs {
					if ownerRef.Name == legacyName && ownerRef.Kind == "SpanAttributeSampler" {
						found = true
						break
					}
				}
				Expect(found).Should(BeFalse(), "Owner reference for %s should not be present", legacyName)
			}

			objectNames := []string{
				odigosv1.ActionMigratedLegacyPrefix + ActionName + "-legacy-span-1",
				odigosv1.ActionMigratedLegacyPrefix + ActionName + "-legacy-span-2",
			}
			for _, objectName := range objectNames {
				found := false
				for _, ownerRef := range ownerRefs {
					if ownerRef.Name == objectName && ownerRef.Kind == "Action" {
						found = true
						break
					}
				}
				Expect(found).Should(BeTrue(), "Owner reference for %s should be present", objectName)
			}

			actualSamplingConfig := sampling.SamplingConfig{}
			json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &actualSamplingConfig)
			Expect(len(actualSamplingConfig.ServiceRules)).Should(Equal(3), "Expected 3 rules in the processor config", actualSamplingConfig.ServiceRules)
			ruleDetailsList, err := getRuleDetailsFromRawJSON(processor.Spec.ProcessorConfig.Raw, "service", -1)
			Expect(err).ShouldNot(HaveOccurred())

			for _, rule := range testRules {
				found := false
				for _, ruleDetails := range ruleDetailsList {
					// Extract fields from rule_details
					fallbackRatio, ok1 := ruleDetails["fallback_sampling_ratio"].(float64)
					samplingRatio, ok2 := ruleDetails["sampling_ratio"].(float64)
					serviceName, ok3 := ruleDetails["service_name"].(string)
					attributeKey, ok4 := ruleDetails["attribute_key"].(string)
					expectedValue, ok5 := ruleDetails["expected_value"].(string)

					if !ok1 || !ok2 || !ok3 || !ok4 || (!ok5 && rule.Condition.StringCondition.ExpectedValue != "") {
						continue
					}

					if fallbackRatio == rule.FallbackSamplingRatio &&
						samplingRatio == rule.SamplingRatio &&
						serviceName == rule.ServiceName &&
						attributeKey == rule.AttributeKey &&
						expectedValue == rule.Condition.StringCondition.ExpectedValue {
						found = true
						break
					}
				}
				Expect(found).Should(BeTrue(), "Rule with fallback sampling ratio %f, sampling ratio %f, service name %s, attribute key %s should be present", rule.FallbackSamplingRatio, rule.SamplingRatio, rule.ServiceName, rule.AttributeKey)
			}
		})
	})
})
