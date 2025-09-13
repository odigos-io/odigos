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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

var _ = Describe("PiiMasking Controller", func() {
	const (
		ActionName      = "test-piimasking-action"
		ActionNamespace = "default"
	)

	AfterEach(func() {
		cleanupResources()
	})

	Context("When creating legacy PiiMasking Actions", func() {
		It("Should migrate legacy PiiMasking to new Action and create processor", func() {
			By("Creating a legacy PiiMasking action")
			legacyAction := &actionv1.PiiMasking{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName,
					Namespace: ActionNamespace,
				},
				Spec: actionv1.PiiMaskingSpec{
					ActionName: "test-piimasking",
					Notes:      "Test PII masking notes",
					Disabled:   false,
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					PiiCategories: []actionv1.PiiCategory{
						actionv1.CreditCardMasking,
					},
				},
			}

			Expect(k8sClient.Create(testCtx, legacyAction)).Should(Succeed())

			By("Checking that a migrated Action is created")
			migratedAction := &odigosv1.Action{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName,
					Namespace: ActionNamespace,
				}, migratedAction)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(migratedAction.Spec.ActionName).Should(Equal("test-piimasking"))
			Expect(migratedAction.Spec.Notes).Should(Equal("Test PII masking notes"))
			Expect(migratedAction.Spec.Disabled).Should(BeFalse())
			Expect(migratedAction.Spec.Signals).Should(ContainElements(common.TracesObservabilitySignal))
			Expect(migratedAction.Spec.PiiMasking).ShouldNot(BeNil())
			Expect(migratedAction.Spec.PiiMasking.PiiCategories).Should(ContainElements(actionv1.CreditCardMasking))

			By("Checking that a redaction processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName,
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("redaction"))
			Expect(processor.Spec.ProcessorName).Should(Equal("test-piimasking"))
			Expect(processor.Spec.OrderHint).Should(Equal(1))
			Expect(processor.Spec.Signals).Should(ContainElements(common.TracesObservabilitySignal))

			By("Verifying owner references")
			ownerRefs := migratedAction.GetOwnerReferences()
			Expect(ownerRefs).Should(HaveLen(0))
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName,
					Namespace: ActionNamespace,
				}, legacyAction)
				Expect(err).Should(BeNil())
				return len(legacyAction.GetOwnerReferences()) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(legacyAction.GetOwnerReferences()[0].Name).Should(Equal(odigosv1.ActionMigratedLegacyPrefix + ActionName))
			Expect(legacyAction.GetOwnerReferences()[0].Kind).Should(Equal("Action"))
		})

		It("Should not update existing migrated Action when legacy action changes", func() {
			By("Creating a legacy PiiMasking action")
			legacyAction := &actionv1.PiiMasking{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName,
					Namespace: ActionNamespace,
				},
				Spec: actionv1.PiiMaskingSpec{
					ActionName: "test-piimasking",
					Notes:      "Initial notes",
					Disabled:   false,
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					PiiCategories: []actionv1.PiiCategory{
						actionv1.CreditCardMasking,
					},
				},
			}

			Expect(k8sClient.Create(testCtx, legacyAction)).Should(Succeed())

			By("Waiting for initial migration")
			migratedAction := &odigosv1.Action{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName,
					Namespace: ActionNamespace,
				}, migratedAction)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Updating the legacy action")
			legacyAction.Spec.Notes = "Updated PII masking notes"
			legacyAction.Spec.Disabled = true

			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName,
					Namespace: ActionNamespace,
				}, legacyAction)
				Expect(err).Should(BeNil())
				return len(legacyAction.GetOwnerReferences()) == 1
			}, timeout, interval).Should(BeTrue())
			Expect(k8sClient.Update(testCtx, legacyAction)).Should(Succeed())

			By("Checking that the migrated Action is not updated")
			Consistently(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName,
					Namespace: ActionNamespace,
				}, migratedAction)
				if err != nil {
					return false
				}
				return migratedAction.Spec.Notes == "Initial notes" &&
					migratedAction.Spec.Disabled == false
			}, timeout, interval).Should(BeTrue())
		})

		It("Should handle unsupported signals gracefully", func() {
			By("Creating a legacy PiiMasking action with unsupported signals")
			legacyAction := &actionv1.PiiMasking{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-unsupported",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.PiiMaskingSpec{
					ActionName: "test-piimasking-unsupported",
					Notes:      "Test with unsupported signals",
					Disabled:   false,
					Signals:    []common.ObservabilitySignal{common.LogsObservabilitySignal}, // Unsupported for PII masking
					PiiCategories: []actionv1.PiiCategory{
						actionv1.CreditCardMasking,
					},
				},
			}

			Expect(k8sClient.Create(testCtx, legacyAction)).Should(Succeed())

			By("Checking that the action is still migrated despite unsupported signals")
			migratedAction := &odigosv1.Action{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName + "-unsupported",
					Namespace: ActionNamespace,
				}, migratedAction)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(migratedAction.Spec.PiiMasking).ShouldNot(BeNil())
			Expect(migratedAction.Spec.PiiMasking.PiiCategories).Should(ContainElements(actionv1.CreditCardMasking))
			Expect(migratedAction.Spec.Notes).Should(Equal("Test with unsupported signals"))
			Expect(migratedAction.Spec.Signals).Should(ContainElements(common.LogsObservabilitySignal))
		})
	})
})
