package podsinjectionstatus

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

func syncWorkload(ctx context.Context, client ctrl.Client, pw k8sconsts.PodWorkload) error {

	// get the instrumentation config for the workload.
	// if not found - there is no place to persist the info, so we skip processing.
	icName := workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind)
	ic := odigosv1.InstrumentationConfig{}
	err := client.Get(ctx, types.NamespacedName{Namespace: pw.Namespace, Name: icName}, &ic)
	if err != nil {
		return ctrl.IgnoreNotFound(err)
	}

	// get the workload object to extract the label selector
	// this can be optimized later by writing the label selector into the instrumentation config
	workloadObj := workload.ClientObjectFromWorkloadKind(pw.Kind)
	err = client.Get(ctx, types.NamespacedName{Namespace: pw.Namespace, Name: pw.Name}, workloadObj)
	if err != nil {
		return ctrl.IgnoreNotFound(err)
	}

	genericWorkload, err := workload.ObjectToWorkload(workloadObj)
	if err != nil {
		return err
	}

	labelSelector := genericWorkload.LabelSelector()
	if labelSelector == nil {
		// TODO: handle this case
		return nil
	}

	// get the pods that match the label selector
	pods := &corev1.PodList{}
	err = client.List(ctx, pods, ctrl.MatchingLabels(labelSelector.MatchLabels))
	if err != nil {
		return err
	}

	podsInjectionStatus := odigosv1.PodsInjectionStatus{}

	for _, pod := range pods.Items {
		if agentHashValue, ok := pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel]; ok {
			if agentHashValue == ic.Spec.AgentsMetaHash {
				podsInjectionStatus.NumberUpToDate++
			} else {
				podsInjectionStatus.NumberOutOfDate++
			}
		} else {
			podsInjectionStatus.NumberNotInjected++
		}
	}

	ic.Status.PodsInjectionStatus = &podsInjectionStatus

	err = client.Status().Update(ctx, &ic)
	if err != nil {
		return err
	}

	return nil
}
