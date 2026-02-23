package collectorconfig

import (
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	filelogReceiverName                  = "filelog"
	logsPipelineName                     = "logs"
	odigosLogsResourceAttrsProcessorName = "odigoslogsresourceattrsprocessor"
)

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
		odigosLogsResourceAttrsProcessorName,
	}, manifestProcessorNames...)
	// append odigos traffic metrics processor last (after manifest processors)
	pipelineProcessors = append(pipelineProcessors, odigosTrafficMetricsProcessorName)

	return config.Config{
		Receivers: getReceivers(sources, odigosNamespace),
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				logsPipelineName: {
					Receivers:  []string{filelogReceiverName, odigosEbpfReceiverName},
					Processors: pipelineProcessors,
					Exporters:  []string{clusterCollectorLogsExporterName},
				},
			},
		},
	}
}
