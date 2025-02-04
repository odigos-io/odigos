package workloadrollout

import (
	"context"
	"encoding/hex"
	"time"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type instrumentationConfigReconciler struct {
	client.Client
}

const requeueWaitingForWorkloadRollout = 10 * time.Second

func (r *instrumentationConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name)
	if err != nil {
		logger.Error(err, "error parsing workload info from runtime object name")
		return ctrl.Result{}, nil
	}

	workloadObj:= workload.ClientObjectFromWorkloadKind(workloadKind)
	err = r.Get(ctx, client.ObjectKey{Name: workloadName, Namespace: req.Namespace}, workloadObj)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var ic odigosv1alpha1.InstrumentationConfig
	err = r.Get(ctx, req.NamespacedName, &ic)
	now := time.Now()

	if err != nil {
		if apierrors.IsNotFound(err) {
			// instrumentation config is deleted, trigger a rollout for the associated workload
			// this should happen once per workload, as the instrumentation config is deleted
			rolloutErr := rolloutRestartWorkload(ctx, workloadObj, r.Client, now)
			if rolloutErr != nil {
				logger.Error(rolloutErr, "error rolling out workload", "name", workloadName, "namespace", req.Namespace)
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	savedRolloutHash := ic.Status.WorkloadRolloutHash
	newRolloutHash, err := configHash(&ic)
	if err != nil {
		logger.Error(err, "error calculating rollout hash")
		return ctrl.Result{}, nil
	}

	if savedRolloutHash == newRolloutHash {
		return ctrl.Result{}, nil
	}

	// if the rollout is ongoing, wait for it to finish, requeue
	if !isWorkloadRolloutDone(workloadObj) {
		return ctrl.Result{RequeueAfter: requeueWaitingForWorkloadRollout}, nil
	}

	rolloutErr := rolloutRestartWorkload(ctx, workloadObj, r.Client, now)
	if rolloutErr != nil {
		logger.Error(rolloutErr, "error rolling out workload", "name", workloadName, "namespace", req.Namespace)
	}

	ic.Status.WorkloadRolloutHash = newRolloutHash
	meta.SetStatusCondition(&ic.Status.Conditions, rolloutCondition(rolloutErr))
	err = r.Client.Update(ctx, &ic)
	return utils.K8SUpdateErrorHandler(err)
}

func rolloutCondition(rolloutErr error) metav1.Condition {
	cond := metav1.Condition{
		Type:    odigosv1alpha1.WorkloadRolloutConditionType,
	}

	if rolloutErr == nil {
		cond.Status = metav1.ConditionTrue
	} else {
		cond.Status = metav1.ConditionFalse
		cond.Message = rolloutErr.Error()
	}

	return cond
}

func configHash(ic *odigosv1alpha1.InstrumentationConfig) (string, error) {
	if !ic.Spec.AgentInjectionEnabled {
		return "", nil
	}

	newRolloutHashBytes, err := hashForContainersConfig(ic.Spec.Containers)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(newRolloutHashBytes), nil
}
