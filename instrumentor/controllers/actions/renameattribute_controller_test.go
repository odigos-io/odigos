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

var _ = Describe("RenameAttribute Controller", func() {
	const (
		ActionName      = "test-renameattribute-action"
		ActionNamespace = "default"
	)

	AfterEach(func() {
		cleanupResources()
	})

	Context("When creating legacy RenameAttribute Actions", func() {
		It("Should migrate legacy RenameAttribute to new Action and create processor", func() {
			By("Creating a legacy RenameAttribute action")
			legacyAction := &actionv1.RenameAttribute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName,
					Namespace: ActionNamespace,
				},
				Spec: actionv1.RenameAttributeSpec{
					ActionName: "test-renameattribute",
					Notes:      "Test rename attribute notes",
					Disabled:   false,
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal, common.LogsObservabilitySignal},
					Renames: map[string]string{
						"old.attribute.name": "new.attribute.name",
						"service.name":       "service.name.new",
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

			Expect(migratedAction.Spec.ActionName).Should(Equal("test-renameattribute"))
			Expect(migratedAction.Spec.Notes).Should(Equal("Test rename attribute notes"))
			Expect(migratedAction.Spec.Disabled).Should(BeFalse())
			Expect(migratedAction.Spec.Signals).Should(ContainElements(common.TracesObservabilitySignal, common.LogsObservabilitySignal))
			Expect(migratedAction.Spec.RenameAttribute).ShouldNot(BeNil())
			Expect(migratedAction.Spec.RenameAttribute.Renames).Should(HaveLen(2))
			Expect(migratedAction.Spec.RenameAttribute.Renames["old.attribute.name"]).Should(Equal("new.attribute.name"))
			Expect(migratedAction.Spec.RenameAttribute.Renames["service.name"]).Should(Equal("service.name.new"))

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
			Expect(processor.Spec.ProcessorName).Should(Equal("test-renameattribute"))
			Expect(processor.Spec.OrderHint).Should(Equal(-50))
			Expect(processor.Spec.Signals).Should(ContainElements(common.TracesObservabilitySignal, common.LogsObservabilitySignal))

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
			By("Creating a legacy RenameAttribute action")
			legacyAction := &actionv1.RenameAttribute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName,
					Namespace: ActionNamespace,
				},
				Spec: actionv1.RenameAttributeSpec{
					ActionName: "test-renameattribute",
					Notes:      "Initial notes",
					Disabled:   false,
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Renames: map[string]string{
						"old.attribute": "new.attribute",
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
			legacyAction.Spec.Notes = "Updated rename attribute notes"
			legacyAction.Spec.Renames["additional.attribute"] = "renamed.attribute"
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
					len(migratedAction.Spec.RenameAttribute.Renames) == 1 &&
					migratedAction.Spec.Disabled == false
			}, timeout, interval).Should(BeTrue())
		})

		It("Should handle multiple signals correctly", func() {
			By("Creating a legacy RenameAttribute action with multiple signals")
			legacyAction := &actionv1.RenameAttribute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-multi-signal",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.RenameAttributeSpec{
					ActionName: "test-renameattribute-multi",
					Notes:      "Test with multiple signals",
					Disabled:   false,
					Signals: []common.ObservabilitySignal{
						common.TracesObservabilitySignal,
						common.LogsObservabilitySignal,
						common.MetricsObservabilitySignal,
					},
					Renames: map[string]string{
						"trace.attribute": "span.attribute",
						"log.attribute":   "log.new.attribute",
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
	})
})
