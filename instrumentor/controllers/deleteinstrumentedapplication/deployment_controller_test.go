package deleteinstrumentedapplication

import (
	"context"
	"time"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/common/consts"
	"github.com/keyval-dev/odigos/common/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("DeleteInstrumentedApplication Deployment controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		DeploymentName      = "test-deployment"
		DeploymentNamespace = "default"
		DeploymentKind      = "Deployment"

		timeout  = time.Second * 10
		duration = time.Second * 2
		interval = time.Millisecond * 250
	)

	baseTestDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DeploymentName,
			Namespace: DeploymentNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "test"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "test",
						},
					},
				},
			},
		},
	}

	baseInstrumentedApplication := &odigosv1.InstrumentedApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.GetRuntimeObjectName(DeploymentName, DeploymentKind),
			Namespace: DeploymentNamespace,
		},
		Spec: odigosv1.InstrumentedApplicationSpec{
			Languages: []common.LanguageByContainer{
				{
					ContainerName: "test",
					Language:      common.GoProgrammingLanguage,
				},
			},
		},
	}

	Context("Delete instrumented application when un-instrumenting a deployment", func() {
		var ctx context.Context
		var deployment *appsv1.Deployment
		var instrumentedApplication *odigosv1.InstrumentedApplication

		BeforeEach(func() {
			ctx = context.Background()
			deployment = baseTestDeployment.DeepCopy()
			deployment.Labels = map[string]string{consts.OdigosInstrumentationLabel: consts.InstrumentationEnabled}
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			instrumentedApplication = baseInstrumentedApplication.DeepCopy()
			Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, deployment)).Should(Succeed())
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, instrumentedApplication))).Should(Succeed())
			Expect(deleteNsLabels(ctx, DeploymentNamespace)).Should(Succeed())
		})

		It("should delete the associated runtime details on instrumentation label deleted", func() {

			By("By removing the instrumentation label from the deployment")
			delete(deployment.Labels, consts.OdigosInstrumentationLabel)
			Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())

			By("By checking the runtime details are deleted")
			Eventually(func() bool {
				var runtimeDetails odigosv1.InstrumentedApplication
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: DeploymentNamespace, Name: utils.GetRuntimeObjectName(DeploymentName, DeploymentKind)}, &runtimeDetails)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})

		It("should delete the associated runtime details on instrumentation label set to disabled", func() {

			By("By setting the instrumentation label to disabled")
			deployment.Labels[consts.OdigosInstrumentationLabel] = consts.InstrumentationDisabled
			Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())

			By("By checking the runtime details are deleted")
			Eventually(func() bool {
				var runtimeDetails odigosv1.InstrumentedApplication
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: DeploymentNamespace, Name: utils.GetRuntimeObjectName(DeploymentName, DeploymentKind)}, &runtimeDetails)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})

		It("should delete the associated runtime details even when ns is instrumented", func() {

			By("By setting the ns instrumentation label to enabled")
			Expect(setNsInstrumentationEnabled(ctx, DeploymentNamespace)).Should(Succeed())

			By("By specifying the workload instrumentation label to disabled which overrides the ns instrumentation label")
			deployment.Labels[consts.OdigosInstrumentationLabel] = consts.InstrumentationDisabled
			Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())

			By("By checking the runtime details are deleted")
			Eventually(func() bool {
				var runtimeDetails odigosv1.InstrumentedApplication
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: DeploymentNamespace, Name: utils.GetRuntimeObjectName(DeploymentName, DeploymentKind)}, &runtimeDetails)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Retain instrumented application when un-instrumenting a deployment", func() {
		var ctx context.Context
		var deployment *appsv1.Deployment
		var instrumentedApplication *odigosv1.InstrumentedApplication

		BeforeEach(func() {
			ctx = context.Background()
			deployment = baseTestDeployment.DeepCopy()
			deployment.Labels = map[string]string{consts.OdigosInstrumentationLabel: consts.InstrumentationEnabled}
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			instrumentedApplication = baseInstrumentedApplication.DeepCopy()
			Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, deployment)).Should(Succeed())
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, instrumentedApplication))).Should(Succeed())
			Expect(deleteNsLabels(ctx, DeploymentNamespace)).Should(Succeed())
		})

		It("should retain the associated runtime details on instrumentation label deleted when ns is instrumented", func() {

			By("By setting the ns instrumentation label to enabled")
			Expect(setNsInstrumentationEnabled(ctx, DeploymentNamespace)).Should(Succeed())

			By("By removing the instrumentation label from the deployment")
			delete(deployment.Labels, consts.OdigosInstrumentationLabel)
			Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())

			By("By checking the runtime details are retained")
			Consistently(func() bool {
				var runtimeDetails odigosv1.InstrumentedApplication
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: DeploymentNamespace, Name: utils.GetRuntimeObjectName(DeploymentName, DeploymentKind)}, &runtimeDetails)
				return err == nil
			}, duration, interval).Should(BeTrue())
		})
	})

	Context("Delete reported name annotation when un-instrumenting a deployment", func() {
		var ctx context.Context
		var deployment *appsv1.Deployment
		var instrumentedApplication *odigosv1.InstrumentedApplication

		BeforeEach(func() {
			ctx = context.Background()
			deployment = baseTestDeployment.DeepCopy()
			deployment.Labels = map[string]string{consts.OdigosInstrumentationLabel: consts.InstrumentationEnabled}
			deployment.Annotations = map[string]string{consts.OdigosReportedNameAnnotation: "test"}
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			instrumentedApplication = baseInstrumentedApplication.DeepCopy()
			Expect(k8sClient.Create(ctx, instrumentedApplication)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, deployment)).Should(Succeed())
		})

		It("should delete the reported name annotation on instrumentation label deleted", func() {

			By("By removing the instrumentation label from the deployment")
			deployment.Labels[consts.OdigosInstrumentationLabel] = consts.InstrumentationDisabled
			Expect(k8sClient.Update(ctx, deployment)).Should(Succeed())

			By("By checking the reported name annotation is deleted")
			Eventually(func() bool {
				var dep appsv1.Deployment
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: DeploymentNamespace, Name: DeploymentName}, &dep)
				if err != nil {
					return false
				}
				_, foundAnnotation := dep.Annotations[consts.OdigosReportedNameAnnotation]
				if foundAnnotation {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})
	})
})

func deleteNsLabels(ctx context.Context, ns string) error {
	return k8sClient.Patch(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}, client.RawPatch(types.MergePatchType, []byte(`{
		"metadata": {
			"labels": null
		}
	}`)))
}

func setNsInstrumentationEnabled(ctx context.Context, ns string) error {
	return k8sClient.Patch(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}, client.RawPatch(types.MergePatchType, []byte(`{
		"metadata": {
			"labels": {
				"`+consts.OdigosInstrumentationLabel+`": "`+consts.InstrumentationEnabled+`"
			}
		}
	}`)))
}
