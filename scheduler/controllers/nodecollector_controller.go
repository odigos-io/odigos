package controllers

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	k8sutilsconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	odigospredicates "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	nodeCollectorGroupUtil "github.com/odigos-io/odigos/scheduler/controllers/collectorgroups"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NodeCollectorsGroupReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ImagePullSecrets []string
	OdigosVersion    string
}

func (r *NodeCollectorsGroupReconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	namespace := env.GetCurrentNamespace()

	var instrumentedConfigs odigosv1.InstrumentationConfigList
	err := r.List(ctx, &instrumentedConfigs)
	if err != nil {
		logger.Error(err, "failed to list InstrumentationConfigs")
		return ctrl.Result{}, err
	}
	numberOfInstrumentedApps := len(instrumentedConfigs.Items)

	if numberOfInstrumentedApps == 0 {
		if err = utils.DeleteCollectorGroup(ctx, r.Client, namespace, k8sutilsconsts.OdigosNodeCollectorCollectorGroupName); err != nil {
			return ctrl.Result{}, err
		}
	}

	clusterCollectorGroup, err := utils.GetCollectorGroup(ctx, r.Client, namespace, k8sutilsconsts.OdigosClusterCollectorCollectorGroupName)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.V(3).Info("collector group doesn't exist", "collectorGroupName", clusterCollectorGroup)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to get collector group", "collectorGroupName", k8sutilsconsts.OdigosClusterCollectorCollectorGroupName)
		return ctrl.Result{}, err
	}

	odigosConfig, err := utils.GetCurrentOdigosConfig(ctx, r.Client)
	if err != nil {
		logger.Error(err, "failed to get odigos config")
		return ctrl.Result{}, err
	}

	if nodeCollectorGroupUtil.ShouldHaveNodeCollectorGroup(clusterCollectorGroup.Status.Ready, numberOfInstrumentedApps) {
		err = utils.ApplyCollectorGroup(ctx, r.Client, nodeCollectorGroupUtil.NewNodeCollectorGroup(odigosConfig))
		if err != nil {
			logger.Error(err, "failed to create data collection collector group")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *NodeCollectorsGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// here we enumerate the inputs events that the controller when data collection collector group should be updated

	err := ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.InstrumentationConfig{}).
		Named("nodecollectorgroup-instrumentationconfig").
		WithEventFilter(&odigospredicates.ExistencePredicate{}).
		Complete(r)
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Named("nodecollectorgroup-odigosconfig").
		WithEventFilter(&odigospredicates.OdigosConfigMapPredicate).
		Complete(r)
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.CollectorsGroup{}).
		Named("nodecollectorgroup-collectorsgroup").
		WithEventFilter(&odigospredicates.OdigosCollectorsGroupCluster).
		WithEventFilter(&odigospredicates.CgBecomesReadyPredicate{}).
		Complete(r)
	if err != nil {
		return err
	}

	return nil
}
