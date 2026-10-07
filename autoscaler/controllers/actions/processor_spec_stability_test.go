package actions

// The Action controller SSA-patches the Processor it owns and watches that Processor back through
// Owns(&odigosv1.Processor{}), so a rendered Spec that is not byte-stable re-triggers the
// controller forever. That is CORE-1716: renameAttributeConfig ranged a Go map, the Processor Spec
// changed on every reconcile and the autoscaler spun at ~100% CPU, resyncing the gateway
// ConfigMap/Deployment/Service/HPA with it.
//
// #5975 made renameAttributeConfig stable. These tests pin the same contract for every action
// type, for the two renderers that do not produce a per-action Processor CR, and end to end
// through ActionReconciler.Reconcile, which is where the loop actually closes.

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	actionscatalog "github.com/odigos-io/odigos/actions"
	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	apiactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/consts"
	actionstatus "github.com/odigos-io/odigos/status/action/generated"

	"github.com/stretchr/testify/require"
)

const psNamespace = "odigos-system"

// psRenders is the number of times every fixture is rendered. Go randomises the start offset of a
// map range, so a two-entry map lands on the same order about half the time; 100 renders puts a
// false "stable" verdict at 2^-99.
const psRenders = 100

// psCase pins one ActionSpec config field to the production path that turns it into collector
// state, together with a fixture whose collections all hold at least two entries — a one-entry
// collection cannot observe a map range.
type psCase struct {
	// renderer names the production function under test, for failure messages.
	renderer string
	action   *odigosv1.Action
	// wantProcessorTypes is the collector processor each render must produce, in order. Without it
	// the stability assertions would still hold for a renderer that quietly produced the wrong
	// processor, as long as it did so consistently.
	wantProcessorTypes []string
	// wantOrderHints are the literal OrderHints that place these processors in the gateway
	// pipeline, spelled out rather than read back from the config so that a changed hint has to be
	// re-justified here.
	wantOrderHints []int
	// render returns the Processors that reach the cluster. Two renders of the same fixture must
	// marshal identically.
	render func(t *testing.T, a *odigosv1.Action) []*odigosv1.Processor
}

// psCatalogOnce loads the embedded action catalog, which k8sutils/pkg/action consults to decide
// whether an action is a config extension. The catalog is a package-level global that each
// binary's main loads at startup; until it is loaded GetActionByType finds nothing and every
// action silently looks like a Processor-CR action, which would make the assertions below pass
// vacuously. It is loaded once per test binary and never unloaded, since a process-wide registry
// cannot be restored safely.
var psCatalogOnce sync.Once

func psRequireActionCatalogLoaded(t *testing.T) {
	t.Helper()
	psCatalogOnce.Do(func() { require.NoError(t, actionscatalog.Load()) })
	_, ok := actionscatalog.GetActionByType("PiiMasking")
	require.True(t, ok, "action catalog is not loaded; every config-extension assertion would be vacuous")
}

// psNotRendered lists ActionSpec config fields that no renderer in this repository consumes, with
// the reason. TestEveryActionSpecConfigFieldIsClassified fails when a field is in neither map.
var psNotRendered = map[string]string{
	"SpanRenamer": "applied to the agent as dynamic per-container config rather than through a " +
		"collector processor, so it is deliberately absent from the action catalog and from both " +
		"collector-side renderers",
}

func psScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, odigosv1.AddToScheme(s))
	return s
}

func psSignals() []common.ObservabilitySignal {
	// Three signals, because the K8sAttributes branch of convertActionToProcessor collects them
	// into a set: with one signal the range is stable whatever the code does.
	return []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.LogsObservabilitySignal,
	}
}

// psRenderProcessorCR drives convertActionToProcessor, the renderer behind the per-action
// Processor CR that Reconcile SSA-patches.
func psRenderProcessorCR(t *testing.T, a *odigosv1.Action) []*odigosv1.Processor {
	t.Helper()
	c := fake.NewClientBuilder().WithScheme(psScheme(t)).WithObjects(a.DeepCopy()).Build()
	proc, err := convertActionToProcessor(context.Background(), c, a)
	require.NoError(t, err)
	return []*odigosv1.Processor{proc}
}

// psRenderConfigExtension drives the config-extension path as the gateway consumes it. The
// conversion itself ranges a map and is deliberately unordered; FilterAndSortProcessorsByOrderHint
// is what makes the rendered gateway config stable, so the contract lives over the pair.
func psRenderConfigExtension(t *testing.T, a *odigosv1.Action) []*odigosv1.Processor {
	t.Helper()
	psRequireActionCatalogLoaded(t)
	list := odigosv1.ActionList{Items: []odigosv1.Action{*a.DeepCopy()}}
	converted := commonconf.ConvertActionsToConfigExtensionProcessors(list)
	return commonconf.FilterAndSortProcessorsByOrderHint(
		&odigosv1.ProcessorList{Items: converted}, odigosv1.CollectorsGroupRoleClusterGateway)
}

// psRenderSharedProcessor drives the shared URL-templatization Processor in both span-metrics
// states, since the two produce different collector roles.
func psRenderSharedProcessor(t *testing.T, _ *odigosv1.Action) []*odigosv1.Processor {
	t.Helper()
	off, err := buildUrlTemplatizationProcessor(psNamespace, false)
	require.NoError(t, err)
	on, err := buildUrlTemplatizationProcessor(psNamespace, true)
	require.NoError(t, err)
	return []*odigosv1.Processor{off, on}
}

func psCases() map[string]psCase {
	return map[string]psCase{
		"AddClusterInfo": {
			renderer: "convertActionToProcessor", render: psRenderProcessorCR,
			wantProcessorTypes: []string{"resource"},
			wantOrderHints:     []int{1},
			action: mkAction("add-cluster-info", odigosv1.ActionSpec{
				ActionName: "add cluster info", Signals: psSignals(),
				AddClusterInfo: &actionsv1.AddClusterInfoConfig{
					ClusterAttributes: []actionsv1.OtelAttributeWithValue{
						{AttributeName: "k8s.cluster.name", AttributeStringValue: stringPtr("prod-eu")},
						{AttributeName: "cloud.region", AttributeStringValue: stringPtr("eu-west-1")},
						{AttributeName: "deployment.environment", AttributeStringValue: stringPtr("production")},
					},
				},
			}),
		},
		"DeleteAttribute": {
			renderer: "convertActionToProcessor", render: psRenderProcessorCR,
			wantProcessorTypes: []string{"transform"},
			wantOrderHints:     []int{-100},
			action: mkAction("delete-attribute", odigosv1.ActionSpec{
				ActionName: "delete attribute", Signals: psSignals(),
				DeleteAttribute: &actionsv1.DeleteAttributeConfig{
					AttributeNamesToDelete: []string{"z.secret", "a.secret", "m.secret"},
				},
			}),
		},
		"RenameAttribute": {
			renderer: "convertActionToProcessor", render: psRenderProcessorCR,
			wantProcessorTypes: []string{"transform"},
			wantOrderHints:     []int{-50},
			action: mkAction("rename-attribute", odigosv1.ActionSpec{
				ActionName: "rename attribute", Signals: psSignals(),
				RenameAttribute: &actionsv1.RenameAttributeConfig{
					Renames: map[string]string{"z.attr": "z.new", "a.attr": "a.new", "m.attr": "m.new"},
				},
			}),
		},
		"ExtractAttribute": {
			renderer: "convertActionToProcessor", render: psRenderProcessorCR,
			wantProcessorTypes: []string{consts.OdigosExtractAttributeProcessorType},
			wantOrderHints:     []int{2},
			action: mkAction("extract-attribute", odigosv1.ActionSpec{
				ActionName: "extract attribute", Signals: psSignals(),
				ExtractAttribute: &apiactions.ExtractAttributeConfig{
					ExtractAttributeConfig: actionsapi.ExtractAttributeConfig{
						Extractions: []actionsapi.Extraction{
							{TargetAttributeName: "order.id", LookupKey: "orders", DataFormat: actionsapi.FormatResourcePath},
							{TargetAttributeName: "user.id", LookupKey: "user_id", DataFormat: actionsapi.FormatJSON},
							{TargetAttributeName: "tenant.id", Regex: `tenant=(\w+)`},
						},
					},
				},
			}),
		},
		"K8sAttributes": {
			renderer: "convertActionToProcessor", render: psRenderProcessorCR,
			wantProcessorTypes: []string{"k8sattributes"},
			wantOrderHints:     []int{0},
			action: mkAction("k8s-attributes", odigosv1.ActionSpec{
				ActionName: "k8s attributes", Signals: psSignals(),
				K8sAttributes: &actionsv1.K8sAttributesConfig{
					CollectContainerAttributes: true,
					CollectClusterUID:          true,
					LabelsAttributes: []actionsv1.K8sLabelAttribute{
						// z-label and b-label share the pod source, so sortByPrecedence has an
						// equal-precedence pair to tie-break on.
						{LabelKey: "z-label", AttributeKey: "z.label", FromSources: []actionsv1.K8sAttributeSource{actionsv1.PodAttributeSource, actionsv1.NamespaceAttributeSource}},
						{LabelKey: "b-label", AttributeKey: "b.label", FromSources: []actionsv1.K8sAttributeSource{actionsv1.PodAttributeSource}},
						{LabelKey: "a-label", AttributeKey: "a.label", FromSources: []actionsv1.K8sAttributeSource{actionsv1.NodeAttributeSource}},
					},
					AnnotationsAttributes: []actionsv1.K8sAnnotationAttribute{
						{AnnotationKey: "z-annotation", AttributeKey: "z.annotation", FromSources: []actionsv1.K8sAttributeSource{actionsv1.PodAttributeSource}},
						{AnnotationKey: "b-annotation", AttributeKey: "b.annotation", FromSources: []actionsv1.K8sAttributeSource{actionsv1.PodAttributeSource}},
						{AnnotationKey: "a-annotation", AttributeKey: "a.annotation", FromSources: []actionsv1.K8sAttributeSource{actionsv1.NamespaceAttributeSource}},
					},
				},
			}),
		},
		"PiiMasking": {
			renderer: "ConvertActionsToConfigExtensionProcessors", render: psRenderConfigExtension,
			wantProcessorTypes: []string{consts.OdigosPiiMaskingProcessorType},
			wantOrderHints:     []int{0},
			action: mkAction("pii-masking", odigosv1.ActionSpec{
				ActionName: "pii masking", Signals: psSignals(),
				PiiMasking: &apiactions.PiiMaskingConfig{
					PiiMaskingConfig: actionsapi.PiiMaskingConfig{
						PiiCategories: []actionsapi.PiiCategory{
							actionsapi.CreditCardMasking, actionsapi.EmailMasking, actionsapi.JwtMasking,
						},
					},
				},
			}),
		},
		"DbQueryTemplatization": {
			renderer: "ConvertActionsToConfigExtensionProcessors", render: psRenderConfigExtension,
			wantProcessorTypes: []string{consts.OdigosSQLQueryProcessorType},
			wantOrderHints:     []int{0},
			action: mkAction("db-query-templatization", odigosv1.ActionSpec{
				ActionName: "db query templatization", Signals: psSignals(),
				DbQueryTemplatization: &apiactions.DbQueryTemplatizationConfig{},
			}),
		},
		"InferDbAttributes": {
			renderer: "ConvertActionsToConfigExtensionProcessors", render: psRenderConfigExtension,
			wantProcessorTypes: []string{consts.OdigosSQLQueryProcessorType},
			wantOrderHints:     []int{0},
			action: mkAction("infer-db-attributes", odigosv1.ActionSpec{
				ActionName: "infer db attributes", Signals: psSignals(),
				InferDbAttributes: &apiactions.InferDbAttributesConfig{},
			}),
		},
		"URLTemplatization": {
			renderer: "buildUrlTemplatizationProcessor", render: psRenderSharedProcessor,
			wantProcessorTypes: []string{consts.OdigosURLTemplateProcessorType, consts.OdigosURLTemplateProcessorType},
			wantOrderHints:     []int{1, 1},
			action: mkAction("url-templatization", odigosv1.ActionSpec{
				ActionName: "url templatization", Signals: psSignals(),
				URLTemplatization: &apiactions.URLTemplatizationConfig{
					Rules: []apiactions.UrlTemplatizationRule{
						{Templates: []string{"/api/v1/orders/{id}", "/api/v1/users/{id}"}},
						{Templates: []string{"/health", "/ready"}},
					},
				},
			}),
		},
	}
}

func psMarshal(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return string(raw)
}

// TestEveryActionSpecConfigFieldIsClassified fails when a new action config is added to ActionSpec
// without either a stability fixture or an explicit "nothing renders this" entry. Without it a new
// action type can reintroduce CORE-1716 with no test noticing.
func TestEveryActionSpecConfigFieldIsClassified(t *testing.T) {
	specType := reflect.TypeOf(odigosv1.ActionSpec{})
	require.Positive(t, specType.NumField())

	cases := psCases()
	configFields := 0
	for i := 0; i < specType.NumField(); i++ {
		f := specType.Field(i)
		// Every action config is a pointer to a struct; ActionName/Notes/Disabled/Signals are not.
		if f.Type.Kind() != reflect.Ptr || f.Type.Elem().Kind() != reflect.Struct {
			continue
		}
		configFields++
		_, covered := cases[f.Name]
		reason, excused := psNotRendered[f.Name]
		require.Falsef(t, covered && excused,
			"ActionSpec.%s is both rendered and listed as not rendered", f.Name)
		require.Truef(t, covered || excused,
			"ActionSpec.%s has no stability fixture in psCases() and no entry in psNotRendered. "+
				"Add the renderer that turns it into collector state, or document why nothing does.", f.Name)
		if excused {
			require.NotEmptyf(t, reason, "psNotRendered[%q] must explain itself", f.Name)
		}
	}

	require.Equal(t, len(cases)+len(psNotRendered), configFields,
		"psCases()/psNotRendered hold entries that are not ActionSpec config fields")
	require.GreaterOrEqual(t, configFields, 10, "ActionSpec lost action configs unexpectedly")
}

// TestStabilityFixturesCarryMultiValuedCollections is the anti-vacuity guard for the stability
// test below: a fixture whose collections hold a single entry renders identically whether or not
// the production code ranges a map, so weakening a fixture would silently disarm the gate.
func TestStabilityFixturesCarryMultiValuedCollections(t *testing.T) {
	for name, tc := range psCases() {
		t.Run(name, func(t *testing.T) {
			require.GreaterOrEqual(t, len(tc.action.Spec.Signals), 2,
				"the K8sAttributes renderer collects signals into a set; fixtures need at least two")

			cfg := reflect.ValueOf(tc.action.Spec).FieldByName(name)
			require.False(t, cfg.IsNil(), "fixture for %s does not set Spec.%s", name, name)
			psRequireCollectionsAreMultiValued(t, cfg.Elem(), "Spec."+name)
		})
	}
}

// psRequireCollectionsAreMultiValued walks a config struct and its inlined embedded structs,
// requiring every populated slice or map to hold at least two entries.
func psRequireCollectionsAreMultiValued(t *testing.T, v reflect.Value, path string) {
	t.Helper()
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		fv := v.Field(i)
		fieldPath := path + "." + f.Name
		switch fv.Kind() {
		case reflect.Slice, reflect.Map:
			if fv.Len() > 0 {
				require.GreaterOrEqualf(t, fv.Len(), 2,
					"%s holds one entry; a single-entry collection cannot observe a map range", fieldPath)
			}
		case reflect.Struct:
			if f.Anonymous {
				psRequireCollectionsAreMultiValued(t, fv, path)
			}
		}
	}
}

// TestRenderedActionStateIsStableForEveryActionType is the CORE-1716 gate: rendering the same
// Action repeatedly must produce byte-identical collector state.
func TestRenderedActionStateIsStableForEveryActionType(t *testing.T) {
	for name, tc := range psCases() {
		t.Run(name, func(t *testing.T) {
			procs := tc.render(t, tc.action)
			// Without this an action the renderer silently drops would compare equal to itself
			// on every iteration and report the contract as held.
			require.NotEmptyf(t, procs, "%s produced no Processor for ActionSpec.%s", tc.renderer, name)
			gotTypes := make([]string, 0, len(procs))
			gotHints := make([]int, 0, len(procs))
			for _, p := range procs {
				gotTypes = append(gotTypes, p.Spec.Type)
				gotHints = append(gotHints, p.Spec.OrderHint)
			}
			require.Equal(t, tc.wantProcessorTypes, gotTypes,
				"ActionSpec.%s rendered the wrong collector processor", name)
			require.Equal(t, tc.wantOrderHints, gotHints,
				"ActionSpec.%s would be placed elsewhere in the collector pipeline", name)

			first := psMarshal(t, procs)
			for i := 1; i < psRenders; i++ {
				require.Equalf(t, first, psMarshal(t, tc.render(t, tc.action)),
					"%s rendered ActionSpec.%s differently on render %d; an unstable Spec makes the "+
						"Action controller SSA-patch and re-trigger itself forever (CORE-1716)",
					tc.renderer, name, i+1)
			}
		})
	}
}

// TestReconcileSendsAnIdenticalProcessorPatchEveryTime closes the loop the way production does:
// ActionReconciler.Reconcile builds the Processor and SSA-applies it, and Owns(Processor) feeds
// the resulting update straight back in. Identical patch bodies are what stops the spin.
func TestReconcileSendsAnIdenticalProcessorPatchEveryTime(t *testing.T) {
	for name, tc := range psCases() {
		if tc.renderer != "convertActionToProcessor" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv(consts.CurrentNamespaceEnvVar, psNamespace)

			var payloads []string
			recordPatch := interceptor.Funcs{
				Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
					proc, ok := obj.(*odigosv1.Processor)
					if !ok {
						return c.Patch(ctx, obj, patch, opts...)
					}
					data, err := patch.Data(proc)
					if err != nil {
						return err
					}
					payloads = append(payloads, string(data))
					// Short-circuit: the fake client does not implement server-side apply, and the
					// contract under test is the payload the controller sends, not the merge.
					return nil
				},
			}

			action := tc.action.DeepCopy()
			c := fake.NewClientBuilder().
				WithScheme(psScheme(t)).
				WithObjects(action).
				WithStatusSubresource(action).
				WithInterceptorFuncs(recordPatch).
				Build()
			r := &ActionReconciler{Client: c}
			req := ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: action.Namespace, Name: action.Name}}

			for i := 0; i < psRenders; i++ {
				_, err := r.Reconcile(context.Background(), req)
				require.NoError(t, err)
			}

			require.Len(t, payloads, psRenders, "every reconcile must apply the Processor")
			for i, got := range payloads {
				require.Equalf(t, payloads[0], got,
					"reconcile %d applied a different Processor body for %s; the Owns(Processor) "+
						"watch turns that into an endless reconcile loop (CORE-1716)", i+1, name)
			}
		})
	}
}

// TestConfigExtensionProcessorsAreStableAcrossProcessorTypes covers the composition that renders
// the gateway: ConvertActionsToConfigExtensionProcessors ranges a map, so a cluster holding both a
// PII-masking and a SQL-query config extension only renders stably because
// FilterAndSortProcessorsByOrderHint tie-breaks equal OrderHints on name.
func TestConfigExtensionProcessorsAreStableAcrossProcessorTypes(t *testing.T) {
	psRequireActionCatalogLoaded(t)
	list := odigosv1.ActionList{Items: []odigosv1.Action{
		*mkAction("pii", odigosv1.ActionSpec{Signals: psSignals(),
			PiiMasking: &apiactions.PiiMaskingConfig{PiiMaskingConfig: actionsapi.PiiMaskingConfig{
				PiiCategories: []actionsapi.PiiCategory{actionsapi.EmailMasking}}}}),
		*mkAction("sql", odigosv1.ActionSpec{Signals: psSignals(),
			DbQueryTemplatization: &apiactions.DbQueryTemplatizationConfig{}}),
	}}

	var first string
	for i := 0; i < psRenders; i++ {
		sorted := commonconf.FilterAndSortProcessorsByOrderHint(
			&odigosv1.ProcessorList{Items: commonconf.ConvertActionsToConfigExtensionProcessors(list)},
			odigosv1.CollectorsGroupRoleClusterGateway)
		names := make([]string, 0, len(sorted))
		for _, p := range sorted {
			names = append(names, p.Name)
		}
		require.Lenf(t, names, 2, "expected one processor per distinct config-extension type, got %v", names)
		if i == 0 {
			first = fmt.Sprint(names)
			continue
		}
		require.Equal(t, first, fmt.Sprint(names),
			"the rendered gateway processor order changed between identical inputs")
	}
	require.Equal(t,
		fmt.Sprint([]string{consts.OdigosPiiMaskingProcessorType, consts.OdigosSQLQueryProcessorType}),
		first)
}

// TestFilterAndSortProcessorsByOrderHintTieBreaksOnName pins the tie-break the comment in
// processors.go promises. Feeding the same two processors in both input orders is what catches a
// sort that falls back to input order, which is where the config-extension map range would leak
// into the rendered gateway config.
func TestFilterAndSortProcessorsByOrderHintTieBreaksOnName(t *testing.T) {
	proc := func(name string, orderHint int, roles ...odigosv1.CollectorsGroupRole) odigosv1.Processor {
		return odigosv1.Processor{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: psNamespace},
			Spec:       odigosv1.ProcessorSpec{OrderHint: orderHint, CollectorRoles: roles},
		}
	}
	gateway := odigosv1.CollectorsGroupRoleClusterGateway

	forward := []odigosv1.Processor{proc("aaa", 0, gateway), proc("zzz", 0, gateway)}
	reversed := []odigosv1.Processor{proc("zzz", 0, gateway), proc("aaa", 0, gateway)}

	names := func(in []odigosv1.Processor) []string {
		sorted := commonconf.FilterAndSortProcessorsByOrderHint(&odigosv1.ProcessorList{Items: in}, gateway)
		out := make([]string, 0, len(sorted))
		for _, p := range sorted {
			out = append(out, p.Name)
		}
		return out
	}

	require.Equal(t, []string{"aaa", "zzz"}, names(forward))
	require.Equal(t, []string{"aaa", "zzz"}, names(reversed),
		"equal OrderHints must sort by name, not by the order the list happened to arrive in")

	// OrderHint still wins over the name tie-break.
	require.Equal(t, []string{"zzz", "aaa"},
		names([]odigosv1.Processor{proc("aaa", 5, gateway), proc("zzz", -5, gateway)}))

	// A processor that does not serve this role is dropped, and a disabled one is dropped too.
	nodeOnly := proc("node-only", 0, odigosv1.CollectorsGroupRoleNodeCollector)
	disabled := proc("disabled", 0, gateway)
	disabled.Spec.Disabled = true
	require.Equal(t, []string{"aaa"},
		names([]odigosv1.Processor{proc("aaa", 0, gateway), nodeOnly, disabled}))
}

// TestReconcileDoesNotCreateAProcessorCRForConfigExtensionActions pins the routing decision that
// keeps the two renderers apart. Config-extension actions reach the collector through the gateway
// config, so reconciling one must not also apply a Processor CR, and the TransformedToProcessor
// condition must not be left behind claiming it did.
func TestReconcileDoesNotCreateAProcessorCRForConfigExtensionActions(t *testing.T) {
	psRequireActionCatalogLoaded(t)

	for name, tc := range psCases() {
		if tc.renderer != "ConvertActionsToConfigExtensionProcessors" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv(consts.CurrentNamespaceEnvVar, psNamespace)

			action := tc.action.DeepCopy()
			action.Status.Conditions = []metav1.Condition{{
				Type:               actionstatus.TransformedToProcessorType,
				Status:             metav1.ConditionTrue,
				Reason:             actionstatus.TransformedToProcessorProcessorCreated.Name,
				Message:            "stale",
				LastTransitionTime: metav1.Now(),
			}}

			var w utsWrites
			c := utsClient(t, &w, action)
			r := &ActionReconciler{Client: c}

			_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{
				Namespace: action.Namespace, Name: action.Name}})
			require.NoError(t, err)

			require.Empty(t, w.patched, "a config-extension action must not apply a Processor CR")

			var reconciled odigosv1.Action
			require.NoError(t, c.Get(context.Background(), client.ObjectKey{
				Namespace: action.Namespace, Name: action.Name}, &reconciled))
			require.Nil(t,
				meta.FindStatusCondition(reconciled.Status.Conditions, actionstatus.TransformedToProcessorType),
				"the stale TransformedToProcessor condition must be cleared")
		})
	}
}
