package controllers

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	nodeCollectorGroupUtil "github.com/odigos-io/odigos/scheduler/controllers/collectorgroups"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ImagePullSecrets []string
	OdigosVersion    string
}

func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling InstrumentationConfig")

	namespace := env.GetCurrentNamespace()

	var instrumentedConfigs odigosv1.InstrumentationConfigList
	err := r.List(ctx, &instrumentedConfigs)
	if err != nil {
		logger.Error(err, "failed to list InstrumentationConfigs")
		return ctrl.Result{}, err
	}
	numberOfInstrumentedApps := len(instrumentedConfigs.Items)

	if numberOfInstrumentedApps == 0 {
		if err = utils.DeleteCollectorGroup(ctx, r.Client, namespace, consts.OdigosNodeCollectorCollectorGroupName); err != nil {
			return ctrl.Result{}, err
		}
	}

	clusterCollectorGroup, err := utils.GetCollectorGroup(ctx, r.Client, namespace, consts.OdigosClusterCollectorCollectorGroupName)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.V(3).Info("collector group doesn't exist", "collectorGroupName", clusterCollectorGroup)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to get collector group", "collectorGroupName", consts.OdigosClusterCollectorCollectorGroupName)
		return ctrl.Result{}, err
	}

	dataCollectionExists := true
	_, err = utils.GetCollectorGroup(ctx, r.Client, namespace, consts.OdigosNodeCollectorDaemonSetName)
	if err != nil {
		if errors.IsNotFound(err) {
			dataCollectionExists = false
		} else {
			logger.Error(err, "failed to get collector group", "collectorGroupName", consts.OdigosNodeCollectorCollectorGroupName)
			return ctrl.Result{}, err
		}
	}

	if nodeCollectorGroupUtil.ShouldCreateNodeCollectorGroup(clusterCollectorGroup.Status.Ready, dataCollectionExists, numberOfInstrumentedApps) {
		err = utils.CreateCollectorGroup(ctx, r.Client, nodeCollectorGroupUtil.NewNodeCollectorGroup())
		if err != nil {
			logger.Error(err, "failed to create data collection collector group")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *InstrumentationConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.InstrumentationConfig{}).
		Complete(r)
}
