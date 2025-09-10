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

var _ = Describe("AddClusterInfo Controller", func() {
	const (
		ActionName      = "test-addclusterinfo-action"
		ActionNamespace = "default"
	)

	AfterEach(func() {
		cleanupResources()
	})

	Context("When creating legacy AddClusterInfo Actions", func() {
		It("Should migrate legacy AddClusterInfo to new Action and create processor", func() {
			By("Creating a legacy AddClusterInfo action")
			legacyAction := &actionv1.AddClusterInfo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName,
					Namespace: ActionNamespace,
				},
				Spec: actionv1.AddClusterInfoSpec{
					ActionName: "test-addclusterinfo",
					Notes:      "Test notes",
					Disabled:   false,
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal, common.LogsObservabilitySignal},
					ClusterAttributes: []actionv1.OtelAttributeWithValue{
						{
							AttributeName:        "cluster.name",
							AttributeStringValue: stringPtr("test-cluster"),
						},
						{
							AttributeName:        "cluster.region",
							AttributeStringValue: stringPtr("us-west-2"),
						},
					},
					OverwriteExistingValues: true,
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

			Expect(migratedAction.Spec.ActionName).Should(Equal("test-addclusterinfo"))
			Expect(migratedAction.Spec.Notes).Should(Equal("Test notes"))
			Expect(migratedAction.Spec.Disabled).Should(BeFalse())
			Expect(migratedAction.Spec.Signals).Should(ContainElements(common.TracesObservabilitySignal, common.LogsObservabilitySignal))
			Expect(migratedAction.Spec.AddClusterInfo).ShouldNot(BeNil())
			Expect(migratedAction.Spec.AddClusterInfo.ClusterAttributes).Should(HaveLen(2))
			Expect(migratedAction.Spec.AddClusterInfo.OverwriteExistingValues).Should(BeTrue())

			By("Checking that a resource processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      odigosv1.ActionMigratedLegacyPrefix + ActionName,
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("resource"))
			Expect(processor.Spec.ProcessorName).Should(Equal("test-addclusterinfo"))
			Expect(processor.Spec.OrderHint).Should(Equal(1))
			Expect(processor.Spec.Signals).Should(ContainElements(common.TracesObservabilitySignal, common.LogsObservabilitySignal))

			By("Verifying owner references")
			ownerRefs := migratedAction.GetOwnerReferences()
			Expect(ownerRefs).Should(HaveLen(1))
			Expect(ownerRefs[0].Name).Should(Equal(ActionName))
			Expect(ownerRefs[0].Kind).Should(Equal("AddClusterInfo"))
		})

		It("Should not update existing migrated Action when legacy action changes", func() {
			By("Creating a legacy AddClusterInfo action")
			legacyAction := &actionv1.AddClusterInfo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName,
					Namespace: ActionNamespace,
				},
				Spec: actionv1.AddClusterInfoSpec{
					ActionName: "test-addclusterinfo",
					Notes:      "Initial notes",
					Disabled:   false,
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					ClusterAttributes: []actionv1.OtelAttributeWithValue{
						{
							AttributeName:        "cluster.name",
							AttributeStringValue: stringPtr("test-cluster"),
						},
					},
					OverwriteExistingValues: false,
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
			legacyAction.Spec.Notes = "Updated notes"
			legacyAction.Spec.ClusterAttributes = append(legacyAction.Spec.ClusterAttributes, actionv1.OtelAttributeWithValue{
				AttributeName:        "cluster.version",
				AttributeStringValue: stringPtr("v1.0.0"),
			})
			legacyAction.Spec.OverwriteExistingValues = true

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
					len(migratedAction.Spec.AddClusterInfo.ClusterAttributes) == 1 &&
					migratedAction.Spec.AddClusterInfo.OverwriteExistingValues == false
			}, timeout, interval).Should(BeTrue())
		})
	})
})

func stringPtr(s string) *string {
	return &s
}
