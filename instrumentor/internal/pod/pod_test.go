package pod

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	k8snode "github.com/odigos-io/odigos/k8sutils/pkg/node"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPod(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pod Utilities Suite")
}

var _ = Describe("AddOdigletInstalledAffinity", func() {
	var (
		pod                   *corev1.Pod
		odigletInstalledLabel string
	)

	BeforeEach(func() {
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "test-container",
						Image: "test-image",
					},
				},
			},
		}
		odigletInstalledLabel = k8snode.DetermineNodeOdigletInstalledLabelByTier()
	})

	Context("when pod has no affinity", func() {
		It("should add node affinity with odiglet installed requirement", func() {
			AddOdigletInstalledAffinity(pod)

			Expect(pod.Spec.Affinity).NotTo(BeNil())
			Expect(pod.Spec.Affinity.NodeAffinity).NotTo(BeNil())
			Expect(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).NotTo(BeNil())
			Expect(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(1))

			term := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0]
			Expect(term.MatchExpressions).To(HaveLen(1))
			Expect(term.MatchExpressions[0].Key).To(Equal(odigletInstalledLabel))
			Expect(term.MatchExpressions[0].Operator).To(Equal(corev1.NodeSelectorOpIn))
			Expect(term.MatchExpressions[0].Values).To(HaveLen(1))
			Expect(term.MatchExpressions[0].Values[0]).To(Equal(k8sconsts.OdigletInstalledLabelValue))
		})
	})

	Context("when pod has empty affinity", func() {
		BeforeEach(func() {
			pod.Spec.Affinity = &corev1.Affinity{}
		})

		It("should add node affinity with odiglet installed requirement", func() {
			AddOdigletInstalledAffinity(pod)

			Expect(pod.Spec.Affinity.NodeAffinity).NotTo(BeNil())
			Expect(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).NotTo(BeNil())
			Expect(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(1))

			term := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0]
			Expect(term.MatchExpressions).To(HaveLen(1))
			Expect(term.MatchExpressions[0].Key).To(Equal(odigletInstalledLabel))
			Expect(term.MatchExpressions[0].Operator).To(Equal(corev1.NodeSelectorOpIn))
			Expect(term.MatchExpressions[0].Values).To(ContainElement(k8sconsts.OdigletInstalledLabelValue))
		})
	})

	Context("when pod has existing node affinity", func() {
		BeforeEach(func() {
			pod.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "existing-label",
										Operator: corev1.NodeSelectorOpIn,
										Values:   []string{"existing-value"},
									},
								},
							},
						},
					},
				},
			}
		})

		It("should append new node selector term", func() {
			AddOdigletInstalledAffinity(pod)

			Expect(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(2))

			// Check existing term is preserved
			existingTerm := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0]
			Expect(existingTerm.MatchExpressions).To(HaveLen(1))
			Expect(existingTerm.MatchExpressions[0].Key).To(Equal("existing-label"))

			// Check new term is added
			newTerm := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[1]
			Expect(newTerm.MatchExpressions).To(HaveLen(1))
			Expect(newTerm.MatchExpressions[0].Key).To(Equal(odigletInstalledLabel))
			Expect(newTerm.MatchExpressions[0].Operator).To(Equal(corev1.NodeSelectorOpIn))
			Expect(newTerm.MatchExpressions[0].Values).To(ContainElement(k8sconsts.OdigletInstalledLabelValue))
		})
	})

	Context("when odiglet affinity already exists", func() {
		BeforeEach(func() {
			pod.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      odigletInstalledLabel,
										Operator: corev1.NodeSelectorOpIn,
										Values:   []string{k8sconsts.OdigletInstalledLabelValue},
									},
								},
							},
						},
					},
				},
			}
		})

		It("should not add duplicate affinity", func() {
			originalTermsCount := len(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)

			AddOdigletInstalledAffinity(pod)

			Expect(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(originalTermsCount))
		})
	})

	Context("when odiglet affinity exists with additional values", func() {
		BeforeEach(func() {
			pod.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      odigletInstalledLabel,
										Operator: corev1.NodeSelectorOpIn,
										Values:   []string{"other-value", k8sconsts.OdigletInstalledLabelValue},
									},
								},
							},
						},
					},
				},
			}
		})

		It("should not add duplicate affinity when value already exists", func() {
			originalTermsCount := len(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)

			AddOdigletInstalledAffinity(pod)

			Expect(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(originalTermsCount))
		})
	})

	Context("when pod has different operator for odiglet label", func() {
		BeforeEach(func() {
			pod.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      odigletInstalledLabel,
										Operator: corev1.NodeSelectorOpExists,
									},
								},
							},
						},
					},
				},
			}
		})

		It("should add new term with In operator", func() {
			AddOdigletInstalledAffinity(pod)

			Expect(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(2))

			// Check new term is added with In operator
			newTerm := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[1]
			Expect(newTerm.MatchExpressions).To(HaveLen(1))
			Expect(newTerm.MatchExpressions[0].Key).To(Equal(odigletInstalledLabel))
			Expect(newTerm.MatchExpressions[0].Operator).To(Equal(corev1.NodeSelectorOpIn))
			Expect(newTerm.MatchExpressions[0].Values).To(ContainElement(k8sconsts.OdigletInstalledLabelValue))
		})
	})
})
