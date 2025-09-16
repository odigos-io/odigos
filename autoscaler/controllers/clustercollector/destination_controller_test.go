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

package clustercollector_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

const (
	timeout  = time.Second * 10
	interval = time.Millisecond * 250
)

var _ = Describe("Destination Controller", func() {
	const (
		DestinationName      = "test-destination"
		DestinationNamespace = "odigos-system"
		SecretName           = "test-secret"
	)

	AfterEach(func() {
		secretList := &corev1.SecretList{}
		k8sClient.List(context.Background(), secretList)
		for _, secret := range secretList.Items {
			Expect(k8sClient.Delete(context.Background(), &secret)).Should(Succeed())
		}

		destinationList := &odigosv1.DestinationList{}
		k8sClient.List(context.Background(), destinationList)
		for _, destination := range destinationList.Items {
			Expect(k8sClient.Delete(context.Background(), &destination)).Should(Succeed())
		}
		resetCollectorDeployment()
	})

	Context("When creating a GoogleCloud Destination with APPLICATION_CREDENTIALS", func() {
		It("Should create a cluster collector deployment with volume, volume mount, and env var", func() {

			By("Creating a secret with GCP credentials")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SecretName,
					Namespace: DestinationNamespace,
				},
				Data: map[string][]byte{
					"GCP_APPLICATION_CREDENTIALS": []byte("fake-gcp-credentials"),
				},
			}
			Expect(k8sClient.Create(context.Background(), secret)).Should(Succeed())

			By("Creating a GoogleCloud destination with APPLICATION_CREDENTIALS")
			destination := &odigosv1.Destination{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DestinationName,
					Namespace: DestinationNamespace,
				},
				Spec: odigosv1.DestinationSpec{
					Type:            common.GoogleCloudDestinationType,
					DestinationName: "test-gcp-destination",
					Data: map[string]string{
						"GCP_APPLICATION_CREDENTIALS": "fake-gcp-credentials",
					},
					SecretRef: &corev1.LocalObjectReference{
						Name: SecretName,
					},
					Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
				},
			}
			Expect(k8sClient.Create(context.Background(), destination)).Should(Succeed())

			By("Waiting for the cluster collector deployment to exist")
			deployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      k8sconsts.OdigosClusterCollectorDeploymentName,
					Namespace: DestinationNamespace,
				}, deployment)
				Expect(err).To(BeNil())
				return len(deployment.Spec.Template.Spec.Volumes) == 1
			}, timeout, interval).Should(BeTrue())

			By("Verifying the deployment has the expected volume")
			Expect(deployment.Spec.Template.Spec.Volumes).To(HaveLen(1))
			volume := deployment.Spec.Template.Spec.Volumes[0]
			Expect(volume.Name).To(Equal(SecretName))
			Expect(volume.VolumeSource.Secret).NotTo(BeNil())
			Expect(volume.VolumeSource.Secret.SecretName).To(Equal(SecretName))
			Expect(volume.VolumeSource.Secret.Items).To(HaveLen(1))
			Expect(volume.VolumeSource.Secret.Items[0].Key).To(Equal("GCP_APPLICATION_CREDENTIALS"))
			Expect(volume.VolumeSource.Secret.Items[0].Path).To(Equal("GCP_APPLICATION_CREDENTIALS"))

			By("Verifying the deployment has the expected volume mount")
			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			container := deployment.Spec.Template.Spec.Containers[0]
			Expect(container.VolumeMounts).To(HaveLen(1))
			volumeMount := container.VolumeMounts[0]
			Expect(volumeMount.Name).To(Equal(SecretName))
			Expect(volumeMount.MountPath).To(Equal("/secrets"))

			By("Verifying the deployment has the expected environment variable")
			hasName := false
			hasValue := false
			for _, envVar := range container.Env {
				if envVar.Name == "GOOGLE_APPLICATION_CREDENTIALS" {
					hasName = true
				}
				if envVar.Value == "/secrets/GCP_APPLICATION_CREDENTIALS" {
					hasValue = true
				}
			}
			Expect(hasName).To(BeTrue())
			Expect(hasValue).To(BeTrue())
		})
	})

	Context("When creating multiple GoogleCloud Destinations with APPLICATION_CREDENTIALS", func() {
		It("Should not create duplicate volumes, volume mounts, or env vars", func() {
			By("Waiting for the cluster collector deployment to be created")
			deployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      k8sconsts.OdigosClusterCollectorDeploymentName,
					Namespace: DestinationNamespace,
				}, deployment)
				Expect(err).To(BeNil())
				return len(deployment.Spec.Template.Spec.Volumes) == 0
			}, timeout, interval).Should(BeTrue())

			By("Creating a secret with GCP credentials")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SecretName,
					Namespace: DestinationNamespace,
				},
				Data: map[string][]byte{
					"GCP_APPLICATION_CREDENTIALS": []byte("fake-gcp-credentials"),
				},
			}
			Expect(k8sClient.Create(context.Background(), secret)).Should(Succeed())

			By("Creating the first GoogleCloud destination")
			destination1 := &odigosv1.Destination{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DestinationName + "-1",
					Namespace: DestinationNamespace,
				},
				Spec: odigosv1.DestinationSpec{
					Type:            common.GoogleCloudDestinationType,
					DestinationName: "test-gcp-destination-1",
					Data: map[string]string{
						"GCP_APPLICATION_CREDENTIALS": "fake-gcp-credentials",
					},
					SecretRef: &corev1.LocalObjectReference{
						Name: SecretName,
					},
					Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
				},
			}
			Expect(k8sClient.Create(context.Background(), destination1)).Should(Succeed())

			By("Waiting for the cluster collector deployment to be updated")
			deployment = &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      k8sconsts.OdigosClusterCollectorDeploymentName,
					Namespace: DestinationNamespace,
				}, deployment)
				Expect(err).To(BeNil())
				return len(deployment.Spec.Template.Spec.Volumes) == 1
			}, timeout, interval).Should(BeTrue())

			By("Creating the second GoogleCloud destination with the same secret")
			destination2 := &odigosv1.Destination{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DestinationName + "-2",
					Namespace: DestinationNamespace,
				},
				Spec: odigosv1.DestinationSpec{
					Type:            common.GoogleCloudDestinationType,
					DestinationName: "test-gcp-destination-2",
					Data: map[string]string{
						"GCP_APPLICATION_CREDENTIALS": "fake-gcp-credentials",
					},
					SecretRef: &corev1.LocalObjectReference{
						Name: SecretName,
					},
					Signals: []common.ObservabilitySignal{common.MetricsObservabilitySignal},
				},
			}
			Expect(k8sClient.Create(context.Background(), destination2)).Should(Succeed())

			By("Waiting for the cluster collector deployment to be unchanged")
			deployment = &appsv1.Deployment{}
			Consistently(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      k8sconsts.OdigosClusterCollectorDeploymentName,
					Namespace: DestinationNamespace,
				}, deployment)
				Expect(err).To(BeNil())
				return len(deployment.Spec.Template.Spec.Volumes) == 1
			}, timeout, interval).Should(BeTrue())

			By("Verifying the deployment has only one volume (no duplicates)")
			Expect(deployment.Spec.Template.Spec.Volumes).To(HaveLen(1))
			volume := deployment.Spec.Template.Spec.Volumes[0]
			Expect(volume.Name).To(Equal(SecretName))

			By("Verifying the deployment has only one volume mount (no duplicates)")
			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			container := deployment.Spec.Template.Spec.Containers[0]
			Expect(container.VolumeMounts).To(HaveLen(1))
			volumeMount := container.VolumeMounts[0]
			Expect(volumeMount.Name).To(Equal(SecretName))

			By("Verifying the deployment has only one environment variable (no duplicates)")
			hasName := 0
			hasValue := 0
			for _, envVar := range container.Env {
				if envVar.Name == "GOOGLE_APPLICATION_CREDENTIALS" {
					hasName++
				}
				if envVar.Value == "/secrets/GCP_APPLICATION_CREDENTIALS" {
					hasValue++
				}
			}
			Expect(hasName).To(Equal(1))
			Expect(hasValue).To(Equal(1))
		})
	})

	Context("When deleting a Google Cloud Destination with APPLICATION_CREDENTIALS", func() {
		It("Should remove the volume, volume mount, and env var", func() {
			By("Creating a secret with GCP credentials")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SecretName,
					Namespace: DestinationNamespace,
				},
				Data: map[string][]byte{
					"GCP_APPLICATION_CREDENTIALS": []byte("fake-gcp-credentials"),
				},
			}
			Expect(k8sClient.Create(context.Background(), secret)).Should(Succeed())

			By("Creating a GoogleCloud destination with APPLICATION_CREDENTIALS")
			destination := &odigosv1.Destination{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DestinationName,
					Namespace: DestinationNamespace,
				},
				Spec: odigosv1.DestinationSpec{
					Type:            common.GoogleCloudDestinationType,
					DestinationName: "test-gcp-destination",
					Data: map[string]string{
						"GCP_APPLICATION_CREDENTIALS": "fake-gcp-credentials",
					},
					SecretRef: &corev1.LocalObjectReference{
						Name: SecretName,
					},
					Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
				},
			}
			Expect(k8sClient.Create(context.Background(), destination)).Should(Succeed())

			By("Waiting for the cluster collector deployment to be updated")
			deployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      k8sconsts.OdigosClusterCollectorDeploymentName,
					Namespace: DestinationNamespace,
				}, deployment)
				Expect(err).To(BeNil())
				return len(deployment.Spec.Template.Spec.Volumes) == 1
			}, timeout, interval).Should(BeTrue())

			By("Deleting the GoogleCloud destination")
			Expect(k8sClient.Delete(context.Background(), destination)).Should(Succeed())

			By("Waiting for the cluster collector deployment to be updated")
			deployment = &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      k8sconsts.OdigosClusterCollectorDeploymentName,
					Namespace: DestinationNamespace,
				}, deployment)
				Expect(err).To(BeNil())
				return len(deployment.Spec.Template.Spec.Volumes) == 0
			}, timeout, interval).Should(BeTrue())

			By("Verifying the deployment has no volumes")
			Expect(deployment.Spec.Template.Spec.Volumes).To(HaveLen(0))

			By("Verifying the deployment has no volume mounts")
			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			container := deployment.Spec.Template.Spec.Containers[0]
			Expect(container.VolumeMounts).To(HaveLen(0))

			By("Verifying the deployment has no GCP environment variables")
			hasName := 0
			hasValue := 0
			for _, envVar := range container.Env {
				if envVar.Name == "GOOGLE_APPLICATION_CREDENTIALS" {
					hasName++
				}
			}
			Expect(hasName).To(Equal(0))
			Expect(hasValue).To(Equal(0))
		})
	})
})
