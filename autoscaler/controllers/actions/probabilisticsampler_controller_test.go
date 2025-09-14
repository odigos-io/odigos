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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/actions"
	"github.com/odigos-io/odigos/common"
)

var _ = Describe("ProbabilisticSampler Controller", func() {
	const (
		ActionName      = "test-probabilistic-action"
		ActionNamespace = "default"
	)

	AfterEach(func() {
		cleanupResources()
	})

	Context("When creating ProbabilisticSampler Actions", func() {
		It("Should create a probabilistic_sampler Processor", func() {
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

			By("Checking that a probabilistic_sampler Processor is created")
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

		It("Should handle different sampling percentages", func() {
			By("Creating an Action with 25% sampling")
			action25 := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-25",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-25-percent-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						ProbabilisticSampler: &actionv1.ProbabilisticSamplerConfig{
							SamplingPercentage: "25",
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action25)).Should(Succeed())

			By("Creating an Action with 75% sampling")
			action75 := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-75",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-75-percent-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						ProbabilisticSampler: &actionv1.ProbabilisticSamplerConfig{
							SamplingPercentage: "75",
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action75)).Should(Succeed())

			By("Checking that both Processors are created")
			processor25 := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName + "-25",
					Namespace: ActionNamespace,
				}, processor25)
				if err != nil {
					return false
				}
				actualSamplingConfig := actions.ProbabilisticSamplerConfig{}
				json.Unmarshal(processor25.Spec.ProcessorConfig.Raw, &actualSamplingConfig)
				return actualSamplingConfig.Value == 25
			}, timeout, interval).Should(BeTrue())

			processor75 := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName + "-75",
					Namespace: ActionNamespace,
				}, processor75)
				if err != nil {
					return false
				}
				actualSamplingConfig := actions.ProbabilisticSamplerConfig{}
				json.Unmarshal(processor75.Spec.ProcessorConfig.Raw, &actualSamplingConfig)
				return actualSamplingConfig.Value == 75
			}, timeout, interval).Should(BeTrue())

			Expect(processor25.Spec.Type).Should(Equal("probabilistic_sampler"))
			Expect(processor75.Spec.Type).Should(Equal("probabilistic_sampler"))
		})

		It("Should handle legacy ProbabilisticSampler", func() {
			By("Creating a legacy ProbabilisticSampler")
			legacySampler := &actionv1.ProbabilisticSampler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-legacy",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.ProbabilisticSamplerSpec{
					ActionName:         "legacy-probabilistic-sampler",
					Signals:            []common.ObservabilitySignal{common.TracesObservabilitySignal},
					SamplingPercentage: "30",
				},
			}

			Expect(k8sClient.Create(testCtx, legacySampler)).Should(Succeed())

			By("Checking that a probabilistic_sampler Processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName + "-legacy",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}
				actualSamplingConfig := actions.ProbabilisticSamplerConfig{}
				json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &actualSamplingConfig)
				return actualSamplingConfig.Value == 30
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("probabilistic_sampler"))
			Expect(processor.Spec.OrderHint).Should(Equal(1))
		})
	})

	Context("When updating ProbabilisticSampler Actions", func() {
		It("Should update the corresponding Processor", func() {
			By("Creating a ProbabilisticSampler Action")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-update",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-update-probabilistic-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						ProbabilisticSampler: &actionv1.ProbabilisticSamplerConfig{
							SamplingPercentage: "40",
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Waiting for Processor to be created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName + "-update",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}
				actualSamplingConfig := actions.ProbabilisticSamplerConfig{}
				json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &actualSamplingConfig)
				return actualSamplingConfig.Value == 40
			}, timeout, interval).Should(BeTrue())

			By("Updating the Action")
			// Get the action again because it had its status updated after the first time
			Expect(k8sClient.Get(testCtx, types.NamespacedName{
				Name:      ActionName + "-update",
				Namespace: ActionNamespace,
			}, action)).Should(Succeed())
			action.Spec.Samplers.ProbabilisticSampler.SamplingPercentage = "60"
			Expect(k8sClient.Update(testCtx, action)).Should(Succeed())

			By("Checking that the Processor is updated")
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName + "-update",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}
				actualSamplingConfig := actions.ProbabilisticSamplerConfig{}
				json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &actualSamplingConfig)
				return actualSamplingConfig.Value == 60
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When handling invalid sampling percentages", func() {
		It("Should handle edge cases gracefully", func() {
			By("Creating an Action with 0% sampling")
			action0 := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-0",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-0-percent-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						ProbabilisticSampler: &actionv1.ProbabilisticSamplerConfig{
							SamplingPercentage: "0",
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action0)).Should(Succeed())

			By("Creating an Action with 100% sampling")
			action100 := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-100",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-100-percent-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						ProbabilisticSampler: &actionv1.ProbabilisticSamplerConfig{
							SamplingPercentage: "100",
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action100)).Should(Succeed())

			By("Checking that both Processors are created")
			processor0 := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName + "-0",
					Namespace: ActionNamespace,
				}, processor0)
				if err != nil {
					return false
				}
				actualSamplingConfig := actions.ProbabilisticSamplerConfig{}
				json.Unmarshal(processor0.Spec.ProcessorConfig.Raw, &actualSamplingConfig)
				return actualSamplingConfig.Value == 0
			}, timeout, interval).Should(BeTrue())

			processor100 := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName + "-100",
					Namespace: ActionNamespace,
				}, processor100)
				if err != nil {
					return false
				}
				actualSamplingConfig := actions.ProbabilisticSamplerConfig{}
				json.Unmarshal(processor100.Spec.ProcessorConfig.Raw, &actualSamplingConfig)
				return actualSamplingConfig.Value == 100
			}, timeout, interval).Should(BeTrue())

			Expect(processor0.Spec.Type).Should(Equal("probabilistic_sampler"))
			Expect(processor100.Spec.Type).Should(Equal("probabilistic_sampler"))
		})
	})

	Context("When creating ProbabilisticSampler Actions with legacy objects", func() {
		It("Should process all legacy ProbabilisticSampler objects into multiple probabilistic_sampler processors with owner references", func() {
			By("Creating multiple legacy ProbabilisticSampler objects")
			legacyProbSampler1 := &actionv1.ProbabilisticSampler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-legacy-prob-1",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.ProbabilisticSamplerSpec{
					ActionName:         "legacy-probabilistic-sampler-1",
					Signals:            []common.ObservabilitySignal{common.TracesObservabilitySignal},
					SamplingPercentage: "25",
				},
			}
			Expect(k8sClient.Create(testCtx, legacyProbSampler1)).Should(Succeed())

			legacyProbSampler2 := &actionv1.ProbabilisticSampler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-legacy-prob-2",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.ProbabilisticSamplerSpec{
					ActionName:         "legacy-probabilistic-sampler-2",
					Signals:            []common.ObservabilitySignal{common.TracesObservabilitySignal},
					SamplingPercentage: "35",
				},
			}
			Expect(k8sClient.Create(testCtx, legacyProbSampler2)).Should(Succeed())

			By("Creating a new Action with ProbabilisticSampler")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-merge-prob-test",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-merge-probabilistic-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						ProbabilisticSampler: &actionv1.ProbabilisticSamplerConfig{
							SamplingPercentage: "50",
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that multiple probabilistic_sampler Processors are created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName + "-legacy-prob-1",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}
				ownerRefs := processor.GetOwnerReferences()
				return len(ownerRefs) == 1
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("probabilistic_sampler"))
			Expect(processor.Spec.OrderHint).Should(Equal(1))

			By("Verifying that the processor does not have owner references to legacy ProbabilisticSampler objects")
			// The processor should not have owner references to all the legacy ProbabilisticSampler objects
			ownerRefs := processor.GetOwnerReferences()
			found := false
			for _, ownerRef := range ownerRefs {
				if ownerRef.Name == ActionName+"-legacy-prob-1" && ownerRef.Kind == "ProbabilisticSampler" {
					found = true
					break
				}
			}
			Expect(found).Should(BeFalse(), "Owner reference for legacy %s should not be present", ActionName+"-legacy-prob-1")

			found = false
			for _, ownerRef := range ownerRefs {
				if ownerRef.Name == odigosv1.ActionMigratedLegacyPrefix+ActionName+"-legacy-prob-1" && ownerRef.Kind == "Action" {
					found = true
					break
				}
			}
			Expect(found).Should(BeTrue(), "Owner reference for %s should be present", odigosv1.ActionMigratedLegacyPrefix+ActionName+"-legacy-prob-1")

			processor2 := &odigosv1.Processor{}

			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName + "-legacy-prob-2",
					Namespace: ActionNamespace,
				}, processor2)
				if err != nil {
					return false
				}
				ownerRefs := processor2.GetOwnerReferences()
				return len(ownerRefs) == 1
			}, timeout, interval).Should(BeTrue())

			Expect(processor2.Spec.Type).Should(Equal("probabilistic_sampler"))
			Expect(processor2.Spec.OrderHint).Should(Equal(1))

			By("Verifying that the processor does not have owner references to the legacy ProbabilisticSampler object")
			ownerRefs2 := processor2.GetOwnerReferences()
			found2 := false
			for _, ownerRef := range ownerRefs2 {
				if ownerRef.Name == odigosv1.ActionMigratedLegacyPrefix+ActionName+"-legacy-prob-2" && ownerRef.Kind == "ProbabilisticSampler" {
					found2 = true
					break
				}
			}
			Expect(found2).Should(BeFalse(), "Owner reference for %s should not be present", ActionName+"-legacy-prob-2")

			found2 = false
			for _, ownerRef := range ownerRefs2 {
				if ownerRef.Name == odigosv1.ActionMigratedLegacyPrefix+ActionName+"-legacy-prob-2" && ownerRef.Kind == "Action" {
					found2 = true
					break
				}
			}
			Expect(found2).Should(BeTrue(), "Owner reference for %s should be present", odigosv1.ActionMigratedLegacyPrefix+ActionName+"-legacy-prob-2")
		})

		It("Should handle mixed legacy and new ProbabilisticSampler configurations", func() {
			By("Creating a legacy ProbabilisticSampler")
			legacyProbSampler := &actionv1.ProbabilisticSampler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-legacy-mixed",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.ProbabilisticSamplerSpec{
					ActionName:         "legacy-mixed-sampler",
					Signals:            []common.ObservabilitySignal{common.TracesObservabilitySignal},
					SamplingPercentage: "30",
				},
			}
			Expect(k8sClient.Create(testCtx, legacyProbSampler)).Should(Succeed())

			By("Creating a new Action with ProbabilisticSampler")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-mixed-prob-test",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-mixed-probabilistic-sampler",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Samplers: &actionv1.SamplersConfig{
						ProbabilisticSampler: &actionv1.ProbabilisticSamplerConfig{
							SamplingPercentage: "60",
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that 2 probabilistic_sampler Processors are created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName + "-legacy-mixed",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}
				ownerRefs := processor.GetOwnerReferences()
				return len(ownerRefs) == 1
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("probabilistic_sampler"))
			Expect(processor.Spec.OrderHint).Should(Equal(1))

			By("Verifying that the processor has owner references to the legacy ProbabilisticSampler object")
			ownerRefs := processor.GetOwnerReferences()

			// Verify that the owner reference does not include the legacy ProbabilisticSampler object
			found := false
			for _, ownerRef := range ownerRefs {
				if ownerRef.Name == ActionName+"-legacy-mixed" && ownerRef.Kind == "ProbabilisticSampler" {
					found = true
					break
				}
			}
			Expect(found).Should(BeFalse(), "Owner reference for legacy ProbabilisticSampler should not be present")

			found = false
			for _, ownerRef := range ownerRefs {
				if ownerRef.Name == odigosv1.ActionMigratedLegacyPrefix+ActionName+"-legacy-mixed" && ownerRef.Kind == "Action" {
					found = true
					break
				}
			}
			Expect(found).Should(BeTrue(), "Owner reference for %s should be present", odigosv1.ActionMigratedLegacyPrefix+ActionName+"-legacy-mixed")

			processor2 := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName + "-mixed-prob-test",
					Namespace: ActionNamespace,
				}, processor2)
				if err != nil {
					return false
				}
				ownerRefs := processor2.GetOwnerReferences()
				return len(ownerRefs) == 1
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("probabilistic_sampler"))
			Expect(processor.Spec.OrderHint).Should(Equal(1))

			By("Verifying that the processor does not have owner references to the legacy ProbabilisticSampler object")
			ownerRefs2 := processor2.GetOwnerReferences()

			// Verify that the owner reference includes the legacy ProbabilisticSampler object
			found2 := false
			for _, ownerRef := range ownerRefs2 {
				if ownerRef.Name == ActionName+"-mixed-prob-test" && ownerRef.Kind == "Action" {
					found2 = true
					break
				}
			}
			Expect(found2).Should(BeTrue(), "Owner reference for legacy ProbabilisticSampler should be present")
		})
	})
})
