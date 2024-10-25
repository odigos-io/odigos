package controllers

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	consts "github.com/odigos-io/odigos/common/consts"
	k8sutilsconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	nodeCollectorGroupUtil "github.com/odigos-io/odigos/scheduler/controllers/collectorgroups"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type NodeCollectorsGroupReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ImagePullSecrets []string
	OdigosVersion    string
}

// makes sure that the controller only reacts to events related to the odigos-config configmap
// and does not trigger on other configmaps
type odigosConfigPredicate struct{}

func (i *odigosConfigPredicate) Create(e event.CreateEvent) bool {
	return e.Object.GetName() == consts.OdigosConfigurationName
}

func (i *odigosConfigPredicate) Update(e event.UpdateEvent) bool {
	return e.ObjectNew.GetName() == consts.OdigosConfigurationName
}

func (i *odigosConfigPredicate) Delete(e event.DeleteEvent) bool {
	return e.Object.GetName() == consts.OdigosConfigurationName
}

func (i *odigosConfigPredicate) Generic(e event.GenericEvent) bool {
	return e.Object.GetName() == consts.OdigosConfigurationName
}

var _ predicate.Predicate = &odigosConfigPredicate{}

// For instrumentation configs, we only care if the object exists or not, since we count if there are more than 0.
// thus, we can filter out all updates events which will not affect reconciliation
type existingPredicate struct{}

func (i *existingPredicate) Create(e event.CreateEvent) bool {
	return true
}

func (i *existingPredicate) Update(e event.UpdateEvent) bool {
	return false
}

func (i *existingPredicate) Delete(e event.DeleteEvent) bool {
	return true
}

func (i *existingPredicate) Generic(e event.GenericEvent) bool {
	return false
}

var _ predicate.Predicate = &existingPredicate{}

// this predicate filters collectorsgroup events.
// it will only forward events that are:
// 1. for cluster collector group
// 2. If the cluster collector group was not ready and now it is ready
type clusterCollectorBecomesReadyPredicate struct{}

func (i *clusterCollectorBecomesReadyPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (i *clusterCollectorBecomesReadyPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectNew.GetName() != k8sutilsconsts.OdigosClusterCollectorCollectorGroupName {
		return false
	}

	oldCollectorGroup, ok := e.ObjectOld.(*odigosv1.CollectorsGroup)
	if !ok {
		return false
	}
	newCollectorGroup, ok := e.ObjectNew.(*odigosv1.CollectorsGroup)
	if !ok {
		return false
	}

	return !oldCollectorGroup.Status.Ready && newCollectorGroup.Status.Ready
}

func (i *clusterCollectorBecomesReadyPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i *clusterCollectorBecomesReadyPredicate) Generic(e event.GenericEvent) bool {
	return false
}

var _ predicate.Predicate = &clusterCollectorBecomesReadyPredicate{}

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
		WithEventFilter(&existingPredicate{}).
		Complete(r)
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Named("nodecollectorgroup-odigosconfig").
		WithEventFilter(&odigosConfigPredicate{}).
		Complete(r)
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.CollectorsGroup{}).
		Named("nodecollectorgroup-collectorsgroup").
		WithEventFilter(&clusterCollectorBecomesReadyPredicate{}).
		Complete(r)
	if err != nil {
		return err
	}

	return nil
}
