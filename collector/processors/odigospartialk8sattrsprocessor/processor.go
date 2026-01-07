package odigospartialk8sattrsprocessor

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"github.com/odigos-io/odigos/collector/processor/odigospartialk8sattrsprocessor/internal/kube"
)

type serviceNameProcesser struct {
	podMetadataClient kube.Client
}

func newServiceNameProcessor(config *rest.Config) (*serviceNameProcesser, error) {
	client, err := kube.NewMetadataClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pod metadata client: %w", err)
	}

	return &serviceNameProcesser{
		podMetadataClient: client,
	}, nil
}

func (p *serviceNameProcesser) start(ctx context.Context) error {
	stopCh := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(stopCh)
	}()

	return p.podMetadataClient.Start(stopCh)
}

// ownerKindToSemconvKey maps Kubernetes owner kinds to their semconv attribute keys
var ownerKindToSemconvKey = map[string]string{
	"ReplicaSet":  string(semconv.K8SReplicaSetNameKey),
	"DaemonSet":   string(semconv.K8SDaemonSetNameKey),
	"StatefulSet": string(semconv.K8SStatefulSetNameKey),
	"Job":         string(semconv.K8SJobNameKey),
	"CronJob":     string(semconv.K8SCronJobNameKey),
}

func (processor *serviceNameProcesser) processResource(resource pcommon.Resource) {
	attrs := resource.Attributes()
	podUIDVal, isPodUIDExist := attrs.Get("k8s.pod.uid")
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

	// Add pod name and namespace
	if podMeta.Name != "" {
		attrs.PutStr(string(semconv.K8SPodNameKey), podMeta.Name)
	}
	if podMeta.Namespace != "" {
		attrs.PutStr(string(semconv.K8SNamespaceNameKey), podMeta.Namespace)
	}

	// Set the owner resource name using the appropriate semconv key based on owner kind
	if podMeta.OwnerName != "" && podMeta.OwnerKind != "" {
		if semconvKey, ok := ownerKindToSemconvKey[podMeta.OwnerKind]; ok {
			attrs.PutStr(semconvKey, podMeta.OwnerName)
		}
	}

	// ServiceName is pre-computed when pod is added/updated in the cache
	if podMeta.ServiceName == "" {
		return
	}

	attrs.PutStr(string(semconv.ServiceNameKey), podMeta.ServiceName)
	attrs.PutStr(string(semconv.ServiceNamespaceKey), podMeta.Namespace)
}

// processLogs is the method that processes log data
func (p *serviceNameProcesser) processLogs(_ context.Context, logs plog.Logs) (plog.Logs, error) {
	allResourceLogs := logs.ResourceLogs()
	for i := 0; i < allResourceLogs.Len(); i++ {
		resourceLog := allResourceLogs.At(i)
		p.processResource(resourceLog.Resource())
	}
	return logs, nil
}
