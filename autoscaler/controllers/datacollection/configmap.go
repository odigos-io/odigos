package datacollection

import (
	"context"
	"fmt"
	"reflect"

	"github.com/odigos-io/odigos/autoscaler/controllers/datacollection/custom"

	"github.com/ghodss/yaml"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/utils"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	configKey = "conf"
)

func syncConfigMap(apps *odigosv1.InstrumentedApplicationList, dests *odigosv1.DestinationList, processors *odigosv1.ProcessorList,
	datacollection *odigosv1.CollectorsGroup, ctx context.Context,
	c client.Client, scheme *runtime.Scheme) (string, error) {
	logger := log.FromContext(ctx)
	desired, err := getDesiredConfigMap(apps, dests, processors, datacollection, scheme)
	desiredData := desired.Data[configKey]
	if err != nil {
		logger.Error(err, "failed to get desired config map")
		return "", err
	}

	existing := &v1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: datacollection.Namespace, Name: datacollection.Name}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(0).Info("creating config map")
			_, err := createConfigMap(desired, ctx, c)
			if err != nil {
				logger.Error(err, "failed to create config map")
				return "", err
			}
			return desiredData, nil
		} else {
			logger.Error(err, "failed to get config map")
			return "", err
		}
	}

	logger.V(0).Info("Patching config map")
	_, err = patchConfigMap(ctx, existing, desired, c)
	if err != nil {
		logger.Error(err, "failed to patch config map")
		return "", err
	}

	return desiredData, nil
}

func patchConfigMap(ctx context.Context, existing *v1.ConfigMap, desired *v1.ConfigMap, c client.Client) (*v1.ConfigMap, error) {
	if reflect.DeepEqual(existing.Data, desired.Data) &&
		reflect.DeepEqual(existing.ObjectMeta.OwnerReferences, desired.ObjectMeta.OwnerReferences) {
		log.FromContext(ctx).V(0).Info("Config maps already match")
		return existing, nil
	}
	updated := existing.DeepCopy()
	updated.Data = desired.Data
	updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences
	patch := client.MergeFrom(existing)
	if err := c.Patch(ctx, updated, patch); err != nil {
		return nil, err
	}

	return updated, nil
}

func createConfigMap(desired *v1.ConfigMap, ctx context.Context, c client.Client) (*v1.ConfigMap, error) {
	if err := c.Create(ctx, desired); err != nil {
		return nil, err
	}

	return desired, nil
}

func getDesiredConfigMap(apps *odigosv1.InstrumentedApplicationList, dests *odigosv1.DestinationList, processors *odigosv1.ProcessorList,
	datacollection *odigosv1.CollectorsGroup, scheme *runtime.Scheme) (*v1.ConfigMap, error) {
	cmData, err := getConfigMapData(apps, dests, processors)
	if err != nil {
		return nil, err
	}

	desired := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      datacollection.Name,
			Namespace: datacollection.Namespace,
		},
		Data: map[string]string{
			configKey: cmData,
		},
	}

	if custom.ShouldApplyCustomDataCollection(dests) {
		custom.AddCustomConfigMap(dests, &desired)
	}

	if err := ctrl.SetControllerReference(datacollection, &desired, scheme); err != nil {
		return nil, err
	}

	return &desired, nil
}

func getConfigMapData(apps *odigosv1.InstrumentedApplicationList, dests *odigosv1.DestinationList, processors *odigosv1.ProcessorList) (string, error) {

	empty := struct{}{}

	processorsCfg, tracesProcessors, metricsProcessors, logsProcessors := commonconf.GetCrdProcessorsConfigMap(processors, odigosv1.CollectorsGroupRoleNodeCollector)
	processorsCfg["batch"] = empty
	processorsCfg["odigosresourcename"] = empty
	processorsCfg["resource"] = commonconf.GenericMap{
		"attributes": []commonconf.GenericMap{{
			"key":    "k8s.node.name",
			"value":  "${NODE_NAME}",
			"action": "upsert",
		}},
	}
	processorsCfg["resourcedetection"] = commonconf.GenericMap{"detectors": []string{"ec2", "gcp", "azure"}}

	cfg := commonconf.Config{
		Receivers: commonconf.GenericMap{
			"zipkin": empty,
			"otlp": commonconf.GenericMap{
				"protocols": commonconf.GenericMap{
					"grpc": empty,
					"http": empty,
				},
			},
		},
		Exporters: commonconf.GenericMap{
			"otlp/gateway": commonconf.GenericMap{
				"endpoint": fmt.Sprintf("odigos-gateway.%s:4317", utils.GetCurrentNamespace()),
				"tls": commonconf.GenericMap{
					"insecure": true,
				},
			},
		},
		Processors: processorsCfg,
		Extensions: commonconf.GenericMap{
			"health_check": empty,
			"zpages":       empty,
		},
		Service: commonconf.Service{
			Pipelines:  map[string]commonconf.Pipeline{},
			Extensions: []string{"health_check", "zpages"},
		},
	}

	collectTraces := false
	collectMetrics := false
	collectLogs := false
	for _, dst := range dests.Items {
		for _, s := range dst.Spec.Signals {
			if s == common.LogsObservabilitySignal && !custom.DestRequiresCustom(dst.Spec.Type) {
				collectLogs = true
			}
			if s == common.TracesObservabilitySignal || dst.Spec.Type == common.PrometheusDestinationType {
				collectTraces = true
			}
			if s == common.MetricsObservabilitySignal && !custom.DestRequiresCustom(dst.Spec.Type) {
				collectMetrics = true
			}
		}
	}

	if collectLogs {
		includes := make([]string, 0)
		for _, element := range apps.Items {
			// Paths for log files: /var/log/pods/<namespace>_<pod name>_<pod ID>/<container name>/<auto-incremented file number>.log
			// Pod specifiers
			// 	Deployment:  <namespace>_<deployment  name>-<replicaset suffix[~10]>-<pod suffix[~5]>_<pod ID>
			// 	DeamonSet:   <namespace>_<daemonset   name>-<            pod suffix[~5]            >_<pod ID>
			// 	StatefulSet: <namespace>_<statefulset name>-<        ordinal index integer        >_<pod ID>
			// The suffixes are not the same lenght always, so we cannot match the pattern reliably.
			// We expect there to exactly one OwnerReference
			if len(element.OwnerReferences) != 1 {
				log.Log.V(0).Error(
					fmt.Errorf("Unexpected number of OwnerReferences: %d", len(element.OwnerReferences)),
					"failed to compile include list for configmap",
				)
				continue
			}
			owner := element.OwnerReferences[0]
			name := owner.Name
			includes = append(includes, fmt.Sprintf("/var/log/pods/%s_%s-*_*/*/*.log", element.Namespace, name))
		}

		odigosSystemNamespaceName := utils.GetCurrentNamespace()
		cfg.Receivers["filelog"] = commonconf.GenericMap{
			"include":           includes,
			"exclude":           []string{"/var/log/pods/kube-system_*/**/*", "/var/log/pods/" + odigosSystemNamespaceName + "_*/**/*"},
			"start_at":          "beginning",
			"include_file_path": true,
			"include_file_name": false,
			"operators": []commonconf.GenericMap{
				{
					"type": "router",
					"id":   "get-format",
					"routes": []commonconf.GenericMap{
						{
							"output": "parser-docker",
							"expr":   `body matches "^\\{"`,
						},
						{
							"output": "parser-crio",
							"expr":   `body matches "^[^ Z]+ "`,
						},
						{
							"output": "parser-containerd",
							"expr":   `body matches "^[^ Z]+Z"`,
						},
					},
				},
				{
					"type":   "regex_parser",
					"id":     "parser-crio",
					"regex":  `^(?P<time>[^ Z]+) (?P<stream>stdout|stderr) (?P<logtag>[^ ]*) ?(?P<log>.*)$`,
					"output": "extract_metadata_from_filepath",
					"timestamp": commonconf.GenericMap{
						"parse_from":  "attributes.time",
						"layout_type": "gotime",
						"layout":      "2006-01-02T15:04:05.999999999Z07:00",
					},
				},
				{
					"type":   "regex_parser",
					"id":     "parser-containerd",
					"regex":  `^(?P<time>[^ ^Z]+Z) (?P<stream>stdout|stderr) (?P<logtag>[^ ]*) ?(?P<log>.*)$`,
					"output": "extract_metadata_from_filepath",
					"timestamp": commonconf.GenericMap{
						"parse_from": "attributes.time",
						"layout":     "%Y-%m-%dT%H:%M:%S.%LZ",
					},
				},
				{
					"type":   "json_parser",
					"id":     "parser-docker",
					"output": "extract_metadata_from_filepath",
					"timestamp": commonconf.GenericMap{
						"parse_from": "attributes.time",
						"layout":     "%Y-%m-%dT%H:%M:%S.%LZ",
					},
				},
				{
					"type": "move",
					"from": "attributes.log",
					"to":   "body",
				},
				{
					"type":       "regex_parser",
					"id":         "extract_metadata_from_filepath",
					"regex":      `^.*\/(?P<namespace>[^_]+)_(?P<pod_name>[^_]+)_(?P<uid>[a-f0-9\-]{36})\/(?P<container_name>[^\._]+)\/(?P<restart_count>\d+)\.log$`,
					"parse_from": `attributes["log.file.path"]`,
				},
				{
					"type": "move",
					"from": "attributes.stream",
					"to":   `attributes["log.iostream"]`,
				},
				{
					"type": "move",
					"from": "attributes.container_name",
					"to":   `attributes["k8s.container.name"]`,
				},
				{
					"type": "move",
					"from": "attributes.namespace",
					"to":   `attributes["k8s.namespace.name"]`,
				},
				{
					"type": "move",
					"from": "attributes.pod_name",
					"to":   `attributes["k8s.pod.name"]`,
				},
				{
					"type": "move",
					"from": "attributes.restart_count",
					"to":   `attributes["k8s.container.restart_count"]`,
				},
				{
					"type": "move",
					"from": "attributes.uid",
					"to":   `attributes["k8s.pod.uid"]`,
				},
			},
		}

		cfg.Service.Pipelines["logs"] = commonconf.Pipeline{
			Receivers:  []string{"filelog"},
			Processors: append([]string{"batch", "odigosresourcename", "resource", "resourcedetection"}, logsProcessors...),
			Exporters:  []string{"otlp/gateway"},
		}
	}

	if collectTraces {
		cfg.Service.Pipelines["traces"] = commonconf.Pipeline{
			Receivers:  []string{"otlp", "zipkin"},
			Processors: append([]string{"batch", "odigosresourcename", "resource", "resourcedetection"}, tracesProcessors...),
			Exporters:  []string{"otlp/gateway"},
		}
	}

	if collectMetrics {
		cfg.Receivers["kubeletstats"] = commonconf.GenericMap{
			"auth_type":            "serviceAccount",
			"endpoint":             "https://${env:NODE_NAME}:10250",
			"insecure_skip_verify": true,
			"collection_interval":  "10s",
		}

		cfg.Service.Pipelines["metrics"] = commonconf.Pipeline{
			Receivers:  []string{"otlp", "kubeletstats"},
			Processors: append([]string{"batch", "odigosresourcename", "resource", "resourcedetection"}, metricsProcessors...),
			Exporters:  []string{"otlp/gateway"},
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
