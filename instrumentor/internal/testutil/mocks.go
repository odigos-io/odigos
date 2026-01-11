package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	mockNamespaceBase   = "test-namespace"
	mockDaemonSetName   = "test-daemonset"
	mockStatefulSetName = "test-statefulset"
)

var (
	mockDefaultSDKs = map[common.ProgrammingLanguage]common.OtelSdk{}
)

func NewOdigosSystemNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "odigos-system",
		},
	}
}

func NewMockNamespace() *corev1.Namespace {
	name := generateUUIDNamespace(mockNamespaceBase)
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func NewMockOdigosConfig() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosConfigurationName,
			Namespace: consts.DefaultOdigosNamespace,
		},
	}
}

func NewMockTestDeployment(ns *corev1.Namespace, name string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns.GetName(),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": name},
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
}

func NewMockTestDaemonSet(ns *corev1.Namespace) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockDaemonSetName,
			Namespace: ns.GetName(),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "test-ds"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": "test-ds"},
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
}

func NewMockTestStatefulSet(ns *corev1.Namespace) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockStatefulSetName,
			Namespace: ns.GetName(),
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "test-ss"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": "test-ss"},
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
}

func NewMockTestCronJob(ns *corev1.Namespace, name string) *batchv1beta1.CronJob {
	return &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns.GetName(),
		},
		Spec: batchv1beta1.CronJobSpec{
			Schedule: "*/5 * * * *",
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app.kubernetes.io/name": name},
						},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "test",
								},
							},
						},
					},
				},
			},
		},
	}
}

func NewMockTestJob(ns *corev1.Namespace, name string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns.GetName(),
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": name},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
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
}

// NewMockSource returns a single source for a workload (deployment, daemonset, statefulset)
func NewMockSource(workloadObject client.Object, disabled bool) *odigosv1.Source {
	gvk, _ := apiutil.GVKForObject(workloadObject, scheme.Scheme)
	namespace := workloadObject.GetNamespace()
	if gvk.Kind == string(k8sconsts.WorkloadKindNamespace) && len(namespace) == 0 {
		namespace = workloadObject.GetName()
	}
	return &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workload.CalculateWorkloadRuntimeObjectName(workloadObject.GetName(), gvk.Kind),
			Namespace: namespace,
			Labels: map[string]string{
				k8sconsts.WorkloadNameLabel:      workloadObject.GetName(),
				k8sconsts.WorkloadNamespaceLabel: namespace,
				k8sconsts.WorkloadKindLabel:      gvk.Kind,
			},
			Finalizers: []string{k8sconsts.DeleteInstrumentationConfigFinalizer},
		},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Name:      workloadObject.GetName(),
				Namespace: namespace,
				Kind:      k8sconsts.WorkloadKind(gvk.Kind),
			},
			DisableInstrumentation: disabled,
		},
	}
}

// NewMockRegexSource returns a single source for a deployment, based on a regex pattern
func NewMockRegexSource(workloadObject client.Object, pattern string, disabled bool) *odigosv1.Source {
	gvk, _ := apiutil.GVKForObject(workloadObject, scheme.Scheme)
	namespace := workloadObject.GetNamespace()
	patternHash := sha256.Sum256([]byte(pattern))
	// For regex sources, the label should be a hash of the pattern (as set by the webhook defaulter)
	name := "regex-" + hex.EncodeToString(patternHash[:])[:16]
	return &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				k8sconsts.WorkloadNamespaceLabel: namespace,
				k8sconsts.WorkloadKindLabel:      gvk.Kind,
			},
			Finalizers: []string{k8sconsts.DeleteInstrumentationConfigFinalizer},
		},
		Spec: odigosv1.SourceSpec{
			Workload: k8sconsts.PodWorkload{
				Name:      pattern,
				Namespace: namespace,
				Kind:      k8sconsts.WorkloadKindDeployment,
			},
			DisableInstrumentation:   disabled,
			MatchWorkloadNameAsRegex: true,
		},
	}
}

// givin a workload object (deployment, daemonset, statefulset) return a mock instrumented application
// with a single container with the GoProgrammingLanguage
func NewMockInstrumentationConfig(workloadObject client.Object) *odigosv1.InstrumentationConfig {
	gvk, _ := apiutil.GVKForObject(workloadObject, scheme.Scheme)
	return &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workload.CalculateWorkloadRuntimeObjectName(workloadObject.GetName(), gvk.Kind),
			Namespace: workloadObject.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
					Name:       workloadObject.GetName(),
					UID:        workloadObject.GetUID(),
				},
			},
		},
		Status: odigosv1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1.RuntimeDetailsByContainer{
				{
					ContainerName: "test",
					Language:      common.GoProgrammingLanguage,
				},
			},
		},
	}
}

func NewMockEmptyInstrumentationRule(name, ns string) *odigosv1.InstrumentationRule {
	return &odigosv1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: odigosv1.InstrumentationRuleSpec{},
	}
}

func NewMockDataCollection() *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorDaemonSetName,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: odigosv1.CollectorsGroupSpec{
			Role: odigosv1.CollectorsGroupRoleNodeCollector,
		},
	}
}

// this helps to avoid the "already exists" error when creating a new namespace.
// it promotes test isolation and avoid conflicts between tests.
func generateUUIDNamespace(baseName string) string {
	return fmt.Sprintf("%s-%s", baseName, uuid.New().String())
}

func MockGetDefaultSDKs() map[common.ProgrammingLanguage]common.OtelSdk {
	return mockDefaultSDKs
}

func SetDefaultSDK(language common.ProgrammingLanguage, sdk common.OtelSdk) {
	mockDefaultSDKs[language] = sdk
}
