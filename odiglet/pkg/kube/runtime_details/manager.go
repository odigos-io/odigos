package runtime_details

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"

	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

func SetupWithManager(mgr ctrl.Manager, clientset *kubernetes.Clientset, criClient *criwrapper.CriClient, appendEnvVarNames map[string]struct{}) error {

	runtimeDetectionEnvs := map[string]struct{}{
		// LD_PRELOAD is special, and is always collected.
		// It has special handling that does not require it to be set in the "AppendOdigosVariables" list.
		consts.LdPreloadEnvVarName: {},
	}
	for envName := range appendEnvVarNames {
		runtimeDetectionEnvs[envName] = struct{}{}
	}

	readyPred := &odigospredicate.AllContainersReadyPredicate{}
	err := builder.
		ControllerManagedBy(mgr).
		Named("Odiglet-RuntimeDetails-Pods").
		For(&corev1.Pod{}, builder.WithPredicates(readyPred)).
		Complete(&PodsReconciler{
			Client:               mgr.GetClient(),
			Scheme:               mgr.GetScheme(),
			Clientset:            clientset,
			CriClient:            criClient,
			RuntimeDetectionEnvs: runtimeDetectionEnvs,
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("Odiglet-RuntimeDetails-InstrumentationConfig").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(&instrumentationConfigPredicate{}).
		Complete(&InstrumentationConfigReconciler{
			Client:               mgr.GetClient(),
			Scheme:               mgr.GetScheme(),
			Clientset:            clientset,
			CriClient:            criClient,
			RuntimeDetectionEnvs: runtimeDetectionEnvs,
		})
	if err != nil {
		return err
	}

	return nil
}
