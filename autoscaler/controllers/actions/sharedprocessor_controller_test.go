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
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	actions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Shared URLTemplatization Processor", func() {
	const (
		ActionName      = "test-url-templatization-action"
		ActionNamespace = "default"
	)

	AfterEach(func() {
		cleanupResources()
	})

	It("Should remove shared processor when last URLTemplatization action is deleted", func() {
		By("Creating a URLTemplatization action")
		action := &odigosv1.Action{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ActionName,
				Namespace: ActionNamespace,
			},
			Spec: odigosv1.ActionSpec{
				ActionName: "test-url-templatization",
				Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
				URLTemplatization: &actions.URLTemplatizationConfig{
					TemplatizationRulesGroups: []actions.UrlTemplatizationRulesGroup{
						{
							TemplatizationRules: []actions.URLTemplatizationRule{
								{Template: "/users/{id}"},
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(testCtx, action)).Should(Succeed())

		By("Checking that the shared URLTemplatization processor is created")
		processor := &odigosv1.Processor{}
		Eventually(func() bool {
			err := k8sClient.Get(testCtx, types.NamespacedName{
				Name:      consts.URLTemplatizationProcessorName,
				Namespace: ActionNamespace,
			}, processor)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("Deleting the URLTemplatization action")
		Expect(k8sClient.Delete(testCtx, action)).Should(Succeed())

		By("Checking that the shared URLTemplatization processor is removed")
		Eventually(func() bool {
			err := k8sClient.Get(testCtx, types.NamespacedName{
				Name:      consts.URLTemplatizationProcessorName,
				Namespace: ActionNamespace,
			}, processor)
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})
})
