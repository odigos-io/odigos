package sourceinstrumentation

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pw := k8sconsts.PodWorkload{
		Namespace: req.Namespace,
		Kind:      k8sconsts.WorkloadKindDeployment,
		Name:      req.Name,
	}
	return syncWorkload(ctx, r.Client, r.Scheme, pw)
}

type DaemonSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pw := k8sconsts.PodWorkload{
		Namespace: req.Namespace,
		Kind:      k8sconsts.WorkloadKindDaemonSet,
		Name:      req.Name,
	}
	return syncWorkload(ctx, r.Client, r.Scheme, pw)
}

type StatefulSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pw := k8sconsts.PodWorkload{
		Namespace: req.Namespace,
		Kind:      k8sconsts.WorkloadKindStatefulSet,
		Name:      req.Name,
	}
	return syncWorkload(ctx, r.Client, r.Scheme, pw)
}

type CronJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pw := k8sconsts.PodWorkload{
		Namespace: req.Namespace,
		Kind:      k8sconsts.WorkloadKindCronJob,
		Name:      req.Name,
	}
	return syncWorkload(ctx, r.Client, r.Scheme, pw)
}

type DeploymentConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DeploymentConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pw := k8sconsts.PodWorkload{
		Namespace: req.Namespace,
		Kind:      k8sconsts.WorkloadKindDeploymentConfig,
		Name:      req.Name,
	}
	return syncWorkload(ctx, r.Client, r.Scheme, pw)
}

type RolloutReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *RolloutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pw := k8sconsts.PodWorkload{
		Namespace: req.Namespace,
		Kind:      k8sconsts.WorkloadKindArgoRollout,
		Name:      req.Name,
	}
	return syncWorkload(ctx, r.Client, r.Scheme, pw)
}
