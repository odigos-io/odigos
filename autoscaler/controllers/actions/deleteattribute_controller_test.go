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

var _ = Describe("DeleteAttribute Controller", func() {
	const (
		ActionName      = "test-deleteattribute-action"
		ActionNamespace = "default"
	)

	AfterEach(func() {
		cleanupResources()
	})

	Context("When creating legacy DeleteAttribute Actions", func() {
		It("Should migrate legacy DeleteAttribute to new Action and create processor", func() {
			By("Creating a legacy DeleteAttribute action")
			legacyAction := &actionv1.DeleteAttribute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName,
					Namespace: ActionNamespace,
				},
				Spec: actionv1.DeleteAttributeSpec{
					ActionName: "test-deleteattribute",
					Notes:      "Test delete attribute notes",
					Disabled:   false,
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal, common.LogsObservabilitySignal},
					AttributeNamesToDelete: []string{
						"sensitive.data",
						"internal.id",
						"debug.info",
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

			Expect(migratedAction.Spec.ActionName).Should(Equal("test-deleteattribute"))
			Expect(migratedAction.Spec.Notes).Should(Equal("Test delete attribute notes"))
			Expect(migratedAction.Spec.Disabled).Should(BeFalse())
			Expect(migratedAction.Spec.Signals).Should(ContainElements(common.TracesObservabilitySignal, common.LogsObservabilitySignal))
			Expect(migratedAction.Spec.DeleteAttribute).ShouldNot(BeNil())
			Expect(migratedAction.Spec.DeleteAttribute.AttributeNamesToDelete).Should(HaveLen(3))
			Expect(migratedAction.Spec.DeleteAttribute.AttributeNamesToDelete).Should(ContainElements("sensitive.data", "internal.id", "debug.info"))

			By("Checking that a transform processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName,
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("transform"))
			Expect(processor.Spec.ProcessorName).Should(Equal("test-deleteattribute"))
			Expect(processor.Spec.OrderHint).Should(Equal(-100))
			Expect(processor.Spec.Signals).Should(ContainElements(common.TracesObservabilitySignal, common.LogsObservabilitySignal))

			By("Verifying owner references")
			ownerRefs := migratedAction.GetOwnerReferences()
			Expect(ownerRefs).Should(HaveLen(1))
			Expect(ownerRefs[0].Name).Should(Equal(ActionName))
			Expect(ownerRefs[0].Kind).Should(Equal("DeleteAttribute"))
		})

		It("Should not update existing migrated Action when legacy action changes", func() {
			By("Creating a legacy DeleteAttribute action")
			legacyAction := &actionv1.DeleteAttribute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName,
					Namespace: ActionNamespace,
				},
				Spec: actionv1.DeleteAttributeSpec{
					ActionName: "test-deleteattribute",
					Notes:      "Initial notes",
					Disabled:   false,
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AttributeNamesToDelete: []string{
						"sensitive.data",
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
			legacyAction.Spec.Notes = "Updated delete attribute notes"
			legacyAction.Spec.AttributeNamesToDelete = append(legacyAction.Spec.AttributeNamesToDelete, "additional.sensitive.data")
			legacyAction.Spec.Disabled = true

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
					len(migratedAction.Spec.DeleteAttribute.AttributeNamesToDelete) == 1 &&
					migratedAction.Spec.Disabled == false
			}, timeout, interval).Should(BeTrue())
		})

		It("Should handle multiple signals correctly", func() {
			By("Creating a legacy DeleteAttribute action with multiple signals")
			legacyAction := &actionv1.DeleteAttribute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-multi-signal",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.DeleteAttributeSpec{
					ActionName: "test-deleteattribute-multi",
					Notes:      "Test with multiple signals",
					Disabled:   false,
					Signals: []common.ObservabilitySignal{
						common.TracesObservabilitySignal,
						common.LogsObservabilitySignal,
						common.MetricsObservabilitySignal,
					},
					AttributeNamesToDelete: []string{
						"trace.sensitive",
						"log.sensitive",
						"metric.sensitive",
					},
				},
			}

			Expect(k8sClient.Create(testCtx, legacyAction)).Should(Succeed())

			By("Checking that the migrated Action handles multiple signals")
			migratedAction := &odigosv1.Action{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName + "-multi-signal",
					Namespace: ActionNamespace,
				}, migratedAction)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(migratedAction.Spec.Signals).Should(HaveLen(3))
			Expect(migratedAction.Spec.Signals).Should(ContainElements(
				common.TracesObservabilitySignal,
				common.LogsObservabilitySignal,
				common.MetricsObservabilitySignal,
			))

			By("Checking that the processor is created with correct signals")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName + "-multi-signal",
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Signals).Should(HaveLen(3))
			Expect(processor.Spec.Signals).Should(ContainElements(
				common.TracesObservabilitySignal,
				common.LogsObservabilitySignal,
				common.MetricsObservabilitySignal,
			))
		})

		It("Should handle empty attribute list gracefully", func() {
			By("Creating a legacy DeleteAttribute action with no attributes to delete")
			legacyAction := &actionv1.DeleteAttribute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-empty",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.DeleteAttributeSpec{
					ActionName:             "test-deleteattribute-empty",
					Notes:                  "Test with empty attribute list",
					Disabled:               false,
					Signals:                []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AttributeNamesToDelete: []string{},
				},
			}

			Expect(k8sClient.Create(testCtx, legacyAction)).Should(Succeed())

			By("Checking that the action is still migrated despite empty attribute list")
			migratedAction := &odigosv1.Action{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName + "-empty",
					Namespace: ActionNamespace,
				}, migratedAction)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(migratedAction.Spec.DeleteAttribute).ShouldNot(BeNil())
			Expect(migratedAction.Spec.DeleteAttribute.AttributeNamesToDelete).Should(HaveLen(0))
			Expect(migratedAction.Spec.Notes).Should(Equal("Test with empty attribute list"))
			Expect(migratedAction.Spec.Signals).Should(ContainElements(common.TracesObservabilitySignal))
		})

		It("Should handle disabled action correctly", func() {
			By("Creating a disabled legacy DeleteAttribute action")
			legacyAction := &actionv1.DeleteAttribute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-disabled",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.DeleteAttributeSpec{
					ActionName: "test-deleteattribute-disabled",
					Notes:      "Test disabled action",
					Disabled:   true,
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AttributeNamesToDelete: []string{
						"sensitive.data",
					},
				},
			}

			Expect(k8sClient.Create(testCtx, legacyAction)).Should(Succeed())

			By("Checking that the disabled action is migrated correctly")
			migratedAction := &odigosv1.Action{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName + "-disabled",
					Namespace: ActionNamespace,
				}, migratedAction)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(migratedAction.Spec.Disabled).Should(BeTrue())
			Expect(migratedAction.Spec.Notes).Should(Equal("Test disabled action"))
			Expect(migratedAction.Spec.DeleteAttribute.AttributeNamesToDelete).Should(ContainElements("sensitive.data"))
		})
	})
})
