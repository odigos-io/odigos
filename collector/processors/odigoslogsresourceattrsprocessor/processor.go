package odigoslogsresourceattrsprocessor

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"

	"github.com/odigos-io/odigos/collector/processor/odigoslogsresourceattrsprocessor/internal/kube"
)

const (
	// logFilePathAttribute is the attribute set by the filelog receiver when include_file_path is true
	logFilePathAttribute = "log.file.path"
)

type partialK8sAttrsProcessor struct {
	podMetadataClient kube.Client
	logger            *zap.Logger
}

func newPartialK8sAttrsProcessor(logger *zap.Logger) *partialK8sAttrsProcessor {
	return &partialK8sAttrsProcessor{
		logger: logger,
	}
}

func (p *partialK8sAttrsProcessor) Start(ctx context.Context, _ component.Host) error {
	p.logger.Info("Starting odigoslogsresourceattrsprocessor")
	client, err := newKubeClient()
	if err != nil {
		p.logger.Error("Failed to create kube client", zap.Error(err))
		return fmt.Errorf("failed to create kube client: %w", err)
	}
	p.podMetadataClient = client

	err = p.podMetadataClient.Start(ctx)
	if err != nil {
		p.logger.Error("Failed to start pod metadata client", zap.Error(err))
		return fmt.Errorf("failed to start pod metadata client: %w", err)
	}
	p.logger.Info("odigoslogsresourceattrsprocessor started successfully")
	return nil
}

// workloadKindToSemconvKey maps Kubernetes workload kinds to their attribute keys.
// Note: Argo Rollout uses a custom attribute since there's no semconv key for it.
var workloadKindToSemconvKey = map[kube.WorkloadKind]string{
	kube.WorkloadKindDeployment:       string(semconv.K8SDeploymentNameKey),
	kube.WorkloadKindDaemonSet:        string(semconv.K8SDaemonSetNameKey),
	kube.WorkloadKindStatefulSet:      string(semconv.K8SStatefulSetNameKey),
	kube.WorkloadKindJob:              string(semconv.K8SJobNameKey),
	kube.WorkloadKindCronJob:          string(semconv.K8SCronJobNameKey),
	kube.WorkloadKindDeploymentConfig: string(semconv.K8SDeploymentNameKey),
	kube.WorkloadKindArgoRollout:      kube.K8SArgoRolloutNameAttribute,
	kube.WorkloadKindStaticPod:        string(semconv.K8SPodNameKey),
}

// extractPodUIDFromFilePath extracts the pod UID from a Kubernetes log file path.
// Path format: /var/log/pods/{namespace}_{pod_name}_{uid}/{container_name}/{file}.log
func extractPodUIDFromFilePath(filePath string) string {
	segments := strings.Split(filePath, "/")

	for i, segment := range segments {
		if segment == "pods" && i+1 < len(segments) {
			podDirName := segments[i+1]
			// The pod directory format is: {namespace}_{pod_name}_{uid}
			// Find the last underscore - UID is everything after it
			lastUnderscoreIdx := strings.LastIndex(podDirName, "_")
			if lastUnderscoreIdx != -1 && lastUnderscoreIdx < len(podDirName)-1 {
				return podDirName[lastUnderscoreIdx+1:]
			}
			break
		}
	}
	return ""
}

// extractPodUIDFromLogRecords looks for log.file.path in log records and extracts the pod UID.
// Returns the first UID found, since all logs in a ResourceLogs should be from the same pod.
func extractPodUIDFromLogRecords(resourceLog plog.ResourceLogs) string {
	scopeLogs := resourceLog.ScopeLogs()
	for i := 0; i < scopeLogs.Len(); i++ {
		logRecords := scopeLogs.At(i).LogRecords()
		for j := 0; j < logRecords.Len(); j++ {
			if filePathVal, exists := logRecords.At(j).Attributes().Get(logFilePathAttribute); exists {
				if uid := extractPodUIDFromFilePath(filePathVal.AsString()); uid != "" {
					return uid
				}
			}
		}
	}
	return ""
}

func (processor *partialK8sAttrsProcessor) enrichResourceWithPodMetadata(attrs pcommon.Map, podUID string) {
	podMeta, found := processor.podMetadataClient.GetPodMetadata(types.UID(podUID))
	if !found {
		return
	}

	attrs.PutStr(string(semconv.K8SPodUIDKey), podUID)

	if podMeta.Name != "" {
		attrs.PutStr(string(semconv.K8SPodNameKey), podMeta.Name)
	}
	if podMeta.Namespace != "" {
		attrs.PutStr(string(semconv.K8SNamespaceNameKey), podMeta.Namespace)
	}

	if podMeta.WorkloadName != "" && podMeta.WorkloadKind != "" {
		if semconvKey, ok := workloadKindToSemconvKey[podMeta.WorkloadKind]; ok {
			attrs.PutStr(semconvKey, podMeta.WorkloadName)
		}

		attrs.PutStr(string(semconv.ServiceNameKey), podMeta.WorkloadName)
	}
}

// processLogs is the method that processes log data
func (p *partialK8sAttrsProcessor) processLogs(_ context.Context, logs plog.Logs) (plog.Logs, error) {
	allResourceLogs := logs.ResourceLogs()
	for i := 0; i < allResourceLogs.Len(); i++ {
		resourceLog := allResourceLogs.At(i)
		attrs := resourceLog.Resource().Attributes()

		// First check if k8s.pod.uid is already in resource attributes (e.g., from OTLP receiver)
		var podUID string
		if podUIDVal, exists := attrs.Get(string(semconv.K8SPodUIDKey)); exists {
			podUID = podUIDVal.AsString()
		} else {
			podUID = extractPodUIDFromLogRecords(resourceLog)
		}

		if podUID == "" {
			continue
		}

		p.enrichResourceWithPodMetadata(attrs, podUID)
	}
	return logs, nil
}
