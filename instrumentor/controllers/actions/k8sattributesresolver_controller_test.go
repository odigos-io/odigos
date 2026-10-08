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
	"github.com/odigos-io/odigos/common"
)

var _ = Describe("K8sAttributes Action", func() {
	const (
		ActionName      = "test-k8sattributes"
		ActionNamespace = "default"
	)

	AfterEach(func() {
		cleanupResources()
	})

	Context("When using FromSources for multi-source label extraction", func() {
		It("Should create a Processor with labels from multiple sources", func() {
			By("Creating an Action with labels from both pod and namespace")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-multi-source",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "multi-source-k8sattributes",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					K8sAttributes: &actionv1.K8sAttributesConfig{
						LabelsAttributes: []actionv1.K8sLabelAttribute{
							{
								LabelKey:     "environment",
								AttributeKey: "k8s.environment",
								FromSources: []actionv1.K8sAttributeSource{
									actionv1.PodAttributeSource,
									actionv1.NamespaceAttributeSource,
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that a Processor is created with labels from both sources")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "odigos-k8sattributes",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}

				// Parse the processor config
				var config map[string]interface{}
				err = json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &config)
				if err != nil {
					return false
				}

				// Check that label attributes include entries for both pod and namespace sources
				extract, ok := config["extract"].(map[string]interface{})
				if !ok {
					return false
				}

				labels, ok := extract["labels"].([]interface{})
				if !ok {
					return false
				}

				// We expect two entries: one for namespace, one for pod
				// Both should have the same tag_name and key, but different from values
				hasPodSource := false
				hasNamespaceSource := false
				for _, label := range labels {
					if labelMap, ok := label.(map[string]interface{}); ok {
						tagName, _ := labelMap["tag_name"].(string)
						key, _ := labelMap["key"].(string)
						from, _ := labelMap["from"].(string)

						if tagName == "k8s.environment" && key == "environment" {
							if from == "pod" {
								hasPodSource = true
							}
							if from == "namespace" {
								hasNamespaceSource = true
							}
						}
					}
				}

				return hasPodSource && hasNamespaceSource
			}, timeout, interval).Should(BeTrue())
		})

		It("Should order labels by precedence (namespace before pod)", func() {
			By("Creating an Action with labels from multiple sources")
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-precedence",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "precedence-k8sattributes",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					K8sAttributes: &actionv1.K8sAttributesConfig{
						LabelsAttributes: []actionv1.K8sLabelAttribute{
							{
								LabelKey:     "app.label",
								AttributeKey: "k8s.app.label",
								FromSources: []actionv1.K8sAttributeSource{
									actionv1.NamespaceAttributeSource,
									actionv1.PodAttributeSource,
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that labels are ordered by precedence (namespace before pod)")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "odigos-k8sattributes",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}

				// Parse the processor config
				var config map[string]interface{}
				err = json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &config)
				if err != nil {
					return false
				}

				extract, ok := config["extract"].(map[string]interface{})
				if !ok {
					return false
				}

				labels, ok := extract["labels"].([]interface{})
				if !ok || len(labels) < 2 {
					return false
				}

				// Find the indices of namespace and pod entries
				namespaceIdx := -1
				podIdx := -1
				for i, label := range labels {
					if labelMap, ok := label.(map[string]interface{}); ok {
						tagName, _ := labelMap["tag_name"].(string)
						from, _ := labelMap["from"].(string)
						if tagName == "k8s.app.label" {
							if from == "namespace" {
								namespaceIdx = i
							}
							if from == "pod" {
								podIdx = i
							}
						}
					}
				}

				// Namespace should come before pod (lower precedence processed first)
				return namespaceIdx != -1 && podIdx != -1 && namespaceIdx < podIdx
			}, timeout, interval).Should(BeTrue())
		})

		It("Should handle backward compatibility with From field", func() {
			By("Creating an Action with the deprecated From field")
			namespaceSource := actionv1.NamespaceAttributeSource
			action := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-compat",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "compat-k8sattributes",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					K8sAttributes: &actionv1.K8sAttributesConfig{
						LabelsAttributes: []actionv1.K8sLabelAttribute{
							{
								LabelKey:     "old.style.label",
								AttributeKey: "k8s.old.style.label",
								From:         &namespaceSource,
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

			By("Checking that the deprecated From field is handled correctly")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "odigos-k8sattributes",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}

				// Parse the processor config
				var config map[string]interface{}
				err = json.Unmarshal(processor.Spec.ProcessorConfig.Raw, &config)
				if err != nil {
					return false
				}

				extract, ok := config["extract"].(map[string]interface{})
				if !ok {
					return false
				}

				labels, ok := extract["labels"].([]interface{})
				if !ok {
					return false
				}

				// Check that the label is extracted from namespace
				for _, label := range labels {
					if labelMap, ok := label.(map[string]interface{}); ok {
						tagName, _ := labelMap["tag_name"].(string)
						from, _ := labelMap["from"].(string)
						if tagName == "k8s.old.style.label" && from == "namespace" {
							return true
						}
					}
				}

				return false
			}, timeout, interval).Should(BeTrue())
		})
	})
})
