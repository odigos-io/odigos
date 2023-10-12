package resources

import (
	"fmt"

	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common" // TODO: move it to neutral place
	"github.com/keyval-dev/odigos/cli/pkg/labels"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ownTelemetryOtelConfig                    = "odigos-own-telemetry-otel-config"
	ownTelemetryCollectorConfig               = "odigos-own-telemetry-collector-config"
	ownTelemetryCollectorConfigKeyName        = "odigos-own-telemetry-otelcol-config"
	ownTelemetryCollectorPodVolumeName        = "odigos-own-telemetry-otelcol-config"
	ownTelemetryCollectorImage                = "otel/opentelemetry-collector:0.86.0"
	ownTelemetryCollectorAppName              = "own-telemetry-collector"
	ownTelemetryCollectorServiceName          = "own-telemetry-collector"
	ownTelemetryCollectorDeploymentName       = "own-telemetry-collector"
	ownTelemetryCollectorContainerName        = "own-telemetry-collector"
	ownTelemetryCollectorConfigDir            = "/etc/otelcol" // since we use otel/opentelemetry-collector which expect the image at this path
	ownTelemetryCollectorConfigConfigFileName = "config.yaml"  // since we use otel/opentelemetry-collector which expect the config file to be called this way
	ownTelemetryOdigosCloudCollectorHost      = "odigos-cloud-col.keyval.dev"
	odigosCloudCollectorEnvName               = "ODIGOS_CLOUD_COL_HOST"
)

// used for odigos opensource which does not collect own telemetry
func NewOwnTelemetryConfigMapDisabled() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: v1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: ownTelemetryOtelConfig,
		},
		Data: map[string]string{
			"OTEL_SDK_DISABLED": "true",
		},
	}
}

// for odigos cloud which process own telemetry
func NewOwnTelemetryConfigMapOtlpGrpc(ns string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: v1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: ownTelemetryOtelConfig,
		},
		Data: map[string]string{
			"OTEL_EXPORTER_OTLP_PROTOCOL": "grpc",
			"OTEL_EXPORTER_OTLP_INSECURE": "true",
			// the http:// scheme is not actually used, it how the exporter is expecting the value with grpc
			"OTEL_EXPORTER_OTLP_ENDPOINT": fmt.Sprintf("http://%s.%s:4317", ownTelemetryCollectorServiceName, ns),
		},
	}
}

func getOtelcolConfigMapValue() string {
	empty := struct{}{}
	cfg := commonconf.Config{
		Receivers: commonconf.GenericMap{
			"otlp": commonconf.GenericMap{
				"protocols": commonconf.GenericMap{
					"grpc": empty,
					"http": empty,
				},
			},
		},
		Processors: commonconf.GenericMap{
			"batch": empty,
		},
		Exporters: commonconf.GenericMap{
			"otlp": commonconf.GenericMap{
				"endpoint": "${env:ODIGOS_CLOUD_COL_HOST}:4317",
				"headers": commonconf.GenericMap{
					"authorization": "Bearer ${ODIGOS_CLOUD_TOKEN}",
				},
			},
		},
		Extensions: commonconf.GenericMap{
			"health_check": empty,
			"zpages":       empty,
		},
		Service: commonconf.Service{
			Pipelines: map[string]commonconf.Pipeline{
				"logs": commonconf.Pipeline{
					Receivers:  []string{"otlp"},
					Processors: []string{"batch"},
					Exporters:  []string{"otlp"},
				},
			},
			Extensions: []string{"health_check", "zpages"},
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return ""
	}

	return string(data)
}

func NewOwnTelemetryCollectorConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: v1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: ownTelemetryCollectorConfig,
		},
		Data: map[string]string{
			ownTelemetryCollectorConfigKeyName: getOtelcolConfigMapValue(),
		},
	}
}

func NewOwnTelemetryCollectorDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ownTelemetryCollectorDeploymentName,
			Labels: map[string]string{
				"app":                       ownTelemetryCollectorAppName,
				labels.OdigosSystemLabelKey: labels.OdigosSystemLabelValue,
			},
			Annotations: map[string]string{
				"odigos.io/skip": "true",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: intPtr(1),
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"app": ownTelemetryCollectorAppName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"app": ownTelemetryCollectorAppName,
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: ownTelemetryCollectorPodVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: ownTelemetryCollectorConfig,
									},
									Items: []corev1.KeyToPath{
										{
											Key:  ownTelemetryCollectorConfigKeyName,
											Path: ownTelemetryCollectorConfigConfigFileName,
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  ownTelemetryCollectorContainerName,
							Image: ownTelemetryCollectorImage,
							Ports: []corev1.ContainerPort{{ContainerPort: 4317}, {ContainerPort: 4318}},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      ownTelemetryCollectorPodVolumeName,
									MountPath: ownTelemetryCollectorConfigDir,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  odigosCloudCollectorEnvName,
									Value: ownTelemetryOdigosCloudCollectorHost,
								},
								{
									Name: odigosCloudTokenEnvName,
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: odigosCloudSecretName,
											},
											Key: odigosCloudApiKeySecretKey,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func NewOwnTelemetryCollectorService() *corev1.Service {
	return &corev1.Service{
		TypeMeta: v1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: ownTelemetryCollectorServiceName,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": ownTelemetryCollectorAppName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "otlpgrpc",
					Protocol:   corev1.ProtocolTCP,
					Port:       4317,
					TargetPort: intstr.FromInt(4317),
				},
				{
					Name:       "otlphttp",
					Protocol:   corev1.ProtocolTCP,
					Port:       4318,
					TargetPort: intstr.FromInt(4318),
				},
			},
		},
	}
}

func intPtr(n int32) *int32 {
	return &n
}

func int64Ptr(n int64) *int64 {
	return &n
}
