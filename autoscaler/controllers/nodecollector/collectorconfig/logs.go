package collectorconfig

import (
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	filelogReceiverName = "filelog"
	logsPipelineName    = "logs"
)

// this code was used to enrich the logs with the workload names.
// commented out due to memory concerns for pulling all deplyments for the transformation.
// copied here during refactor for future reference

// err := updateOrCreateK8sAttributesForLogs(&cfg)
// if err != nil {
// 	return "", err
// }
// // remove logs processors from CRD logsProcessors in case it is there so not to add it twice
// for i, processor := range logsProcessors {
// 	if processor == k8sAttributesProcessorName {
// 		logsProcessors = append(logsProcessors[:i], logsProcessors[i+1:]...)
// 		break
// 	}
// }

// set "service.name" for logs same as the workload name.
// note: this does not respect the override service name a user can set in sources.
// cfg.Processors[logsServiceNameProcessorName] = config.GenericMap{
// 	"attributes": []config.GenericMap{
// 		{
// 			"key":            string(semconv.ServiceNameKey),
// 			"from_attribute": string(semconv.K8SDeploymentNameKey),
// 			"action":         "insert", // avoid overwriting existing value
// 		},
// 		{
// 			"key":            string(semconv.ServiceNameKey),
// 			"from_attribute": string(semconv.K8SStatefulSetNameKey),
// 			"action":         "insert", // avoid overwriting existing value
// 		},
// 		{
// 			"key":            string(semconv.ServiceNameKey),
// 			"from_attribute": string(semconv.K8SDaemonSetNameKey),
// 			"action":         "insert", // avoid overwriting existing value
// 		},
// 		{
// 			"key":            string(semconv.ServiceNameKey),
// 			"from_attribute": string(semconv.K8SCronJobNameKey),
// 			"action":         "insert", // avoid overwriting existing value
// 		},
// 	},
// }

// const (
// 	k8sAttributesProcessorName   = "k8sattributes/odigos-k8sattributes"
// 	logsServiceNameProcessorName = "resource/service-name"
// )

// using the k8sattributes processor to add deployment, statefulset and daemonset metadata
// has a high memory consumption in large clusters since it brings all the replicasets in the cluster into the cache.
// this has been disabled for logs, until further investigation is done on how to reduce memory consumption.
// The side effect is that logs record will lack deployment/statefulset/daemonset names and service name that could have been derived from them.
// func updateOrCreateK8sAttributesForLogs(cfg *config.Config) error {

// 	_, k8sProcessorExists := cfg.Processors[k8sAttributesProcessorName]

// 	if k8sProcessorExists {
// 		// make sure it includes the workload names attributes in the processor "extract" section.
// 		// this is added automatically for logs regardless of any action configuration.
// 		k8sAttributesCfg, ok := cfg.Processors[k8sAttributesProcessorName].(config.GenericMap)
// 		if !ok {
// 			return fmt.Errorf("failed to cast k8s attributes processor config to GenericMap")
// 		}
// 		if _, exists := k8sAttributesCfg["extract"]; !exists {
// 			k8sAttributesCfg["extract"] = config.GenericMap{}
// 		}
// 		extract, ok := k8sAttributesCfg["extract"].(map[string]interface{})
// 		if !ok {
// 			return fmt.Errorf("failed to cast k8s attributes processor extract config to GenericMap")
// 		}
// 		if _, exists := extract["metadata"]; !exists {
// 			extract["metadata"] = []interface{}{}
// 		}
// 		metadata, ok := extract["metadata"].([]interface{})
// 		if !ok {
// 			return fmt.Errorf("failed to cast k8s attributes processor metadata config to []interface{}")
// 		}
// 		// convert the metadata to a string array to compare it's values
// 		asStrArray := make([]string, len(metadata))
// 		for i, v := range metadata {
// 			asStrArray[i] = v.(string)
// 		}
// 		// add the workload names attributes to the metadata list
// 		if !slices.Contains(asStrArray, string(semconv.K8SDeploymentNameKey)) {
// 			metadata = append(metadata, string(semconv.K8SDeploymentNameKey))
// 		}
// 		if !slices.Contains(asStrArray, string(semconv.K8SStatefulSetNameKey)) {
// 			metadata = append(metadata, string(semconv.K8SStatefulSetNameKey))
// 		}
// 		if !slices.Contains(asStrArray, string(semconv.K8SDaemonSetNameKey)) {
// 			metadata = append(metadata, string(semconv.K8SDaemonSetNameKey))
// 		}
// 		extract["metadata"] = metadata                                // set the copy back to the config
// 		k8sAttributesCfg["extract"] = extract                         // set the copy back to the config
// 		cfg.Processors[k8sAttributesProcessorName] = k8sAttributesCfg // set the copy back to the config
// 	} else {
// 		// if the processor does not exist, create it with the default configuration
// 		cfg.Processors[k8sAttributesProcessorName] = config.GenericMap{
// 			"auth_type": "serviceAccount",
// 			"filter": config.GenericMap{
// 				"node_from_env_var": k8sconsts.NodeNameEnvVar,
// 			},
// 			"extract": config.GenericMap{
// 				"metadata": []string{
// 					string(semconv.K8SDeploymentNameKey),
// 					string(semconv.K8SStatefulSetNameKey),
// 					string(semconv.K8SDaemonSetNameKey),
// 				},
// 			},
// 			"pod_association": []config.GenericMap{
// 				{
// 					"sources": []config.GenericMap{
// 						{"from": "resource_attribute", "name": string(semconv.K8SPodNameKey)},
// 						{"from": "resource_attribute", "name": string(semconv.K8SNamespaceNameKey)},
// 					},
// 				},
// 			},
// 		}
// 	}

// 	return nil
// }

func getReceivers(sources *odigosv1.InstrumentationConfigList, odigosNamespace string) config.GenericMap {

	includes := make([]string, 0)
	for _, element := range sources.Items {
		// Paths for log files: /var/log/pods/<namespace>_<pod name>_<pod ID>/<container name>/<auto-incremented file number>.log
		// Pod specifiers
		// 	Deployment:  <namespace>_<deployment  name>-<replicaset suffix[~10]>-<pod suffix[~5]>_<pod ID>
		// 	DeamonSet:   <namespace>_<daemonset   name>-<            pod suffix[~5]            >_<pod ID>
		// 	StatefulSet: <namespace>_<statefulset name>-<        ordinal index integer        >_<pod ID>
		// The suffixes are not the same lenght always, so we cannot match the pattern reliably.
		// We expect there to exactly one OwnerReference
		if len(element.OwnerReferences) != 1 {
			log.Log.V(0).Error(
				fmt.Errorf("unexpected number of OwnerReferences for instrumentation config %s/%s during logs configmap compilation: %d", element.Namespace, element.Name, len(element.OwnerReferences)),
				"failed to compile logs include list for configmap for instrumentation config",
			)
			continue
		}
		owner := element.OwnerReferences[0]
		name := owner.Name
		includes = append(includes, fmt.Sprintf("/var/log/pods/%s_%s-*_*/*/*.log", element.Namespace, name))
	}

	return config.GenericMap{
		filelogReceiverName: config.GenericMap{
			"include":           includes,
			"exclude":           []string{"/var/log/pods/kube-system_*/**/*", "/var/log/pods/" + odigosNamespace + "_*/**/*"},
			"start_at":          "end",
			"include_file_path": true,
			"include_file_name": false,
			"operators": []config.GenericMap{
				{
					"id":   "container-parser",
					"type": "container",
				},
			},
			"retry_on_failure": config.GenericMap{
				// From documentation:
				// When true, the receiver will pause reading a file and attempt to resend the current batch of logs
				//  if it encounters an error from downstream components.
				//
				// filelog might get overwhelmed when it just starts and there are already a lot of logs to process in the node.
				// when downstream components (cluster collector and receiving destination) are under too much pressure,
				// they will reject the data, slowing down the filelog receiver and allowing it to retry the data and adjust to
				// downstream pressure.
				"enabled": true,
			},
		},
	}
}

func LogsConfig(nodeCG *odigosv1.CollectorsGroup, odigosNamespace string, manifestProcessorNames []string, sources *odigosv1.InstrumentationConfigList) config.Config {

	pipelineProcessors := append([]string{
		memoryLimiterProcessorName,
		nodeNameProcessorName,
		resourceDetectionProcessorName,
	}, manifestProcessorNames...)
	// append odigos traffic metrics processor last (after manifest processors)
	pipelineProcessors = append(pipelineProcessors, odigosTrafficMetricsProcessorName)

	return config.Config{
		Receivers: getReceivers(sources, odigosNamespace),
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				logsPipelineName: {
					Receivers:  []string{filelogReceiverName},
					Processors: pipelineProcessors,
					Exporters:  []string{clusterCollectorLogsExporterName},
				},
			},
		},
	}
}
