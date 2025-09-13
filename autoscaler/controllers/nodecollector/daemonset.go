package nodecollector

import (
	"context"
	"sync"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/common"
	odigoscommon "github.com/odigos-io/odigos/common"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	NodeCollectorsLabels = map[string]string{
		k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector),
	}
)

type DelayManager struct {
	mu         sync.Mutex
	inProgress bool
}

// RunSyncDaemonSetWithDelayAndSkipNewCalls runs the function with the specified delay and skips new calls until the function execution is finished
func (dm *DelayManager) RunSyncDaemonSetWithDelayAndSkipNewCalls(delay time.Duration, retries int, signals []odigoscommon.ObservabilitySignal,
	collection *odigosv1.CollectorsGroup, ctx context.Context, c client.Client) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Skip new calls if the function is already in progress
	if dm.inProgress {
		return
	}

	dm.inProgress = true

	// Finish the function execution after the delay
	time.AfterFunc(delay, func() {
		var err error
		logger := log.FromContext(ctx)

		dm.mu.Lock()
		defer dm.mu.Unlock()
		defer dm.finishProgress()
		defer func() {
			statusPatchString := common.GetCollectorsGroupDeployedConditionsPatch(err, collection.Spec.Role)
			statusErr := c.Status().Patch(ctx, collection, client.RawPatch(types.MergePatchType, []byte(statusPatchString)))
			if statusErr != nil {
				logger.Error(statusErr, "Failed to patch collectors group status")
				// just log the error, do not fail the reconciliation
			}
		}()

		for i := 0; i < retries; i++ {
			err = syncCollectorGroup(ctx, collection, c)
			if err == nil {
				return
			}
		}

		log.FromContext(ctx).Error(err, "Failed to sync DaemonSet")
	})
}

func (dm *DelayManager) finishProgress() {
	dm.inProgress = false
}

func syncCollectorGroup(ctx context.Context, datacollection *odigosv1.CollectorsGroup, c client.Client) error {
	logger := log.FromContext(ctx)

	configMap, err := getConfigMap(ctx, c, datacollection.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get Config Map data")
		return err
	}

	otelcolConfigContent := configMap.Data[k8sconsts.OdigosNodeCollectorConfigMapKey]
	signals, err := getSignalsFromOtelcolConfig(otelcolConfigContent)
	if err != nil {
		logger.Error(err, "Failed to get signals from otelcol config")
		return err
	}

	err = common.UpdateCollectorGroupReceiverSignals(ctx, c, datacollection, signals)
	if err != nil {
		logger.Error(err, "Failed to update node collectors group received signals")
		return err
	}

	return nil
}
