package sdkconfig

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/opampserver/pkg/connection"
	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configsections"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationConfigReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	ConnectionCache *connection.ConnectionsCache
}

func (i *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instrumentationConfig := &odigosv1.InstrumentationConfig{}
	err := i.Get(ctx, req.NamespacedName, instrumentationConfig)

	if err != nil {
		if apierrors.IsNotFound(err) {
			instrumentationConfig = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	podWorkload := workload.PodWorkload{
		Namespace: req.Namespace,
		Kind:      workload.WorkloadKind(workloadKind),
		Name:      workloadName,
	}

	instrumentationLibrariesConfig, err := configsections.CalcInstrumentationLibrariesRemoteConfig(ctx, i.Client, req.Name, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	opampRemoteConfigInstrumentationLibraries, instrumentationLibrariesSectionName, err := configsections.InstrumentationLibrariesRemoteConfigToOpamp(instrumentationLibrariesConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	updatedConfigMapEntries := protobufs.AgentConfigMap{
		ConfigMap: map[string]*protobufs.AgentConfigFile{
			instrumentationLibrariesSectionName: opampRemoteConfigInstrumentationLibraries,
		},
	}

	connectionInfo := i.ConnectionCache.GetConnectionsInfoByWorkload(podWorkload)

	if connectionInfo != nil {
		for _, con := range connectionInfo {
			connectionProgragmingLanguage := con.ProgrammingLanguage
			workloadInstrumentationConfig, err := configsections.FilterRelevantSdk(instrumentationConfig, connectionProgragmingLanguage)
			if err != nil {
				return ctrl.Result{}, err
			}
			updatedConfigMapEntries.ConfigMap[""] = workloadInstrumentationConfig
			i.ConnectionCache.UpdateWorkloadRemoteConfigByKeys(podWorkload, &updatedConfigMapEntries, connectionProgragmingLanguage)
		}
	}

	return ctrl.Result{}, nil
}

func (i *InstrumentationConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("opampserver-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		Complete(i)
}
