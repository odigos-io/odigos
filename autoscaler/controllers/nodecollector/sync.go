package nodecollector

import (
	"context"
	"slices"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var dm = &DelayManager{}

const (
	syncDaemonsetRetry = 3
)

type nodeCollectorBaseReconciler struct {
	client.Client
	scheme               *runtime.Scheme
	autoscalerDeployment *appsv1.Deployment
}

func (b *nodeCollectorBaseReconciler) reconcileNodeCollector(ctx context.Context) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ics odigosv1.InstrumentationConfigList
	if err := b.Client.List(ctx, &ics); err != nil {
		return ctrl.Result{}, err
	}

	odigosNs := env.GetCurrentNamespace()

	dataCollectionCollectorGroup := &odigosv1.CollectorsGroup{}
	err := b.Client.Get(ctx, client.ObjectKey{Namespace: odigosNs, Name: k8sconsts.OdigosNodeCollectorCollectorGroupName}, dataCollectionCollectorGroup)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		dataCollectionCollectorGroup = nil
	}

	var clusterCollectorCollectorGroup odigosv1.CollectorsGroup
	err = b.Client.Get(ctx, client.ObjectKey{Namespace: odigosNs, Name: k8sconsts.OdigosClusterCollectorConfigMapName}, &clusterCollectorCollectorGroup)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var processors odigosv1.ProcessorList
	if err := b.Client.List(ctx, &processors); err != nil {
		logger.Error(err, "Failed to list processors")
		return ctrl.Result{}, err
	}

	err = b.syncDataCollection(ctx, &ics, clusterCollectorCollectorGroup, &processors, dataCollectionCollectorGroup)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (b *nodeCollectorBaseReconciler) syncDataCollection(ctx context.Context, sources *odigosv1.InstrumentationConfigList, clusterCollectorGroup odigosv1.CollectorsGroup, processors *odigosv1.ProcessorList,
	dataCollection *odigosv1.CollectorsGroup) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Syncing data collection")

	err := b.syncService(ctx, dataCollection)
	if err != nil {
		logger.Error(err, "Failed to sync service")
		return err
	}

	err = b.SyncConfigMap(ctx, sources, clusterCollectorGroup, processors, dataCollection)
	if err != nil {
		logger.Error(err, "Failed to sync config map")
		return err
	}

	// enabled signals also takes into account spanmetrics connector
	// e.g - cluster collector can accept only metrics,
	// while node collector collects both metrics and traces, which it converts to metrics and does not forward downstream.
	// the enabled signals represents what's actually collected from agents in node collector.
	enabledSignals := clusterCollectorGroup.Status.ReceiverSignals
	spanMetricsEnabled := dataCollection != nil && dataCollection.Spec.Metrics != nil && dataCollection.Spec.Metrics.SpanMetrics != nil
	if spanMetricsEnabled {
		if !slices.Contains(enabledSignals, common.TracesObservabilitySignal) {
			enabledSignals = append(enabledSignals, common.TracesObservabilitySignal)
		}
	}

	dm.RunSyncDaemonSetWithDelayAndSkipNewCalls(time.Duration(env.GetSyncDaemonSetDelay())*time.Second, syncDaemonsetRetry, enabledSignals, dataCollection, ctx, b.Client)

	return nil
}
