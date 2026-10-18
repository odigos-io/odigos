package odigospro

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

func offForbidden(resource string, name string) error {
	return apierrors.NewForbidden(schema.GroupResource{Resource: resource}, name, assert.AnError)
}

// Every cluster call the offsets reconcile makes wraps its failure in its own message, and
// those messages are the only signal an operator gets that the offsets stopped being
// refreshed. Two of them ("failed to apply" and "failed to delete go offsets CronJob") differ
// by a single word, so a copy-paste between them is invisible without pinning each one.
func TestOffsetsControllerErrorsNameTheCallThatFailed(t *testing.T) {
	ns := env.GetCurrentNamespace()

	tests := []struct {
		name    string
		cron    string
		mode    k8sconsts.OffsetCronJobMode
		objects []client.Object
		funcs   interceptor.Funcs
		wantErr string
	}{
		{
			name: "the pro secret cannot be read",
			cron: offCron,
			mode: k8sconsts.OffsetCronJobModeDirect,
			funcs: interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if _, ok := obj.(*corev1.Secret); ok {
						return offForbidden("secrets", key.Name)
					}
					return c.Get(ctx, key, obj, opts...)
				},
			},
			wantErr: "failed to get current Odigos tier",
		},
		{
			name:    "the CronJob cannot be applied",
			cron:    offCron,
			mode:    k8sconsts.OffsetCronJobModeDirect,
			objects: []client.Object{onPremSecret(ns)},
			funcs: interceptor.Funcs{
				Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
					return offForbidden("cronjobs", obj.GetName())
				},
			},
			wantErr: "failed to apply go offsets CronJob",
		},
		{
			name:    "the initial Job cannot be created",
			cron:    offCron,
			mode:    k8sconsts.OffsetCronJobModeDirect,
			objects: []client.Object{onPremSecret(ns)},
			funcs: interceptor.Funcs{
				Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					return offForbidden("jobs", obj.GetName())
				},
			},
			wantErr: "failed to create initial offset updater job",
		},
		{
			name: "the CronJob cannot be deleted",
			cron: "",
			mode: k8sconsts.OffsetCronJobModeDirect,
			objects: []client.Object{
				onPremSecret(ns),
				&batchv1.CronJob{ObjectMeta: offCronJobMeta(ns)},
			},
			funcs: interceptor.Funcs{
				Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					return offForbidden("cronjobs", obj.GetName())
				},
			},
			wantErr: "failed to delete go offsets CronJob",
		},
	}

	messages := map[string]bool{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := append([]client.Object{offEffectiveConfig(ns, tt.cron, tt.mode)}, tt.objects...)
			r := newOffsetsReconcilerWithInterceptor(tt.funcs, objects...)

			_, err := r.Reconcile(context.Background(), ctrl.Request{})

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
			messages[tt.wantErr] = true
		})
	}

	// the same four reconciles against a healthy cluster must all succeed, otherwise the
	// assertions above could be passing for a reason that has nothing to do with the
	// injected failure
	for _, tt := range tests {
		t.Run(tt.name+" (healthy cluster)", func(t *testing.T) {
			objects := append([]client.Object{offEffectiveConfig(ns, tt.cron, tt.mode)}, tt.objects...)
			if tt.objects == nil {
				objects = append(objects, onPremSecret(ns))
			}
			r := newOffsetsReconcilerWithInterceptor(interceptor.Funcs{}, objects...)

			_, err := r.Reconcile(context.Background(), ctrl.Request{})
			require.NoError(t, err)
		})
	}

	assert.Len(t, messages, len(tests), "every cluster call must report its own failure")
}

// The CronJob is read and then deleted, so it can disappear in between - a concurrent
// `odigos uninstall`, or a second reconcile of the same change. That is the outcome the
// reconcile wanted, not a failure to retry.
func TestOffsetsControllerDeleteToleratesTheCronJobVanishing(t *testing.T) {
	ns := env.GetCurrentNamespace()

	r := newOffsetsReconcilerWithInterceptor(interceptor.Funcs{
		Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			return apierrors.NewNotFound(schema.GroupResource{Resource: "cronjobs"}, obj.GetName())
		},
	},
		offEffectiveConfig(ns, offCron, k8sconsts.OffsetCronJobModeOff),
		onPremSecret(ns),
		&batchv1.CronJob{ObjectMeta: offCronJobMeta(ns)},
	)

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	assert.NoError(t, err)
}
