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
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/clustercollector"
	commonconfig "github.com/odigos-io/odigos/autoscaler/controllers/common"
	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"
	"github.com/odigos-io/odigos/common/consts"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var testCtx context.Context
var cancel context.CancelFunc

const DestinationNamespace = "odigos-system"

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "ClusterCollector Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	testCtx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "api", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = odigosv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = actionv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	Expect(err).ToNot(HaveOccurred())

	err = clustercollector.SetupWithManager(k8sManager, "")
	Expect(err).ToNot(HaveOccurred())

	setupResources()

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(testCtx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func setupResources() {
	intPtr := func(n int32) *int32 {
		return &n
	}

	commonconfig.ControllerConfig = &controllerconfig.ControllerConfig{
		K8sVersion:     version.MustParseSemantic("0.0.0"),
		CollectorImage: "otelcol",
		OnGKE:          false,
	}

	By("Creating the odigos-system namespace")
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: DestinationNamespace,
		},
	}
	Expect(k8sClient.Create(context.Background(), namespace)).Should(Succeed())

	By("Creating the effective config")
	effectiveConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosEffectiveConfigName,
			Namespace: DestinationNamespace,
		},
		Data: map[string]string{},
	}
	Expect(k8sClient.Create(context.Background(), effectiveConfig)).Should(Succeed())

	By("Creating the odiglet daemonset")
	odigletDaemonset := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigletDaemonSetName,
			Namespace: DestinationNamespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": k8sconsts.OdigletDaemonSetName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": k8sconsts.OdigletDaemonSetName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.OdigletContainerName,
							Image: "odigos/odiglet:latest",
						},
					},
				},
			},
		},
	}
	Expect(k8sClient.Create(context.Background(), odigletDaemonset)).Should(Succeed())

	By("Creating the autoscaler deployment")
	autoscalerDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.AutoScalerDeploymentName,
			Namespace: DestinationNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: intPtr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": k8sconsts.AutoScalerDeploymentName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": k8sconsts.AutoScalerDeploymentName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  k8sconsts.AutoScalerContainerName,
							Image: "odigos/autoscaler:latest",
						},
					},
				},
			},
		},
	}
	Expect(k8sClient.Create(context.Background(), autoscalerDeployment)).Should(Succeed())

	createCollectorsGroupAndDeployment()
}

func defaultCollectorDeployment(replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorDeploymentName,
			Namespace: DestinationNamespace,
			Labels: map[string]string{
				k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleClusterGateway),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleClusterGateway),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleClusterGateway),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "gateway",
							Image:   "odigos/odigosotelcol:latest",
							Command: []string{"/odigosotelcol"},
							Args: []string{fmt.Sprintf("--config=%s:%s/%s/%s",
								k8sconsts.OdigosCollectorConfigMapProviderScheme,
								DestinationNamespace,
								k8sconsts.OdigosClusterCollectorConfigMapName,
								k8sconsts.OdigosClusterCollectorConfigMapKey),
							},
							Env: []corev1.EnvVar{
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("250m"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("512Mi"),
									corev1.ResourceCPU:    resource.MustParse("500m"),
								},
							},
						},
					},
				},
			},
		},
	}
}

func createCollectorsGroupAndDeployment() {
	By("Creating a CollectorsGroup for cluster collector")
	collectorsGroup := &odigosv1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorConfigMapName,
			Namespace: DestinationNamespace,
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role: odigosv1.CollectorsGroupRoleClusterGateway,
			ResourcesSettings: odigosv1.CollectorsGroupResourcesSettings{
				MemoryRequestMiB:     256,
				MemoryLimitMiB:       512,
				CpuRequestMillicores: 250,
				CpuLimitMillicores:   500,
				GomemlimitMiB:        200,
			},
			CollectorOwnMetricsPort: 8888,
		},
	}
	Expect(k8sClient.Create(context.Background(), collectorsGroup)).Should(Succeed())

	By("Creating the cluster collector deployment")
	deployment := defaultCollectorDeployment(1)
	Expect(k8sClient.Create(context.Background(), deployment)).Should(Succeed())
}

func resetCollectorDeployment() {
	deployment := defaultCollectorDeployment(1)
	Expect(k8sClient.Update(context.Background(), deployment)).Should(Succeed())
}
