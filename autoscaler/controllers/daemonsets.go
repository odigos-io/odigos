package controllers

//import (
//	"context"
//	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
//	"github.com/keyval-dev/odigos/common"
//	v1 "k8s.io/api/apps/v1"
//	corev1 "k8s.io/api/core/v1"
//	apierrors "k8s.io/apimachinery/pkg/api/errors"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	ctrl "sigs.k8s.io/controller-runtime"
//	"sigs.k8s.io/controller-runtime/pkg/client"
//	"strings"
//)
//
//func (r *CollectorReconciler) syncDaemonSets(ctx context.Context, collector *odigosv1.Collector) (bool, error) {
//	dsList, err := r.listDaemonSets(ctx, collector)
//	if err != nil {
//		return false, err
//	}
//
//	if len(dsList.Items) == 0 {
//		err = r.createDaemonSets(ctx, collector)
//		if err != nil {
//			return false, err
//		}
//		return true, nil
//	}
//
//	return false, nil
//}
//
//func (r *CollectorReconciler) createDaemonSets(ctx context.Context, collector *odigosv1.Collector) error {
//	destList, err := r.listDestinations(ctx, collector)
//	if err != nil {
//		return err
//	}
//
//	img := r.getCollectorContainerImage(ctx, collector, destList)
//	cmd := "/otelcol"
//	if strings.Contains(img, "contrib") {
//		cmd = "/otelcol-contrib"
//	}
//
//	ds := &v1.DaemonSet{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      collector.Name,
//			Namespace: collector.Namespace,
//			Labels:    commonLabels,
//		},
//		Spec: v1.DaemonSetSpec{
//			Selector: &metav1.LabelSelector{
//				MatchLabels: commonLabels,
//			},
//			Template: corev1.PodTemplateSpec{
//				ObjectMeta: metav1.ObjectMeta{
//					Labels: commonLabels,
//				},
//				Spec: corev1.PodSpec{
//					Volumes: []corev1.Volume{
//						{
//							Name: "collector-conf",
//							VolumeSource: corev1.VolumeSource{
//								ConfigMap: &corev1.ConfigMapVolumeSource{
//									LocalObjectReference: corev1.LocalObjectReference{
//										Name: collector.Name,
//									},
//									Items: []corev1.KeyToPath{
//										{
//											Key:  "collector-conf",
//											Path: "collector-conf.yaml",
//										},
//									},
//								},
//							},
//						},
//						{
//							Name: "varlog",
//							VolumeSource: corev1.VolumeSource{
//								HostPath: &corev1.HostPathVolumeSource{
//									Path: "/var/log",
//								},
//							},
//						},
//					},
//					Containers: []corev1.Container{
//						{
//							Name:    "collector",
//							Image:   img,
//							Command: []string{cmd, "--config=/conf/collector-conf.yaml"},
//							EnvFrom: r.getSecretsFromDests(destList),
//							VolumeMounts: []corev1.VolumeMount{
//								{
//									Name:      "collector-conf",
//									MountPath: "/conf",
//								},
//								{
//									Name:      "varlog",
//									MountPath: "/var/log",
//									ReadOnly:  true,
//								},
//							},
//						},
//					},
//					HostNetwork: true,
//					DNSPolicy:   corev1.DNSClusterFirstWithHostNet,
//				},
//			},
//		},
//	}
//
//	err = ctrl.SetControllerReference(collector, ds, r.Scheme)
//	if err != nil {
//		return err
//	}
//
//	err = r.Create(ctx, ds)
//	if err != nil {
//		if apierrors.IsAlreadyExists(err) {
//			return nil
//		}
//		return err
//	}
//
//	return nil
//}
//
//func (r *CollectorReconciler) listDaemonSets(ctx context.Context, collector *odigosv1.Collector) (*v1.DaemonSetList, error) {
//	var dsList v1.DaemonSetList
//	err := r.List(ctx, &dsList, client.InNamespace(collector.Namespace), client.MatchingFields{ownerKey: collector.Name})
//	if err != nil {
//		return nil, err
//	}
//
//	return &dsList, nil
//}
//
//func (r *CollectorReconciler) getSecretsFromDests(destList *odigosv1.DestinationList) []corev1.EnvFromSource {
//	var result []corev1.EnvFromSource
//	for _, dst := range destList.Items {
//		result = append(result, corev1.EnvFromSource{
//			SecretRef: &corev1.SecretEnvSource{
//				LocalObjectReference: corev1.LocalObjectReference{
//					Name: dst.Spec.SecretRef.Name,
//				},
//			},
//		})
//	}
//
//	return result
//}
//
//func (r *CollectorReconciler) getCollectorContainerImage(ctx context.Context, collector *odigosv1.Collector, destList *odigosv1.DestinationList) string {
//	// TODO: Use more minimal image
//	contribImage := "otel/opentelemetry-collector-contrib:0.55.0"
//	regularImage := "otel/opentelemetry-collector:0.55.0"
//
//	for _, dst := range destList.Items {
//		if dst.Spec.Type == odigosv1.DatadogDestinationType {
//			return contribImage
//		}
//
//		for _, signal := range dst.Spec.Signals {
//			if signal == common.LogsObservabilitySignal {
//				return contribImage
//			}
//		}
//	}
//
//	return regularImage
//}
