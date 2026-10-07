package actions

// The shared URL-templatization Processor is the one Processor in this package that no single
// Action owns: three controllers (ActionReconciler, SharedURLTemplatizationProcessorReconciler and
// URLTemplateNodeCGReconciler) all drive SyncUrlTemplatizationProcessor, and the second of them is
// triggered by the very Processor the sync writes. That makes two things load bearing and neither
// was covered: the two sync modes must differ in whether they patch an existing Processor, and the
// patch body must be identical every time or the Processor watch re-triggers itself forever
// (the CORE-1716 failure mode).

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	apiactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"

	"github.com/stretchr/testify/require"
)

// utsWrites records the mutating calls the sync makes, so a test can assert that a mode did
// nothing rather than only asserting what the stored object ended up looking like.
type utsWrites struct {
	patched []string
	deleted []string
}

func utsClient(t *testing.T, w *utsWrites, objs ...client.Object) client.WithWatch {
	t.Helper()
	return utsClientWith(t, w, interceptor.Funcs{}, objs...)
}

func utsClientWith(t *testing.T, w *utsWrites, funcs interceptor.Funcs, objs ...client.Object) client.WithWatch {
	t.Helper()

	userPatch := funcs.Patch
	funcs.Patch = func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
		if proc, ok := obj.(*odigosv1.Processor); ok {
			data, err := patch.Data(proc)
			if err != nil {
				return err
			}
			w.patched = append(w.patched, string(data))
			// The fake client has no server-side apply; the contract under test is the body the
			// controller sends and whether it sends one at all.
			return nil
		}
		if userPatch != nil {
			return userPatch(ctx, c, obj, patch, opts...)
		}
		return c.Patch(ctx, obj, patch, opts...)
	}

	userDelete := funcs.Delete
	funcs.Delete = func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
		w.deleted = append(w.deleted, obj.GetName())
		if userDelete != nil {
			return userDelete(ctx, c, obj, opts...)
		}
		return c.Delete(ctx, obj, opts...)
	}

	return fake.NewClientBuilder().
		WithScheme(psScheme(t)).
		WithObjects(objs...).
		// Without this the fake client treats status as part of the main object and a
		// Status().Update is silently dropped.
		WithStatusSubresource(&odigosv1.Action{}, &odigosv1.Processor{}).
		WithInterceptorFuncs(funcs).
		Build()
}

func utsUrlTemplatizationAction(name string, disabled bool) *odigosv1.Action {
	a := mkAction(name, odigosv1.ActionSpec{
		ActionName: name,
		Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
		URLTemplatization: &apiactions.URLTemplatizationConfig{
			Rules: []apiactions.UrlTemplatizationRule{
				{Templates: []string{"/api/v1/orders/{id}", "/api/v1/users/{id}"}},
			},
		},
	})
	a.Spec.Disabled = disabled
	return a
}

func utsExistingProcessor() *odigosv1.Processor {
	return &odigosv1.Processor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.URLTemplatizationProcessorName,
			Namespace: psNamespace,
		},
		Spec: odigosv1.ProcessorSpec{Type: consts.OdigosURLTemplateProcessorType},
	}
}

func utsNodeCollectorsGroup(spanMetricsEnabled bool) *odigosv1.CollectorsGroup {
	cg := &odigosv1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorCollectorGroupName,
			Namespace: psNamespace,
		},
		Spec: odigosv1.CollectorsGroupSpec{Role: odigosv1.CollectorsGroupRoleNodeCollector},
	}
	if spanMetricsEnabled {
		cg.Spec.Metrics = &odigosv1.CollectorsGroupMetricsCollectionSettings{
			SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{},
		}
	}
	return cg
}

// TestBuildUrlTemplatizationProcessorPinsTheSharedSpec pins the whole Spec, not only the role, so
// the OrderHint and signal that decide where the processor runs in the pipeline cannot drift.
func TestBuildUrlTemplatizationProcessorPinsTheSharedSpec(t *testing.T) {
	for _, tc := range []struct {
		name              string
		spanMetrics       bool
		wantCollectorRole odigosv1.CollectorsGroupRole
	}{
		{
			name: "span metrics off keeps the processor on the gateway", spanMetrics: false,
			wantCollectorRole: odigosv1.CollectorsGroupRoleClusterGateway,
		},
		{
			// Span metrics label series with span name and http.route, so routes must be templated
			// on the node collector before the metrics are recorded.
			name: "span metrics on moves the processor to the node collector", spanMetrics: true,
			wantCollectorRole: odigosv1.CollectorsGroupRoleNodeCollector,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proc, err := buildUrlTemplatizationProcessor(psNamespace, tc.spanMetrics)
			require.NoError(t, err)

			require.Equal(t, consts.URLTemplatizationProcessorName, proc.Name)
			require.Equal(t, psNamespace, proc.Namespace)
			require.Equal(t, consts.OdigosURLTemplateProcessorType, proc.Spec.Type)
			require.Equal(t, "URL Templatization", proc.Spec.ProcessorName)
			require.False(t, proc.Spec.Disabled)
			require.Equal(t, []common.ObservabilitySignal{common.TracesObservabilitySignal}, proc.Spec.Signals)
			require.Equal(t, []odigosv1.CollectorsGroupRole{tc.wantCollectorRole}, proc.Spec.CollectorRoles)
			require.Equal(t, apiactions.URLTemplatizationConfig{}.OrderHint(), proc.Spec.OrderHint)

			var cfg map[string]any
			require.NoError(t, json.Unmarshal(proc.Spec.ProcessorConfig.Raw, &cfg))
			require.Equal(t,
				map[string]any{"odigos_config_extension": k8sconsts.OdigosConfigK8sExtensionType},
				cfg)
		})
	}
}

// TestHasAnyUrlTemplatizationActionDistinguishesEveryState covers the three ways an action can
// fail to count. A disabled action keeping the processor alive would leave URLs templated after
// the user switched the action off.
func TestHasAnyUrlTemplatizationActionDistinguishesEveryState(t *testing.T) {
	otherType := mkAction("rename", odigosv1.ActionSpec{
		Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
		PiiMasking: &apiactions.PiiMaskingConfig{},
	})
	otherNamespace := utsUrlTemplatizationAction("elsewhere", false)
	otherNamespace.Namespace = "some-other-namespace"

	for _, tc := range []struct {
		name string
		objs []client.Object
		want bool
	}{
		{name: "no actions at all", want: false},
		{name: "an enabled URL templatization action", objs: []client.Object{utsUrlTemplatizationAction("url", false)}, want: true},
		{name: "a disabled URL templatization action", objs: []client.Object{utsUrlTemplatizationAction("url", true)}, want: false},
		{name: "an action of another type", objs: []client.Object{otherType}, want: false},
		{name: "a URL templatization action in another namespace", objs: []client.Object{otherNamespace}, want: false},
		{
			name: "one enabled action beside a disabled one",
			objs: []client.Object{utsUrlTemplatizationAction("off", true), utsUrlTemplatizationAction("on", false)},
			want: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var w utsWrites
			got, err := hasAnyUrlTemplatizationAction(context.Background(), utsClient(t, &w, tc.objs...), psNamespace)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestHasAnyUrlTemplatizationActionPropagatesListErrors(t *testing.T) {
	var w utsWrites
	c := utsClientWith(t, &w, interceptor.Funcs{
		List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
			return errors.New("api server unavailable")
		},
	})
	_, err := hasAnyUrlTemplatizationAction(context.Background(), c, psNamespace)
	require.ErrorContains(t, err, "api server unavailable")
}

// TestSyncUrlTemplatizationProcessorModes is the reason the two modes exist: CreateIfMissing is
// called from every Action reconcile and must not touch an existing Processor, while ApplyFull is
// called from the drift correctors and must always rewrite it. Asserting the recorded write count
// is what catches a mode that silently collapses into the other.
func TestSyncUrlTemplatizationProcessorModes(t *testing.T) {
	for _, tc := range []struct {
		name        string
		mode        URLTemplatizationSyncMode
		exists      bool
		wantPatches int
	}{
		{name: "create if missing creates when absent", mode: URLTemplatizationSyncCreateIfMissing, exists: false, wantPatches: 1},
		{name: "create if missing leaves an existing processor alone", mode: URLTemplatizationSyncCreateIfMissing, exists: true, wantPatches: 0},
		{name: "apply full creates when absent", mode: URLTemplatizationSyncApplyFull, exists: false, wantPatches: 1},
		{name: "apply full rewrites an existing processor", mode: URLTemplatizationSyncApplyFull, exists: true, wantPatches: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(consts.CurrentNamespaceEnvVar, psNamespace)

			objs := []client.Object{utsUrlTemplatizationAction("url", false), utsNodeCollectorsGroup(false)}
			if tc.exists {
				objs = append(objs, utsExistingProcessor())
			}
			var w utsWrites
			c := utsClient(t, &w, objs...)

			require.NoError(t, SyncUrlTemplatizationProcessor(context.Background(), c, tc.mode))

			require.Len(t, w.patched, tc.wantPatches)
			require.Empty(t, w.deleted, "a live URL templatization action must keep the processor")
		})
	}
}

// TestSyncUrlTemplatizationProcessorDeletesWhenNoActionNeedsIt pins the other half of the
// lifecycle: the shared Processor is not owned by any Action, so nothing garbage-collects it.
func TestSyncUrlTemplatizationProcessorDeletesWhenNoActionNeedsIt(t *testing.T) {
	for _, tc := range []struct {
		name string
		objs []client.Object
	}{
		{name: "no actions", objs: nil},
		{name: "the only URL templatization action is disabled", objs: []client.Object{utsUrlTemplatizationAction("url", true)}},
	} {
		for modeName, mode := range map[string]URLTemplatizationSyncMode{
			"create if missing": URLTemplatizationSyncCreateIfMissing,
			"apply full":        URLTemplatizationSyncApplyFull,
		} {
			t.Run(tc.name+", "+modeName, func(t *testing.T) {
				t.Setenv(consts.CurrentNamespaceEnvVar, psNamespace)

				var w utsWrites
				c := utsClient(t, &w, append(append([]client.Object{}, tc.objs...), utsExistingProcessor())...)

				require.NoError(t, SyncUrlTemplatizationProcessor(context.Background(), c, mode))

				require.Equal(t, []string{consts.URLTemplatizationProcessorName}, w.deleted)
				require.Empty(t, w.patched)

				err := c.Get(context.Background(), client.ObjectKey{
					Namespace: psNamespace, Name: consts.URLTemplatizationProcessorName,
				}, &odigosv1.Processor{})
				require.True(t, apierrors.IsNotFound(err), "processor should be gone, got %v", err)
			})
		}
	}
}

func TestSyncUrlTemplatizationProcessorToleratesAnAlreadyDeletedProcessor(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, psNamespace)

	var w utsWrites
	c := utsClient(t, &w)

	require.NoError(t, SyncUrlTemplatizationProcessor(context.Background(), c, URLTemplatizationSyncApplyFull))
	require.Equal(t, []string{consts.URLTemplatizationProcessorName}, w.deleted)
}

func TestSyncUrlTemplatizationProcessorPropagatesARealDeleteError(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, psNamespace)

	var w utsWrites
	c := utsClientWith(t, &w, interceptor.Funcs{
		Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
			return apierrors.NewForbidden(
				schema.GroupResource{Group: "odigos.io", Resource: "processors"},
				consts.URLTemplatizationProcessorName, errors.New("rbac denied"))
		},
	})

	err := SyncUrlTemplatizationProcessor(context.Background(), c, URLTemplatizationSyncApplyFull)
	require.Error(t, err, "only NotFound may be swallowed; an RBAC denial leaves a stale processor")
	require.True(t, apierrors.IsForbidden(err))
}

// TestSyncUrlTemplatizationProcessorHandlesTheNodeCollectorsGroup separates "the group does not
// exist yet" (normal during install, treat span metrics as off) from "we could not read it"
// (guessing the collector role would move the processor to the wrong collector).
func TestSyncUrlTemplatizationProcessorHandlesTheNodeCollectorsGroup(t *testing.T) {
	t.Run("missing group falls back to the gateway role", func(t *testing.T) {
		t.Setenv(consts.CurrentNamespaceEnvVar, psNamespace)

		var w utsWrites
		c := utsClient(t, &w, utsUrlTemplatizationAction("url", false))

		require.NoError(t, SyncUrlTemplatizationProcessor(context.Background(), c, URLTemplatizationSyncApplyFull))
		require.Len(t, w.patched, 1)

		want, err := buildUrlTemplatizationProcessor(psNamespace, false)
		require.NoError(t, err)
		require.JSONEq(t, psMarshal(t, want), w.patched[0])
	})

	t.Run("span metrics on applies the node collector role", func(t *testing.T) {
		t.Setenv(consts.CurrentNamespaceEnvVar, psNamespace)

		var w utsWrites
		c := utsClient(t, &w, utsUrlTemplatizationAction("url", false), utsNodeCollectorsGroup(true))

		require.NoError(t, SyncUrlTemplatizationProcessor(context.Background(), c, URLTemplatizationSyncApplyFull))
		require.Len(t, w.patched, 1)

		want, err := buildUrlTemplatizationProcessor(psNamespace, true)
		require.NoError(t, err)
		require.JSONEq(t, psMarshal(t, want), w.patched[0])
	})

	t.Run("a metrics block without span metrics keeps the gateway role", func(t *testing.T) {
		t.Setenv(consts.CurrentNamespaceEnvVar, psNamespace)

		// Host metrics without span metrics is the shape that separates "Metrics is set" from
		// "span metrics is on"; reading only the outer pointer would move the processor.
		cg := utsNodeCollectorsGroup(false)
		cg.Spec.Metrics = &odigosv1.CollectorsGroupMetricsCollectionSettings{
			HostMetrics: &common.MetricsSourceHostMetricsConfiguration{},
		}

		var w utsWrites
		c := utsClient(t, &w, utsUrlTemplatizationAction("url", false), cg)

		require.NoError(t, SyncUrlTemplatizationProcessor(context.Background(), c, URLTemplatizationSyncApplyFull))
		require.Len(t, w.patched, 1)

		want, err := buildUrlTemplatizationProcessor(psNamespace, false)
		require.NoError(t, err)
		require.JSONEq(t, psMarshal(t, want), w.patched[0])
	})

	t.Run("an unreadable group aborts without patching", func(t *testing.T) {
		t.Setenv(consts.CurrentNamespaceEnvVar, psNamespace)

		var w utsWrites
		c := utsClientWith(t, &w, interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*odigosv1.CollectorsGroup); ok {
					return apierrors.NewInternalError(errors.New("etcd timeout"))
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}, utsUrlTemplatizationAction("url", false))

		err := SyncUrlTemplatizationProcessor(context.Background(), c, URLTemplatizationSyncApplyFull)
		require.ErrorContains(t, err, "get node CollectorsGroup")
		require.Empty(t, w.patched, "the collector role must not be guessed from an unread group")
	})
}

// TestSharedProcessorReconcilerConvergesOnAnIdenticalPatch is the loop that made CORE-1716
// expensive, in its purest form: this reconciler is triggered by the Processor it writes, so an
// unstable body would never stop re-triggering itself.
func TestSharedProcessorReconcilerConvergesOnAnIdenticalPatch(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, psNamespace)

	var w utsWrites
	c := utsClient(t, &w,
		utsUrlTemplatizationAction("url", false),
		utsNodeCollectorsGroup(true),
		utsExistingProcessor())

	r := &SharedURLTemplatizationProcessorReconciler{Client: c}
	for i := 0; i < psRenders; i++ {
		_, err := r.Reconcile(context.Background(), ctrl.Request{})
		require.NoError(t, err)
	}

	require.Len(t, w.patched, psRenders)
	for i, got := range w.patched {
		require.Equalf(t, w.patched[0], got,
			"reconcile %d applied a different body; this reconciler watches the Processor it "+
				"writes, so a changing body never settles (CORE-1716)", i+1)
	}
}

func TestSharedProcessorReconcilersPropagateSyncFailures(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, psNamespace)

	newFailingClient := func() client.WithWatch {
		var w utsWrites
		return utsClientWith(t, &w, interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				return errors.New("api server unavailable")
			},
		})
	}

	_, err := (&SharedURLTemplatizationProcessorReconciler{Client: newFailingClient()}).
		Reconcile(context.Background(), ctrl.Request{})
	require.ErrorContains(t, err, "api server unavailable")

	_, err = (&URLTemplateNodeCGReconciler{Client: newFailingClient()}).
		Reconcile(context.Background(), ctrl.Request{})
	require.ErrorContains(t, err, "api server unavailable")
}

// TestURLTemplateNodeCGReconcilerAppliesTheNewRole pins why this reconciler uses ApplyFull: a span
// metrics toggle has to move an existing Processor between collectors, which CreateIfMissing would
// skip.
func TestURLTemplateNodeCGReconcilerAppliesTheNewRole(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, psNamespace)

	var w utsWrites
	c := utsClient(t, &w,
		utsUrlTemplatizationAction("url", false),
		utsNodeCollectorsGroup(true),
		utsExistingProcessor())

	_, err := (&URLTemplateNodeCGReconciler{Client: c}).Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)

	require.Len(t, w.patched, 1)
	want, err := buildUrlTemplatizationProcessor(psNamespace, true)
	require.NoError(t, err)
	require.JSONEq(t, psMarshal(t, want), w.patched[0])
}

// TestSpanMetricsTogglePredicate guards the filter in front of the reconciler above. It must pass
// exactly the updates that change the collector role: too narrow and the Processor keeps a stale
// role, too wide and every node CollectorsGroup write re-applies it.
func TestSpanMetricsTogglePredicate(t *testing.T) {
	p := urlTemplateNodeCGSpanMetricsTogglePredicate{}

	require.True(t, p.Create(event.CreateEvent{}))
	require.True(t, p.Delete(event.DeleteEvent{}))
	require.False(t, p.Generic(event.GenericEvent{}))

	for _, tc := range []struct {
		name   string
		oldOn  bool
		newOn  bool
		expect bool
	}{
		{name: "span metrics switched on", oldOn: false, newOn: true, expect: true},
		{name: "span metrics switched off", oldOn: true, newOn: false, expect: true},
		{name: "still off", oldOn: false, newOn: false, expect: false},
		{name: "still on", oldOn: true, newOn: true, expect: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expect, p.Update(event.UpdateEvent{
				ObjectOld: utsNodeCollectorsGroup(tc.oldOn),
				ObjectNew: utsNodeCollectorsGroup(tc.newOn),
			}))
		})
	}

	t.Run("a group with metrics settings but no span metrics counts as off", func(t *testing.T) {
		off := utsNodeCollectorsGroup(false)
		off.Spec.Metrics = &odigosv1.CollectorsGroupMetricsCollectionSettings{}
		require.False(t, p.Update(event.UpdateEvent{ObjectOld: utsNodeCollectorsGroup(false), ObjectNew: off}),
			"an allocated but empty Metrics block is not span metrics being enabled")
		require.True(t, p.Update(event.UpdateEvent{ObjectOld: off, ObjectNew: utsNodeCollectorsGroup(true)}))
	})

	t.Run("incomplete or foreign events", func(t *testing.T) {
		require.False(t, p.Update(event.UpdateEvent{ObjectNew: utsNodeCollectorsGroup(true)}))
		require.False(t, p.Update(event.UpdateEvent{ObjectOld: utsNodeCollectorsGroup(true)}))
		// Not a CollectorsGroup: the predicate cannot tell, so it must let the reconciler decide.
		require.True(t, p.Update(event.UpdateEvent{
			ObjectOld: utsExistingProcessor(), ObjectNew: utsExistingProcessor()}))
	})
}
