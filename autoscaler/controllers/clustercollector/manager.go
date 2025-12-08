package clustercollector

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager, odigosVersion string) error {

	err := builder.
		ControllerManagedBy(mgr).
		Named("clustercollector-collectorsgroup").
		For(&odigosv1.CollectorsGroup{}).
		Owns(&appsv1.Deployment{}). // in case the cluster collector deployment is deleted or modified for any reason, this will reconcile and recreate it
		Owns(&corev1.ConfigMap{}).  // in case the configmap is deleted or modified for any reason, this will reconcile and recreate it
		// we assume everything in the collectorsgroup spec is the configuration for the collectors to generate.
		// thus, we need to monitor any change to the spec which is what the generation field is for.
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(&CollectorsGroupReconciler{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			OdigosVersion: odigosVersion,
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("clustercollector-destinations").
		For(&odigosv1.Destination{}).
		// auto scaler only cares about the spec of each destination.
		// filter out events on resource status and metadata changes.
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(&DestinationReconciler{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			OdigosVersion: odigosVersion,
		})
	if err != nil {
		return err
	}

	// We need to react to changes in the InstrumentationConfig CRs because we're relying
	// on the labels to build the connectors configuration in the gateway configmap for datastreams.
	err = builder.
		ControllerManagedBy(mgr).
		Named("clustercollector-instrumentationconfigs").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(predicate.Or(
			odigospredicate.ExistencePredicate{},
			predicate.LabelChangedPredicate{},
		)).
		Complete(&InstrumentationConfigReconciler{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			OdigosVersion: odigosVersion,
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("clustercollector-processors").
		For(&odigosv1.Processor{}).
		// auto scaler only cares about the spec of each processor.
		// filter out events on resource status and metadata changes.
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(&ProcessorReconciler{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			OdigosVersion: odigosVersion,
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("clustercollector-secret").
		For(&corev1.Secret{}).
		// we need to handle secrets only when they are updated.
		// this is to trigger redeployment of the cluster collector in case of destination secret change.
		// when the secret was just created (via auto-scaler restart or initial deployment), the cluster collector will be reconciled by other controllers.
		WithEventFilter(predicate.And(&odigospredicate.OnlyUpdatesPredicate{}, &predicate.ResourceVersionChangedPredicate{})).
		Complete(&SecretReconciler{
			Client:        mgr.GetClient(),
			Scheme:        mgr.GetScheme(),
			OdigosVersion: odigosVersion,
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("clustercollector-deployment").
		For(&appsv1.Deployment{}).
		WithEventFilter(&odigospredicate.ClusterCollectorDeploymentPredicate).
		Complete(&ClusterCollectorDeploymentReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	return nil
}
