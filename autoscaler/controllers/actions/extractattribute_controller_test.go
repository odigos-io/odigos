/*
Copyright 2026.

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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
)

// extractAttributeRawConfig mirrors the snake_case shape the autoscaler
// renders into Processor.Spec.ProcessorConfig.Raw. We re-declare it here
// (instead of importing the unexported extractAttributeProcessorConfig from
// the actions package) so the test asserts on the wire format the OTel
// collector actually consumes via mapstructure.
type extractAttributeRawConfig struct {
	Extractions []extractAttributeRawRule `json:"extractions"`
}

type extractAttributeRawRule struct {
	TargetAttributeName string `json:"target_attribute_name"`
	LookupKey        string `json:"lookup_key,omitempty"`
	DataFormat       string `json:"data_format,omitempty"`
	Regex            string `json:"regex,omitempty"`
}

var _ = Describe("ExtractAttribute Controller", func() {
	const (
		ActionName      = "test-extractattribute-action"
		ActionNamespace = "default"
	)

	AfterEach(func() {
		cleanupResources()
	})

	Context("When creating an Action with ExtractAttribute", func() {
		It("Should create a Processor with the correct type and snake_case config (lookupKey+dataFormat)", func() {
			By("Creating an Action with ExtractAttribute using lookupKey+dataFormat")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName,
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "extract study.id",
					Notes:      "Extract study.id from db.statement and http.request.payload",
					Disabled:   false,
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					ExtractAttribute: &odigosactions.ExtractAttributeConfig{
						Extractions: []odigosactions.Extraction{
							{
								TargetAttributeName: "study.id",
								LookupKey:        "study_id",
								DataFormat:       odigosactions.FormatJSON,
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
					Name:      ActionName,
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking that the Processor has the correct type and metadata")
			Expect(processor.Spec.Type).Should(Equal("odigosextractattribute"))
			Expect(processor.Spec.ProcessorName).Should(Equal("extract study.id"))
			Expect(processor.Spec.OrderHint).Should(Equal(2))
			Expect(processor.Spec.Disabled).Should(BeFalse())
			Expect(processor.Spec.Notes).Should(Equal("Extract study.id from db.statement and http.request.payload"))
			Expect(processor.Spec.Signals).Should(ContainElement(common.TracesObservabilitySignal))
			Expect(processor.Spec.CollectorRoles).Should(ContainElement(odigosv1.CollectorsGroupRoleClusterGateway))

			By("Checking that the rendered ProcessorConfig uses snake_case keys")
			var rendered extractAttributeRawConfig
			Expect(json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &rendered)).Should(Succeed())
			Expect(rendered.Extractions).Should(HaveLen(1))
			Expect(rendered.Extractions[0].TargetAttributeName).Should(Equal("study.id"))
			Expect(rendered.Extractions[0].LookupKey).Should(Equal("study_id"))
			Expect(rendered.Extractions[0].DataFormat).Should(Equal("json"))
			Expect(rendered.Extractions[0].Regex).Should(BeEmpty())

			By("Verifying that the Action owns the Processor")
			ownerRefs := processor.GetOwnerReferences()
			Expect(ownerRefs).Should(HaveLen(1))
			Expect(ownerRefs[0].Name).Should(Equal(ActionName))
			Expect(ownerRefs[0].Kind).Should(Equal("Action"))
		})

		It("Should create a Processor when using a custom regex", func() {
			By("Creating an Action with ExtractAttribute using regex")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-regex",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "extract via regex",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					ExtractAttribute: &odigosactions.ExtractAttributeConfig{
						Extractions: []odigosactions.Extraction{
							{
								TargetAttributeName: "request.id",
								Regex:            `request_id=([A-Za-z0-9-]+)`,
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
					Name:      ActionName + "-regex",
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking that regex-only extractions omit lookup_key and data_format")
			var rendered extractAttributeRawConfig
			Expect(json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &rendered)).Should(Succeed())
			Expect(rendered.Extractions).Should(HaveLen(1))
			Expect(rendered.Extractions[0].TargetAttributeName).Should(Equal("request.id"))
			Expect(rendered.Extractions[0].Regex).Should(Equal(`request_id=([A-Za-z0-9-]+)`))
			Expect(rendered.Extractions[0].LookupKey).Should(BeEmpty())
			Expect(rendered.Extractions[0].DataFormat).Should(BeEmpty())
		})

		It("Should preserve the order of multiple extractions", func() {
			By("Creating an Action with multiple extractions")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-multi",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "extract multiple",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					ExtractAttribute: &odigosactions.ExtractAttributeConfig{
						Extractions: []odigosactions.Extraction{
							{
								TargetAttributeName: "extracted_study.id",
								LookupKey:        "studies",
								DataFormat:       odigosactions.FormatResourcePath,
							},
							{
								TargetAttributeName: "extracted_project.id",
								LookupKey:        "projects",
								DataFormat:       odigosactions.FormatResourcePath,
							},
							{
								TargetAttributeName: "trace.id",
								Regex:            `traceId=([0-9a-f]+)`,
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName + "-multi",
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			var rendered extractAttributeRawConfig
			Expect(json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &rendered)).Should(Succeed())
			Expect(rendered.Extractions).Should(HaveLen(3))

			Expect(rendered.Extractions[0].TargetAttributeName).Should(Equal("extracted_study.id"))
			Expect(rendered.Extractions[0].LookupKey).Should(Equal("studies"))
			Expect(rendered.Extractions[0].DataFormat).Should(Equal("resource_path"))

			Expect(rendered.Extractions[1].TargetAttributeName).Should(Equal("extracted_project.id"))
			Expect(rendered.Extractions[1].LookupKey).Should(Equal("projects"))
			Expect(rendered.Extractions[1].DataFormat).Should(Equal("resource_path"))

			Expect(rendered.Extractions[2].TargetAttributeName).Should(Equal("trace.id"))
			Expect(rendered.Extractions[2].Regex).Should(Equal(`traceId=([0-9a-f]+)`))
			Expect(rendered.Extractions[2].LookupKey).Should(BeEmpty())
			Expect(rendered.Extractions[2].DataFormat).Should(BeEmpty())
		})

		It("Should reject logs and metrics signals", func() {
			By("Creating an Action with unsupported signals")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-unsupported-signals",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "extract unsupported",
					Signals: []common.ObservabilitySignal{
						common.TracesObservabilitySignal,
						common.LogsObservabilitySignal,
						common.MetricsObservabilitySignal,
					},
					ExtractAttribute: &odigosactions.ExtractAttributeConfig{
						Extractions: []odigosactions.Extraction{
							{
								TargetAttributeName: "user.id",
								LookupKey:        "user_id",
								DataFormat:       odigosactions.FormatJSON,
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that no Processor is created")
			processor := &odigosv1.Processor{}
			Consistently(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName + "-unsupported-signals",
					Namespace: ActionNamespace,
				}, processor)
				return err != nil
			}, timeout, interval).Should(BeTrue())

			By("Checking that the Action status reports the unsupported signal")
			Eventually(func() string {
				updatedAction := &odigosv1.Action{}
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName + "-unsupported-signals",
					Namespace: ActionNamespace,
				}, updatedAction)
				if err != nil {
					return ""
				}
				condition := meta.FindStatusCondition(updatedAction.Status.Conditions, odigosv1.ActionTransformedToProcessorType)
				if condition == nil {
					return ""
				}
				return condition.Message
			}, timeout, interval).Should(ContainSubstring("unsupported signal in ExtractAttribute action: LOGS"))
		})
	})

	Context("When the ExtractAttribute config is invalid", func() {
		It("Should not create a Processor when lookupKey and regex are both set", func() {
			By("Creating an Action with both lookupKey and regex set on an extraction")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-invalid-both",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "invalid - both lookupKey and regex",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					ExtractAttribute: &odigosactions.ExtractAttributeConfig{
						Extractions: []odigosactions.Extraction{
							{
								TargetAttributeName: "x",
								LookupKey:        "user_id",
								DataFormat:       odigosactions.FormatJSON,
								Regex:            `user_id=(\d+)`,
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that no Processor is created")
			processor := &odigosv1.Processor{}
			Consistently(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName + "-invalid-both",
					Namespace: ActionNamespace,
				}, processor)
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})

		It("Should not create a Processor when lookupKey is set without dataFormat", func() {
			By("Creating an Action with lookupKey but no dataFormat")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-invalid-format",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "invalid - missing dataFormat",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					ExtractAttribute: &odigosactions.ExtractAttributeConfig{
						Extractions: []odigosactions.Extraction{
							{
								TargetAttributeName: "x",
								LookupKey:        "user_id",
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that no Processor is created")
			processor := &odigosv1.Processor{}
			Consistently(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      ActionName + "-invalid-format",
					Namespace: ActionNamespace,
				}, processor)
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
