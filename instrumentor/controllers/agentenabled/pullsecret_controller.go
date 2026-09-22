package agentenabled

import (
	"context"
	"errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/pro"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

type PullSecretSyncReconciler struct {
	client.Client
}

type syncOdigosPullSecretsAnnotationPredicate struct {
	predicate.Funcs
}

func (syncOdigosPullSecretsAnnotationPredicate) Create(e event.CreateEvent) bool {
	return pullSecretSyncAnnotation(e.Object) != ""
}

func (syncOdigosPullSecretsAnnotationPredicate) Update(e event.UpdateEvent) bool {
	return pullSecretSyncAnnotation(e.ObjectNew) != "" &&
		pullSecretSyncAnnotation(e.ObjectOld) != pullSecretSyncAnnotation(e.ObjectNew)
}

func (syncOdigosPullSecretsAnnotationPredicate) Delete(event.DeleteEvent) bool {
	return false
}

func pullSecretSyncAnnotation(obj client.Object) string {
	if obj == nil {
		return ""
	}
	return obj.GetAnnotations()[k8sconsts.SyncOdigosPullSecretsAnnotation]
}

func (r *PullSecretSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)
	conf, err := k8sutils.GetCurrentOdigosConfiguration(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !conf.SyncOdigosPullSecrets {
		return ctrl.Result{}, nil
	}

	listed := false
	for _, name := range conf.ImagePullSecrets {
		if name == req.Name {
			listed = true
			break
		}
	}
	if !listed {
		return ctrl.Result{}, nil
	}

	ics := odigosv1.InstrumentationConfigList{}
	if err := r.List(ctx, &ics); err != nil {
		return ctrl.Result{}, err
	}

	sourceNs := env.GetCurrentNamespace()
	seen := map[string]struct{}{}
	var errs error
	for i := range ics.Items {
		ns := ics.Items[i].Namespace
		if ns == "" || ns == sourceNs {
			continue
		}
		if _, ok := seen[ns]; ok {
			continue
		}
		seen[ns] = struct{}{}
		copyErr := pro.CopyImagePullSecrets(ctx, r.Client, r.Client, sourceNs, ns, []string{req.Name})
		if errors.Is(copyErr, pro.ErrDestAlreadyExists) {
			logger.Error(copyErr, "refusing to overwrite an unmanaged image pull secret; delete or rename it and re-annotate the source to retry",
				"secret", req.Name, "namespace", ns)
			copyErr = pro.IgnoreDestAlreadyExists(copyErr)
		}
		if copyErr != nil {
			errs = errors.Join(errs, copyErr)
		}
	}
	return ctrl.Result{}, errs
}
