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

var _ = Describe("K8sAttributesResolver Controller", func() {
	const (
		ActionName      = "test-k8sattributes"
		ActionNamespace = "default"
	)

	AfterEach(func() {
		cleanupResources()
	})

	Context("When creating a K8sAttributesResolver", func() {
		It("Should create a Processor with basic configuration", func() {
			By("Creating a K8sAttributesResolver with basic settings")
			resolver := &actionv1.K8sAttributesResolver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-basic",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.K8sAttributesSpec{
					ActionName: "test-basic-k8sattributes",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
				},
			}

			Expect(k8sClient.Create(testCtx, resolver)).Should(Succeed())

			By("Checking that a Processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "odigos-k8sattributes",
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(processor.Spec.Type).Should(Equal("k8sattributes"))
			Expect(processor.Spec.ProcessorName).Should(Equal("Unified Kubernetes Attributes"))
			Expect(processor.Spec.OrderHint).Should(Equal(0))
			Expect(processor.Spec.Disabled).Should(BeFalse())
			Expect(processor.Spec.CollectorRoles).Should(ContainElement(odigosv1.CollectorsGroupRoleNodeCollector))

			ownerRefs := processor.GetOwnerReferences()
			Expect(len(ownerRefs)).Should(Equal(1))
			Expect(ownerRefs[0].Name).Should(Equal(odigosv1.ActionMigratedLegacyPrefix + ActionName + "-basic"))
			Expect(ownerRefs[0].Kind).Should(Equal("Action"))
		})

		It("Should create a Processor with container attributes enabled", func() {
			By("Creating a K8sAttributesResolver with container attributes")
			resolver := &actionv1.K8sAttributesResolver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-container",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.K8sAttributesSpec{
					ActionName:                 "test-container-k8sattributes",
					Signals:                    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					CollectContainerAttributes: true,
				},
			}

			Expect(k8sClient.Create(testCtx, resolver)).Should(Succeed())

			By("Checking that a Processor is created with container attributes")
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

				// Check that container attributes are included
				extract, ok := config["extract"].(map[string]interface{})
				if !ok {
					return false
				}

				metadata, ok := extract["metadata"].([]interface{})
				if !ok {
					return false
				}

				// Check for container-related attributes
				hasContainerName := false
				hasContainerID := false
				hasContainerImageName := false
				hasContainerImageTag := false

				for _, attr := range metadata {
					if attrStr, ok := attr.(string); ok {
						switch attrStr {
						case "k8s.container.name":
							hasContainerName = true
						case "container.id":
							hasContainerID = true
						case "container.image.name":
							hasContainerImageName = true
						case "container.image.tag":
							hasContainerImageTag = true
						}
					}
				}

				return hasContainerName && hasContainerID && hasContainerImageName && hasContainerImageTag
			}, timeout, interval).Should(BeTrue())
		})

		It("Should create a Processor with cluster UID attributes enabled", func() {
			By("Creating a K8sAttributesResolver with cluster UID")
			resolver := &actionv1.K8sAttributesResolver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-cluster",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.K8sAttributesSpec{
					ActionName:        "test-cluster-k8sattributes",
					Signals:           []common.ObservabilitySignal{common.TracesObservabilitySignal},
					CollectClusterUID: true,
				},
			}

			Expect(k8sClient.Create(testCtx, resolver)).Should(Succeed())

			By("Checking that a Processor is created with cluster UID attributes")
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

				// Check that cluster UID attributes are included
				extract, ok := config["extract"].(map[string]interface{})
				if !ok {
					return false
				}

				metadata, ok := extract["metadata"].([]interface{})
				if !ok {
					return false
				}

				// Check for cluster UID attribute
				hasClusterUID := false
				for _, attr := range metadata {
					if attrStr, ok := attr.(string); ok {
						if attrStr == "k8s.cluster.uid" {
							hasClusterUID = true
							break
						}
					}
				}

				return hasClusterUID
			}, timeout, interval).Should(BeTrue())
		})

		It("Should create a Processor with label attributes", func() {
			By("Creating a K8sAttributesResolver with label attributes")
			resolver := &actionv1.K8sAttributesResolver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-labels",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.K8sAttributesSpec{
					ActionName: "test-labels-k8sattributes",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					LabelsAttributes: []actionv1.K8sLabelAttribute{
						{
							LabelKey:     "app.kubernetes.io/name",
							AttributeKey: "app.kubernetes.name",
							From:         &[]actionv1.K8sAttributeSource{actionv1.PodAttributeSource}[0],
						},
						{
							LabelKey:     "app.kubernetes.io/component",
							AttributeKey: "app.kubernetes.component",
							From:         &[]actionv1.K8sAttributeSource{actionv1.PodAttributeSource}[0],
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, resolver)).Should(Succeed())

			By("Checking that a Processor is created with label attributes")
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

				// Check that label attributes are included
				extract, ok := config["extract"].(map[string]interface{})
				if !ok {
					return false
				}

				labels, ok := extract["labels"].([]interface{})
				if !ok {
					return false
				}

				// Check for specific label attributes
				hasAppName := false
				hasAppComponent := false

				for _, label := range labels {
					if labelMap, ok := label.(map[string]interface{}); ok {
						tagName, _ := labelMap["tag_name"].(string)
						key, _ := labelMap["key"].(string)
						from, _ := labelMap["from"].(string)

						if tagName == "app.kubernetes.name" && key == "app.kubernetes.io/name" && from == "pod" {
							hasAppName = true
						}
						if tagName == "app.kubernetes.component" && key == "app.kubernetes.io/component" && from == "pod" {
							hasAppComponent = true
						}
					}
				}

				return hasAppName && hasAppComponent
			}, timeout, interval).Should(BeTrue())
		})

		It("Should create a Processor with annotation attributes", func() {
			By("Creating a K8sAttributesResolver with annotation attributes")
			resolver := &actionv1.K8sAttributesResolver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-annotations",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.K8sAttributesSpec{
					ActionName: "test-annotations-k8sattributes",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AnnotationsAttributes: []actionv1.K8sAnnotationAttribute{
						{
							AnnotationKey: "kubectl.kubernetes.io/restartedAt",
							AttributeKey:  "kubectl.kubernetes.restartedAt",
							From:          &[]string{"pod"}[0],
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, resolver)).Should(Succeed())

			By("Checking that a Processor is created with annotation attributes")
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

				// Check that annotation attributes are included
				extract, ok := config["extract"].(map[string]interface{})
				if !ok {
					return false
				}

				annotations, ok := extract["annotations"].([]interface{})
				if !ok {
					return false
				}

				// Check for specific annotation attributes
				hasRestartedAt := false
				for _, annotation := range annotations {
					if annotationMap, ok := annotation.(map[string]interface{}); ok {
						tagName, _ := annotationMap["tag_name"].(string)
						key, _ := annotationMap["key"].(string)
						from, _ := annotationMap["from"].(string)

						if tagName == "kubectl.kubernetes.restartedAt" && key == "kubectl.kubernetes.io/restartedAt" && from == "pod" {
							hasRestartedAt = true
							break
						}
					}
				}

				return hasRestartedAt
			}, timeout, interval).Should(BeTrue())
		})

		It("Should create a Processor with multiple signals", func() {
			By("Creating a K8sAttributesResolver with multiple signals")
			resolver := &actionv1.K8sAttributesResolver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-multi-signal",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.K8sAttributesSpec{
					ActionName: "test-multi-signal-k8sattributes",
					Signals: []common.ObservabilitySignal{
						common.TracesObservabilitySignal,
						common.MetricsObservabilitySignal,
						common.LogsObservabilitySignal,
					},
				},
			}

			Expect(k8sClient.Create(testCtx, resolver)).Should(Succeed())

			By("Checking that a Processor is created with multiple signals")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "odigos-k8sattributes",
					Namespace: ActionNamespace,
				}, processor)
				if err != nil {
					return false
				}

				// Check that all signals are included
				expectedSignals := []common.ObservabilitySignal{
					common.TracesObservabilitySignal,
					common.MetricsObservabilitySignal,
					common.LogsObservabilitySignal,
				}

				for _, expectedSignal := range expectedSignals {
					found := false
					for _, actualSignal := range processor.Spec.Signals {
						if actualSignal == expectedSignal {
							found = true
							break
						}
					}
					if !found {
						return false
					}
				}

				return true
			}, timeout, interval).Should(BeTrue())
		})

		It("Should not create a Processor when disabled", func() {
			By("Creating a disabled K8sAttributesResolver")
			resolver := &actionv1.K8sAttributesResolver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-disabled",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.K8sAttributesSpec{
					ActionName: "test-disabled-k8sattributes",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					Disabled:   true,
				},
			}

			Expect(k8sClient.Create(testCtx, resolver)).Should(Succeed())

			By("Checking that no Processor is created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "odigos-k8sattributes",
					Namespace: ActionNamespace,
				}, processor)
				return err != nil
			}, timeout, interval).Should(Not(BeNil()))
		})
	})

	Context("When creating multiple K8sAttributesResolvers", func() {
		It("Should merge configurations from multiple resolvers", func() {
			By("Creating multiple K8sAttributesResolvers with different configurations")
			resolver1 := &actionv1.K8sAttributesResolver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-merge-1",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.K8sAttributesSpec{
					ActionName:                 "test-merge-1-k8sattributes",
					Signals:                    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					CollectContainerAttributes: true,
					LabelsAttributes: []actionv1.K8sLabelAttribute{
						{
							LabelKey:     "app.kubernetes.io/name",
							AttributeKey: "app.kubernetes.name",
						},
					},
				},
			}

			resolver2 := &actionv1.K8sAttributesResolver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-merge-2",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.K8sAttributesSpec{
					ActionName:         "test-merge-2-k8sattributes",
					Signals:            []common.ObservabilitySignal{common.MetricsObservabilitySignal},
					CollectWorkloadUID: true,
					LabelsAttributes: []actionv1.K8sLabelAttribute{
						{
							LabelKey:     "app.kubernetes.io/component",
							AttributeKey: "app.kubernetes.component",
						},
					},
				},
			}

			Expect(k8sClient.Create(testCtx, resolver1)).Should(Succeed())
			Expect(k8sClient.Create(testCtx, resolver2)).Should(Succeed())

			By("Checking that a Processor is created with merged configuration")
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

				// Check that both configurations are merged
				extract, ok := config["extract"].(map[string]interface{})
				if !ok {
					return false
				}

				// Check metadata includes container attributes (from resolver1)
				metadata, ok := extract["metadata"].([]interface{})
				if !ok {
					return false
				}

				hasContainerName := false
				for _, attr := range metadata {
					if attrStr, ok := attr.(string); ok {
						if attrStr == "k8s.container.name" {
							hasContainerName = true
							break
						}
					}
				}

				// Check labels include both label attributes
				labels, ok := extract["labels"].([]interface{})
				if !ok {
					return false
				}

				hasAppName := false
				hasAppComponent := false
				for _, label := range labels {
					if labelMap, ok := label.(map[string]interface{}); ok {
						tagName, _ := labelMap["tag_name"].(string)
						if tagName == "app.kubernetes.name" {
							hasAppName = true
						}
						if tagName == "app.kubernetes.component" {
							hasAppComponent = true
						}
					}
				}

				// Check signals include both
				hasTraces := false
				hasMetrics := false
				for _, signal := range processor.Spec.Signals {
					if signal == common.TracesObservabilitySignal {
						hasTraces = true
					}
					if signal == common.MetricsObservabilitySignal {
						hasMetrics = true
					}
				}

				return hasContainerName && hasAppName && hasAppComponent && hasTraces && hasMetrics
			}, timeout, interval).Should(BeTrue())

			// Check that both resolvers are in owner references
			ownerRefs := processor.GetOwnerReferences()
			Expect(len(ownerRefs)).Should(Equal(2))

			ownerNames := make(map[string]bool)
			ownerKinds := make(map[string]bool)
			for _, ownerRef := range ownerRefs {
				ownerNames[ownerRef.Name] = true
				ownerKinds[ownerRef.Kind] = true
			}
			Expect(ownerNames[odigosv1.ActionMigratedLegacyPrefix+ActionName+"-merge-1"]).Should(BeTrue())
			Expect(ownerNames[odigosv1.ActionMigratedLegacyPrefix+ActionName+"-merge-2"]).Should(BeTrue())
			Expect(ownerNames[ActionName+"-merge-1"]).Should(BeFalse())
			Expect(ownerNames[ActionName+"-merge-2"]).Should(BeFalse())
			Expect(ownerKinds["K8sAttributesResolver"]).Should(BeFalse())
			Expect(ownerKinds["Action"]).Should(BeTrue())
		})
	})

	Context("When updating a K8sAttributesResolver", func() {
		It("Should not update the corresponding Processor", func() {
			By("Creating a K8sAttributesResolver")
			resolver := &actionv1.K8sAttributesResolver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-update",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.K8sAttributesSpec{
					ActionName:                 "test-update-k8sattributes",
					Signals:                    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					CollectContainerAttributes: true,
				},
			}

			Expect(k8sClient.Create(testCtx, resolver)).Should(Succeed())

			By("Waiting for Processor to be created")
			processor := &odigosv1.Processor{}
			Eventually(func() bool {
				err := k8sClient.Get(testCtx, types.NamespacedName{
					Name:      "odigos-k8sattributes",
					Namespace: ActionNamespace,
				}, processor)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Updating the K8sAttributesResolver")
			Expect(k8sClient.Get(testCtx, types.NamespacedName{
				Name:      ActionName + "-update",
				Namespace: ActionNamespace,
			}, resolver)).Should(Succeed())
			resolver.Spec.CollectContainerAttributes = false
			Expect(k8sClient.Update(testCtx, resolver)).Should(Succeed())

			By("Checking that the Processor is not updated")
			Consistently(func() bool {
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

				// Check that container attributes are now included
				extract, ok := config["extract"].(map[string]interface{})
				if !ok {
					return false
				}

				metadata, ok := extract["metadata"].([]interface{})
				if !ok {
					return false
				}

				// Check for container-related attributes
				hasContainerName := false
				for _, attr := range metadata {
					if attrStr, ok := attr.(string); ok {
						if attrStr == "k8s.container.name" {
							hasContainerName = true
							break
						}
					}
				}

				return hasContainerName
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When merging legacy and modern K8sAttributes configurations", func() {
		It("Should merge legacy K8sAttributesResolver with modern Action", func() {
			By("Creating a modern Action with K8sAttributes")
			modernAction := &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-modern-merge",
					Namespace: ActionNamespace,
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "modern-k8sattributes",
					Signals:    []common.ObservabilitySignal{common.MetricsObservabilitySignal},
					K8sAttributes: &actionv1.K8sAttributesConfig{
						CollectWorkloadUID: true,
						CollectClusterUID:  true,
						LabelsAttributes: []actionv1.K8sLabelAttribute{
							{
								LabelKey:     "app.kubernetes.io/component",
								AttributeKey: "app.kubernetes.component",
							},
						},
						AnnotationsAttributes: []actionv1.K8sAnnotationAttribute{
							{
								AnnotationKey: "kubectl.kubernetes.io/restartedAt",
								AttributeKey:  "kubectl.kubernetes.restartedAt",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, modernAction)).Should(Succeed())

			By("Creating a legacy K8sAttributesResolver")
			legacyResolver := &actionv1.K8sAttributesResolver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ActionName + "-legacy-merge",
					Namespace: ActionNamespace,
				},
				Spec: actionv1.K8sAttributesSpec{
					ActionName:                 "legacy-k8sattributes",
					Signals:                    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					CollectContainerAttributes: true,
					LabelsAttributes: []actionv1.K8sLabelAttribute{
						{
							LabelKey:     "app.kubernetes.io/name",
							AttributeKey: "app.kubernetes.name",
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, legacyResolver)).Should(Succeed())

			By("Checking that a Processor is created with merged configuration")
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

				// Check that both configurations are merged
				extract, ok := config["extract"].(map[string]interface{})
				if !ok {
					return false
				}

				// Check metadata includes attributes from both configurations
				metadata, ok := extract["metadata"].([]interface{})
				if !ok {
					return false
				}

				// Check for container attributes (from legacy)
				hasContainerName := false
				// Check for cluster UID attributes (from modern)
				hasClusterUID := false

				for _, attr := range metadata {
					if attrStr, ok := attr.(string); ok {
						switch attrStr {
						case "k8s.container.name":
							hasContainerName = true
						case "k8s.cluster.uid":
							hasClusterUID = true
						}
					}
				}

				// Check labels include both label attributes
				labels, ok := extract["labels"].([]interface{})
				if !ok {
					return false
				}

				hasAppName := false
				hasAppComponent := false
				for _, label := range labels {
					if labelMap, ok := label.(map[string]interface{}); ok {
						tagName, _ := labelMap["tag_name"].(string)
						if tagName == "app.kubernetes.name" {
							hasAppName = true
						}
						if tagName == "app.kubernetes.component" {
							hasAppComponent = true
						}
					}
				}

				// Check annotations include modern annotation
				annotations, ok := extract["annotations"].([]interface{})
				if !ok {
					return false
				}

				hasRestartedAt := false
				for _, annotation := range annotations {
					if annotationMap, ok := annotation.(map[string]interface{}); ok {
						tagName, _ := annotationMap["tag_name"].(string)
						if tagName == "kubectl.kubernetes.restartedAt" {
							hasRestartedAt = true
							break
						}
					}
				}

				// Check signals include both
				hasTraces := false
				hasMetrics := false
				for _, signal := range processor.Spec.Signals {
					if signal == common.TracesObservabilitySignal {
						hasTraces = true
					}
					if signal == common.MetricsObservabilitySignal {
						hasMetrics = true
					}
				}

				return hasContainerName && hasClusterUID &&
					hasAppName && hasAppComponent && hasRestartedAt &&
					hasTraces && hasMetrics
			}, timeout, interval).Should(BeTrue())

			// Check that both actions are in owner references
			ownerRefs := processor.GetOwnerReferences()
			Expect(len(ownerRefs)).Should(Equal(2))

			ownerNames := make(map[string]bool)
			ownerKinds := make(map[string]bool)
			for _, ownerRef := range ownerRefs {
				ownerNames[ownerRef.Name] = true
				ownerKinds[ownerRef.Kind] = true
			}
			Expect(ownerNames[ActionName+"-legacy-merge"]).Should(BeFalse())
			Expect(ownerNames[odigosv1.ActionMigratedLegacyPrefix+ActionName+"-legacy-merge"]).Should(BeTrue())
			Expect(ownerNames[ActionName+"-modern-merge"]).Should(BeTrue())
			Expect(ownerKinds["K8sAttributesResolver"]).Should(BeFalse())
			Expect(ownerKinds["Action"]).Should(BeTrue())
		})
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
