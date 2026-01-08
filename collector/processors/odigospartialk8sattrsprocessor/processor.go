package odigospartialk8sattrsprocessor

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"github.com/odigos-io/odigos/collector/processor/odigospartialk8sattrsprocessor/internal/kube"
)

type partialK8sAttrsProcessor struct {
	podMetadataClient kube.Client
}

func newPartialK8sAttrsProcessor() *partialK8sAttrsProcessor {
	return &partialK8sAttrsProcessor{}
}

func (p *partialK8sAttrsProcessor) Start(ctx context.Context, _ component.Host) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	client, err := kube.NewMetadataClient(config)
	if err != nil {
		return fmt.Errorf("failed to create pod metadata client: %w", err)
	}
	p.podMetadataClient = client

	return p.podMetadataClient.Start(ctx)
}

// workloadKindToSemconvKey maps Kubernetes workload kinds to their attribute keys.
// Note: Argo Rollout uses a custom attribute since there's no semconv key for it.
var workloadKindToSemconvKey = map[kube.WorkloadKind]string{
	kube.WorkloadKindDeployment:  string(semconv.K8SDeploymentNameKey),
	kube.WorkloadKindDaemonSet:   string(semconv.K8SDaemonSetNameKey),
	kube.WorkloadKindStatefulSet: string(semconv.K8SStatefulSetNameKey),
	kube.WorkloadKindJob:         string(semconv.K8SJobNameKey),
	kube.WorkloadKindCronJob:     string(semconv.K8SCronJobNameKey),
	kube.WorkloadKindArgoRollout: kube.K8SArgoRolloutNameAttribute,
}

func (processor *partialK8sAttrsProcessor) processResource(resource pcommon.Resource) {
	attrs := resource.Attributes()
	podUIDVal, isPodUIDExist := attrs.Get(string(semconv.K8SPodUIDKey))
	if !isPodUIDExist {
		return
	}

	podUID := podUIDVal.AsString()

	// Look up pod metadata from the cache (populated by the informer)
	podMeta, found := processor.podMetadataClient.GetPodMetadata(types.UID(podUID))
	if !found {
		// Pod not in cache yet - this can happen briefly after pod creation
		return
	}

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
		attrs.PutStr(string(semconv.ServiceNamespaceKey), podMeta.Namespace)
	}
}

// processLogs is the method that processes log data
func (p *partialK8sAttrsProcessor) processLogs(_ context.Context, logs plog.Logs) (plog.Logs, error) {
	allResourceLogs := logs.ResourceLogs()
	for i := 0; i < allResourceLogs.Len(); i++ {
		resourceLog := allResourceLogs.At(i)
		p.processResource(resourceLog.Resource())
	}
	return logs, nil
}
