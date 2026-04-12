package collectorconfig

import (
	"fmt"

	"github.com/go-logr/logr"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

const (
	filelogReceiverName                  = "filelog"
	logsPipelineName                     = "logs"
	odigosLogsResourceAttrsProcessorName = "odigoslogsresourceattrsprocessor"
)

func getReceivers(logger logr.Logger, sources *odigosv1.InstrumentationConfigList, odigosNamespace string) (config.GenericMap, []string) {
	includes := make([]string, 0)
	anyEbpf := false
	anyNonEbpf := false

	if sources != nil {
		for _, element := range sources.Items {
			if isSourceEbpfLogCaptureEnabled(&element) {
				anyEbpf = true
				continue
			}
			anyNonEbpf = true

			// Paths for log files: /var/log/pods/<namespace>_<pod name>_<pod ID>/<container name>/<auto-incremented file number>.log
			// Pod specifiers
			// 	Deployment:  <namespace>_<deployment  name>-<replicaset suffix[~10]>-<pod suffix[~5]>_<pod ID>
			// 	DeamonSet:   <namespace>_<daemonset   name>-<            pod suffix[~5]            >_<pod ID>
			// 	StatefulSet: <namespace>_<statefulset name>-<        ordinal index integer        >_<pod ID>
			// The suffixes are not the same lenght always, so we cannot match the pattern reliably.
			// We expect there to exactly one OwnerReference
			if len(element.OwnerReferences) != 1 {
				logger.Error(
					fmt.Errorf("unexpected number of OwnerReferences for instrumentation config %s/%s during logs configmap compilation: %d", element.Namespace, element.Name, len(element.OwnerReferences)),
					"failed to compile logs include list for configmap for instrumentation config",
				)
				continue
			}
			owner := element.OwnerReferences[0]
			name := owner.Name
			includes = append(includes, fmt.Sprintf("/var/log/pods/%s_%s-*_*/*/*.log", element.Namespace, name))
		}
	}

	switch {
	case anyEbpf && !anyNonEbpf:
		// eBPF receiver config lives in the common domain; no per-pipeline receiver config needed here.
		return config.GenericMap{}, []string{odigosEbpfReceiverName}
	case anyEbpf && anyNonEbpf:
		// Mixed mode: keep filelog for workloads without eBPF log capture, and also enable odigosebpf.
		return filelogReceiverConfig(includes, odigosNamespace), []string{filelogReceiverName, odigosEbpfReceiverName}
	default:
		return filelogReceiverConfig(includes, odigosNamespace), []string{filelogReceiverName}
	}
}

func filelogReceiverConfig(includes []string, odigosNamespace string) config.GenericMap {
	return config.GenericMap{
		filelogReceiverName: config.GenericMap{
			"include": includes,
			"exclude": []string{"/var/log/pods/kube-system_*/**/*", "/var/log/pods/" + odigosNamespace + "_*/**/*"},
			// 5s (vs upstream 200ms default) avoids a readdir storm from stanza's per-include glob loop on busy nodes.
			"poll_interval":     "5s",
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

func isSourceEbpfLogCaptureEnabled(ic *odigosv1.InstrumentationConfig) bool {
	for _, sdkConfig := range ic.Spec.SdkConfigs {
		if sdkConfig.EbpfLogCapture != nil &&
			sdkConfig.EbpfLogCapture.Enabled != nil &&
			*sdkConfig.EbpfLogCapture.Enabled {
			return true
		}
	}
	return false
}

func LogsConfig(logger logr.Logger, nodeCG *odigosv1.CollectorsGroup, odigosNamespace string, manifestProcessorNames []string, sources *odigosv1.InstrumentationConfigList) config.Config {

	pipelineProcessors := append([]string{
		memoryLimiterProcessorName,
		nodeNameProcessorName,
		resourceDetectionProcessorName,
		odigosLogsResourceAttrsProcessorName,
	}, manifestProcessorNames...)
	// append odigos traffic metrics processor last (after manifest processors)
	pipelineProcessors = append(pipelineProcessors, odigosTrafficMetricsProcessorName)

	receivers, pipelineReceivers := getReceivers(logger, sources, odigosNamespace)

	return config.Config{
		Receivers: receivers,
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				logsPipelineName: {
					Receivers:  pipelineReceivers,
					Processors: pipelineProcessors,
					Exporters:  []string{clusterCollectorLogsExporterName},
				},
			},
		},
	}
}
