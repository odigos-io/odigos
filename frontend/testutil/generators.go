package testutil

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func GenerateNamespaces(count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := range count {
		objs[i] = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("ns-%d", i)},
		}
	}
	return objs
}

func GenerateNamespaceSources(count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := range count {
		nsName := fmt.Sprintf("ns-%d", i)
		objs[i] = &odigosv1alpha1.Source{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("source-ns-%d", i),
				Namespace: nsName,
				Labels: map[string]string{
					k8sconsts.WorkloadNamespaceLabel: nsName,
					k8sconsts.WorkloadNameLabel:      nsName,
					k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindNamespace),
				},
			},
			Spec: odigosv1alpha1.SourceSpec{
				Workload: k8sconsts.PodWorkload{
					Name:      nsName,
					Namespace: nsName,
					Kind:      k8sconsts.WorkloadKindNamespace,
				},
			},
		}
	}
	return objs
}

func GenerateDeployments(namespace string, count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := range count {
		objs[i] = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("deploy-%d", i),
				Namespace: namespace,
			},
			Status: appsv1.DeploymentStatus{ReadyReplicas: 2},
		}
	}
	return objs
}

func GenerateStatefulSets(namespace string, count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := range count {
		objs[i] = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("sts-%d", i),
				Namespace: namespace,
			},
			Status: appsv1.StatefulSetStatus{ReadyReplicas: 1},
		}
	}
	return objs
}

func GenerateDaemonSets(namespace string, count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := range count {
		objs[i] = &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("ds-%d", i),
				Namespace: namespace,
			},
			Status: appsv1.DaemonSetStatus{NumberReady: 3},
		}
	}
	return objs
}

func GenerateCronJobs(namespace string, count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := range count {
		objs[i] = &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("cj-%d", i),
				Namespace: namespace,
			},
		}
	}
	return objs
}

func GenerateSources(namespace string, count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := range count {
		objs[i] = &odigosv1alpha1.Source{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("source-%d", i),
				Namespace: namespace,
				Labels: map[string]string{
					k8sconsts.WorkloadNamespaceLabel: namespace,
					k8sconsts.WorkloadNameLabel:      fmt.Sprintf("deploy-%d", i),
					k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindDeployment),
				},
			},
			Spec: odigosv1alpha1.SourceSpec{
				Workload: k8sconsts.PodWorkload{
					Name:      fmt.Sprintf("deploy-%d", i),
					Namespace: namespace,
					Kind:      k8sconsts.WorkloadKindDeployment,
				},
			},
		}
	}
	return objs
}

func GenerateDestinationsAndSecrets(ns string, count int) (odigosObjs []runtime.Object, k8sObjs []runtime.Object) {
	for i := range count {
		secretName := fmt.Sprintf("dest-secret-%d", i)
		destName := fmt.Sprintf("dest-%d", i)

		k8sObjs = append(k8sObjs, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: ns,
			},
			Data: map[string][]byte{
				"api_key":  []byte(fmt.Sprintf("key-%d", i)),
				"endpoint": []byte(fmt.Sprintf("https://endpoint-%d.example.com", i)),
			},
		})

		odigosObjs = append(odigosObjs, &odigosv1alpha1.Destination{
			ObjectMeta: metav1.ObjectMeta{
				Name:      destName,
				Namespace: ns,
			},
			Spec: odigosv1alpha1.DestinationSpec{
				Type:            "jaeger",
				DestinationName: fmt.Sprintf("Dest %d", i),
				Data:            map[string]string{"host": "localhost"},
				SecretRef:       &corev1.LocalObjectReference{Name: secretName},
			},
		})
	}
	return
}

func GenerateInstrumentationConfigs(namespace string, count int) []runtime.Object {
	objs := make([]runtime.Object, count)
	for i := range count {
		objs[i] = &odigosv1alpha1.InstrumentationConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("ic-%d", i),
				Namespace: namespace,
			},
		}
	}
	return objs
}

func OdigosConfigMap(ns string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosEffectiveConfigName,
			Namespace: ns,
		},
		Data: map[string]string{
			consts.OdigosConfigurationFileName: "ignoredNamespaces: []",
		},
	}
}
