package custom

import (
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	honeycombDataCollectionImage = "honeycombio/honeycomb-kubernetes-agent:2.6.0"
	honeycombConfigMountPath     = "/etc/honeycomb"
	honeycombConfigKey           = "honeycomb-conf"
	honeycombEndpoint            = "HONEYCOMB_ENDPOINT"
)

func addHoneycombConfig(cm *corev1.ConfigMap, dst odigosv1.Destination) {
	template := `    apiHost: %s
    watchers:
      - dataset: kubernetes-logs
        labelSelector: "component=kube-apiserver,tier=control-plane"
        namespace: kube-system
        parser: glog
      - dataset: kubernetes-logs
        labelSelector: "component=kube-scheduler,tier=control-plane"
        namespace: kube-system
        parser: glog
      - dataset: kubernetes-logs
        labelSelector: "component=kube-controller-manager,tier=control-plane"
        namespace: kube-system
        parser: glog
      - dataset: kubernetes-logs
        labelSelector: "k8s-app=kube-proxy"
        namespace: kube-system
        parser: glog
      - dataset: kubernetes-logs
        labelSelector: "k8s-app=kube-dns"
        namespace: kube-system
        parser: glog
    verbosity: info
    splitLogging: false
    metrics:
      clusterName: k8s-cluster
      dataset: kubernetes-metrics
      enabled: true
      metricGroups:
      - node
      - pod`
	cm.Data[honeycombConfigKey] = fmt.Sprintf(template, dst.Spec.Data[honeycombEndpoint])
}

func addHoneycombToDaemonSet(ds *v1.DaemonSet, secretName string) {
	ds.Spec.Template.Spec.Containers = append(ds.Spec.Template.Spec.Containers, corev1.Container{
		Name:  "honeycomb-collector",
		Image: honeycombDataCollectionImage,
		Env: []corev1.EnvVar{
			{
				Name: "NODE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
			{
				Name: "NODE_IP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			},
			{
				Name: "HONEYCOMB_APIKEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretName,
						},
						Key: "HONEYCOMB_API_KEY",
					},
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "varlibdockercontainers",
				MountPath: "/var/lib/docker/containers",
				ReadOnly:  true,
			},
			{
				Name:      "varlog",
				MountPath: "/var/log",
				ReadOnly:  true,
			},
			{
				Name:      "conf",
				MountPath: honeycombConfigMountPath,
				ReadOnly:  false,
			},
		},
	})

	for i, vol := range ds.Spec.Template.Spec.Volumes {
		if vol.Name == "conf" {
			ds.Spec.Template.Spec.Volumes[i].ConfigMap.Items = append(ds.Spec.Template.Spec.Volumes[i].ConfigMap.Items, corev1.KeyToPath{
				Key:  honeycombConfigKey,
				Path: "config.yaml",
			})
		}
	}
}
