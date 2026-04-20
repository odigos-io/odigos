package watchers

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
	"github.com/odigos-io/odigos/frontend/services/common"
	"github.com/odigos-io/odigos/frontend/services/sse"
	commonutils "github.com/odigos-io/odigos/k8sutils/pkg/workload"
	toolscache "k8s.io/client-go/tools/cache"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
)

var instrumentationConfigAddedEventBatcher *EventBatcher
var instrumentationConfigModifiedEventBatcher *EventBatcher
var instrumentationConfigDeletedEventBatcher *EventBatcher

func StartInstrumentationConfigWatcher(ctx context.Context, k8sCache ctrlcache.Cache, metricsConsumer *collectormetrics.OdigosMetricsConsumer) error {
	instrumentationConfigAddedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 1,
			Duration:     3 * time.Second,
			Event:        sse.MessageEventAdded,
			CRDType:      consts.InstrumentationConfig,
			MaxBatchSize: 100,
			MaxDelay:     10 * time.Second,
			SuccessBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Successfully created %d sources", count)
			},
			FailureBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Failed to create %d sources", count)
			},
		},
	)

	instrumentationConfigModifiedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 1,
			Duration:     3 * time.Second,
			Event:        sse.MessageEventModified,
			CRDType:      consts.InstrumentationConfig,
			MaxBatchSize: 100,
			MaxDelay:     10 * time.Second,
			SuccessBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Successfully updated %d sources", count)
			},
			FailureBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Failed to update %d sources", count)
			},
		},
	)

	instrumentationConfigDeletedEventBatcher = NewEventBatcher(
		EventBatcherConfig{
			MinBatchSize: 1,
			Duration:     3 * time.Second,
			Event:        sse.MessageEventDeleted,
			CRDType:      consts.InstrumentationConfig,
			MaxBatchSize: 100,
			MaxDelay:     10 * time.Second,
			SuccessBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Successfully deleted %d sources", count)
			},
			FailureBatchMessageFunc: func(count int, crdType string) string {
				return fmt.Sprintf("Failed to delete %d sources", count)
			},
		},
	)

	informer, err := k8sCache.GetInformer(ctx, &v1alpha1.InstrumentationConfig{})
	if err != nil {
		return fmt.Errorf("failed to get InstrumentationConfig informer: %w", err)
	}

	_, err = informer.AddEventHandler(toolscache.ResourceEventHandlerDetailedFuncs{
		AddFunc: func(obj interface{}, isInInitialList bool) {
			ic, ok := obj.(*v1alpha1.InstrumentationConfig)
			if !ok {
				return
			}

			pw, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(ic.Name, ic.Namespace)
			if err != nil {
				if !isInInitialList {
					genericErrorMessage(sse.MessageEventAdded, consts.InstrumentationConfig, err.Error())
				}
				return
			}

			metricsConsumer.NotifySourceAdded(common.SourceID{
				Namespace: pw.Namespace, Name: pw.Name, Kind: pw.Kind,
			})

			if !isInInitialList {
				handleAddedInstrumentationConfig(ic)
			}
		},
		UpdateFunc: func(_, newObj interface{}) {
			ic, ok := newObj.(*v1alpha1.InstrumentationConfig)
			if !ok {
				return
			}
			handleModifiedInstrumentationConfig(ic)
		},
		DeleteFunc: func(obj interface{}) {
			ic, ok := obj.(*v1alpha1.InstrumentationConfig)
			if !ok {
				tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown)
				if !ok {
					return
				}
				ic, ok = tombstone.Obj.(*v1alpha1.InstrumentationConfig)
				if !ok {
					return
				}
			}

			pw, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(ic.Name, ic.Namespace)
			if err != nil {
				genericErrorMessage(sse.MessageEventDeleted, consts.InstrumentationConfig, err.Error())
				return
			}

			metricsConsumer.NotifySourceDeleted(common.SourceID{
				Namespace: pw.Namespace, Name: pw.Name, Kind: pw.Kind,
			})

			handleDeletedInstrumentationConfig(ic)
		},
	})

	return err
}

func handleAddedInstrumentationConfig(instruConfig *v1alpha1.InstrumentationConfig) {
	pw, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(instruConfig.Name, instruConfig.Namespace)
	if err != nil {
		genericErrorMessage(sse.MessageEventAdded, consts.InstrumentationConfig, err.Error())
		return
	}

	target := fmt.Sprintf("namespace=%s&name=%s&kind=%s", pw.Namespace, pw.Name, pw.Kind)
	data := fmt.Sprintf(`Successfully created "%s" source`, pw.Name)
	instrumentationConfigAddedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}

func handleModifiedInstrumentationConfig(instruConfig *v1alpha1.InstrumentationConfig) {
	pw, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(instruConfig.Name, instruConfig.Namespace)
	if err != nil {
		genericErrorMessage(sse.MessageEventModified, consts.InstrumentationConfig, err.Error())
		return
	}

	target := fmt.Sprintf("namespace=%s&name=%s&kind=%s", pw.Namespace, pw.Name, pw.Kind)
	data := fmt.Sprintf(`Successfully updated "%s" source`, pw.Name)
	instrumentationConfigModifiedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}

func handleDeletedInstrumentationConfig(instruConfig *v1alpha1.InstrumentationConfig) {
	pw, err := commonutils.ExtractWorkloadInfoFromRuntimeObjectName(instruConfig.Name, instruConfig.Namespace)
	if err != nil {
		genericErrorMessage(sse.MessageEventDeleted, consts.InstrumentationConfig, err.Error())
		return
	}

	target := fmt.Sprintf("namespace=%s&name=%s&kind=%s", pw.Namespace, pw.Name, pw.Kind)
	data := fmt.Sprintf(`Successfully deleted "%s" source`, pw.Name)
	instrumentationConfigDeletedEventBatcher.AddEvent(sse.MessageTypeSuccess, data, target)
}
