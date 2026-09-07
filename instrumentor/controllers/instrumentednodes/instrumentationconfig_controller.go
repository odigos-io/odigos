package instrumentednodes

import (
	"context"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

// InstrumentationConfigReconciler syncs instrumented-pods node labels for nodes
// that run pods belonging to the workload of the reconciled InstrumentationConfig.
type InstrumentationConfigReconciler struct {
	client.Client
	NodeLabelRetention time.Duration
}

func (r *InstrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pw, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	nodeNames, err := r.nodeNamesForWorkload(ctx, pw)
	if err != nil {
		return ctrl.Result{}, err
	}

	var syncErrs []error
	var requeueAfter time.Duration
	for nodeName := range nodeNames {
		ra, err := syncNode(ctx, r.Client, nodeName, r.NodeLabelRetention)
		if err != nil {
			syncErrs = append(syncErrs, err)
			continue
		}
		if ra > 0 && (requeueAfter == 0 || ra < requeueAfter) {
			requeueAfter = ra
		}
	}

	res, err := utils.K8SUpdateErrorHandler(errors.Join(syncErrs...))
	if err != nil {
		return res, err
	}
	if requeueAfter > 0 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	return res, nil
}

func (r *InstrumentationConfigReconciler) nodeNamesForWorkload(ctx context.Context, pw k8sconsts.PodWorkload) (map[string]struct{}, error) {
	nodes := map[string]struct{}{}

	workloadObj := workload.ClientObjectFromWorkloadKind(pw.Kind)
	if workloadObj == nil {
		return nodes, nil
	}

	err := r.Get(ctx, client.ObjectKey{Namespace: pw.Namespace, Name: pw.Name}, workloadObj)
	if err != nil {
		return nodes, client.IgnoreNotFound(err)
	}

	switch pw.Kind {
	case k8sconsts.WorkloadKindStaticPod:
		pod := workloadObj.(*corev1.Pod)
		if pod.Spec.NodeName != "" {
			nodes[pod.Spec.NodeName] = struct{}{}
		}
		return nodes, nil
	case k8sconsts.WorkloadKindCronJob:
		var pods corev1.PodList
		err = r.List(ctx, &pods, client.InNamespace(pw.Namespace))
		if err != nil {
			return nil, err
		}
		for i := range pods.Items {
			pod := &pods.Items[i]
			podWorkload, err := workload.PodWorkloadObject(pod)
			if err != nil || podWorkload == nil {
				continue
			}
			if podWorkload.Name == pw.Name && podWorkload.Kind == pw.Kind && pod.Spec.NodeName != "" {
				nodes[pod.Spec.NodeName] = struct{}{}
			}
		}
		return nodes, nil
	default:
		genericWorkload, err := workload.ObjectToWorkload(workloadObj)
		if err != nil {
			return nil, err
		}
		labelSelector := genericWorkload.LabelSelector()
		if labelSelector == nil {
			return nodes, nil
		}

		var pods corev1.PodList
		err = r.List(ctx, &pods, client.InNamespace(pw.Namespace), client.MatchingLabels(labelSelector.MatchLabels))
		if err != nil {
			return nil, err
		}
		for i := range pods.Items {
			if pods.Items[i].Spec.NodeName != "" {
				nodes[pods.Items[i].Spec.NodeName] = struct{}{}
			}
		}
		return nodes, nil
	}
}
