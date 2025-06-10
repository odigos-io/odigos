package instrumentationconfig

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// These controllers handle update of the InstrumentationConfig's ServiceName
// whenever there are changes in the associated Source object.
type SourceReconciler struct {
	client.Client
}

//source Namespace `production`
// labels:
// datastream-A: true
// datastream-B: true

// source workload `Frontend`
// labels:
// dataStream-A: false

// source workload `Backend`
// labels:

// expected instrumentationconfigs:
// Frontend:
// labels:
// dataStream-A: false
// datastream-B: true

// Backend:
// labels:
// datastream-A:true
// datastream-B:true

func (r *SourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	source := &odigosv1alpha1.Source{}
	err := r.Get(ctx, req.NamespacedName, source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	sourceDataStreamsLabels := sourceutils.GetSourceDataStreamsLabels(source)
	// for the example: sourceDataStreamsLabels = {datastream-A: true, datastream-B: true}

	// Namespace Source Reconciliation
	if source.Spec.Workload.Kind == k8sconsts.WorkloadKindNamespace {
		// Step 1: Load all workload sources in the namespace
		var workloadSources odigosv1alpha1.SourceList
		if err := r.List(ctx, &workloadSources, client.InNamespace(req.Namespace)); err != nil {
			return ctrl.Result{}, err
		}

		// Build map: workloadName -> workloadSource for all sources in the namespace that are not namespace sources
		// because we will need to change them in case namespace source labels were changed.
		workloadSourceMap := make(map[string]*odigosv1alpha1.Source)
		for i, wlSource := range workloadSources.Items {

			// skip namespace sources
			if wlSource.Spec.Workload.Kind == k8sconsts.WorkloadKindNamespace {
				continue
			}

			workloadNameFromLabel := wlSource.Labels[wlSource.Spec.Workload.Name]
			if workloadNameFromLabel != "" {
				workloadSourceMap[workloadNameFromLabel] = &workloadSources.Items[i]
			} else {
				logger.Info("Workload Source missing workload-name label", "name", wlSource.Name)
			}
		}

		// Step 2: Load all InstrumentationConfigs in the namespace
		var instConfigs odigosv1alpha1.InstrumentationConfigList
		if err := r.List(ctx, &instConfigs, client.InNamespace(req.Namespace)); err != nil {
			return ctrl.Result{}, err
		}

		for _, instConfig := range instConfigs.Items {

			// Extract workload name from InstrumentationConfig name
			workloadName, _, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(instConfig.Name)
			if err != nil {
				logger.Error(err, "Failed to extract workload info from instrumentation config name", "instrumentationConfig", instConfig.Name)
				continue
			}

			// Get namespace datastream labels and copy them to the mergedLabels map
			mergedLabels := copyLabels(sourceDataStreamsLabels)

			// Apply overrides from workload source (if exists)
			// This is done in case a specific workload source has been uninstrumented even when it's namespace source is instrumented.
			workloadSource, exists := workloadSourceMap[workloadName]
			if exists {
				wlLabels := sourceutils.GetSourceDataStreamsLabels(workloadSource)
				for key, value := range wlLabels {
					if value == "false" {
						if _, nsHasLabel := mergedLabels[key]; nsHasLabel {
							mergedLabels[key] = "false"
						}
					}
				}
			}

			if mergeInstrumentationConfigLabels(&instConfig, mergedLabels) {
				logger.Info("Updating InstrumentationConfig (Namespace reconcile)", "instrumentationConfig", instConfig.Name, "namespace", req.Namespace)
				if err := r.Update(ctx, &instConfig); err != nil {
					return utils.K8SUpdateErrorHandler(err)
				}
			}
		}

		return ctrl.Result{}, nil
	}

	// Workload Source Reconciliation stays the same:

	instConfigName := workload.CalculateWorkloadRuntimeObjectName(source.Spec.Workload.Name, source.Spec.Workload.Kind)
	instConfig := &odigosv1alpha1.InstrumentationConfig{}
	err = r.Get(ctx, types.NamespacedName{Name: instConfigName, Namespace: req.Namespace}, instConfig)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if mergeInstrumentationConfigLabels(instConfig, sourceDataStreamsLabels) {
		logger.Info("Updating InstrumentationConfig (Workload reconcile)", "instrumentationConfig", instConfigName, "namespace", req.Namespace)
		if err := r.Update(ctx, instConfig); err != nil {
			return utils.K8SUpdateErrorHandler(err)
		}
	}

	// Keep existing serviceName logic untouched
	currentServiceName := source.Spec.OtelServiceName
	if currentServiceName == "" {
		currentServiceName = source.Spec.Workload.Name
	}

	if instConfig.Spec.ServiceName != currentServiceName {
		instConfig.Spec.ServiceName = currentServiceName
		logger.Info("Updating InstrumentationConfig service name", "instrumentationConfig", instConfigName, "namespace", req.Namespace, "serviceName", source.Spec.OtelServiceName)
		err = r.Update(ctx, instConfig)
		return utils.K8SUpdateErrorHandler(err)
	}

	return reconcile.Result{}, nil
}

// Utility function for safe map copy
func copyLabels(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for k, v := range source {
		result[k] = v
	}
	return result
}

func mergeInstrumentationConfigLabels(instConfig *odigosv1alpha1.InstrumentationConfig, desiredLabels map[string]string) (updated bool) {
	if instConfig.Labels == nil {
		instConfig.Labels = make(map[string]string)
	}

	// Add / update labels
	for key, value := range desiredLabels {
		if instConfig.Labels[key] != value {
			instConfig.Labels[key] = value
			updated = true
		}
	}

	// Remove datastream labels not present in desiredLabels
	for key := range instConfig.Labels {
		if _, exists := desiredLabels[key]; !exists && sourceutils.IsDataStreamLabel(key) {
			delete(instConfig.Labels, key)
			updated = true
		}
	}

	return updated
}
