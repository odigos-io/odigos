package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/go-logr/logr"
	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonapi "github.com/odigos-io/odigos/common/api"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/loaders"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// K8sWorkloadContainer.urlTemplatization and
// K8sWorkloadContainerUrlTemplatization.liveTrafficLearning are not handed their
// parent objects by gqlgen: they walk graphql.GetFieldContext(ctx).Parent a
// hardcoded number of times and type-assert the ancestor's Result. The hop count
// is a contract against the SCHEMA SHAPE with no type-system backing — flipping
// containers between a list and an object, or wrapping either type in a new one,
// makes both resolvers fall into their "parent is not a workload" branch at
// runtime with no compile or schema error.

const (
	utcOdigosNamespace = "odigos-system"
	utcAppNamespace    = "app-ns"
	utcOtherNamespace  = "other-ns"
	utcWorkloadName    = "api"
	utcOtherWorkload   = "worker"
	utcContainerName   = "server"
	utcOtherContainer  = "sidecar"
)

// --- fixtures -------------------------------------------------------------

func utcScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(s))
	require.NoError(t, odigosv1.AddToScheme(s))
	return s
}

// utcTemplate is the template the fixture configures for one container. Every
// component of the container's identity is in it, so a resolver that reads the
// wrong workload or the wrong container reports a template that names it.
func utcTemplate(namespace, workloadName, containerName string) string {
	return fmt.Sprintf("/%s/%s/%s/{id}", namespace, workloadName, containerName)
}

func utcIC(namespace, workloadName string, containerNames ...string) *odigosv1.InstrumentationConfig {
	ic := &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "deployment-" + workloadName, Namespace: namespace},
	}
	for _, containerName := range containerNames {
		ic.Spec.WorkloadCollectorConfig = append(ic.Spec.WorkloadCollectorConfig, commonapi.ContainerCollectorConfig{
			ContainerName: containerName,
			BaseCollectorConfig: commonapi.BaseCollectorConfig{
				UrlTemplatization: &actionsapi.UrlTemplatizationConfig{
					Templates: []string{utcTemplate(namespace, workloadName, containerName)},
				},
			},
		})
	}
	return ic
}

// utcHarness primes loaders.For(ctx) off a fake cluster and points the
// package-level kube client at the same InstrumentationConfigs, which is what
// services.GetContainerUrlTemplatization reads. workloadsByIds only needs the
// effective-config ConfigMap and the InstrumentationConfig list, so no workload
// manifests, pods or dynamic client are involved.
func utcHarness(t *testing.T, ics ...*odigosv1.InstrumentationConfig) (context.Context, *Resolver, *odigosfake.Clientset) {
	t.Helper()

	t.Setenv(consts.CurrentNamespaceEnvVar, utcOdigosNamespace)

	objects := []client.Object{&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: utcOdigosNamespace},
		Data:       map[string]string{consts.OdigosConfigurationFileName: "configVersion: 1\n"},
	}}
	runtimeObjects := make([]runtime.Object, 0, len(ics))
	for _, ic := range ics {
		objects = append(objects, ic)
		runtimeObjects = append(runtimeObjects, ic)
	}

	clientset := odigosfake.NewSimpleClientset(runtimeObjects...)
	previous := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{OdigosClient: clientset.OdigosV1alpha1()})
	t.Cleanup(func() { kube.SetDefaultClient(previous) })

	c := fake.NewClientBuilder().WithScheme(utcScheme(t)).WithObjects(objects...).Build()
	ctx := loaders.WithLoaders(context.Background(), loaders.NewLoaders(logr.Discard(), c))
	return ctx, &Resolver{Logger: logr.Discard()}, clientset
}

const utcQuery = `query($ids: [K8sWorkloadIdInput!]!) {
  workloadsByIds(ids: $ids) {
    id { namespace kind name }
    containers {
      containerName
      urlTemplatization {
        templatization { templates { template } }
        liveTrafficLearning { unmatchedServerPaths { path count } }
      }
    }
  }
}`

func utcWorkloadID(namespace, name string) model.K8sWorkloadID {
	return model.K8sWorkloadID{Namespace: namespace, Kind: model.K8sResourceKindDeployment, Name: name}
}

func utcVariables(ids ...model.K8sWorkloadID) map[string]interface{} {
	out := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		out = append(out, map[string]interface{}{
			"namespace": id.Namespace,
			"kind":      string(id.Kind),
			"name":      id.Name,
		})
	}
	return map[string]interface{}{"ids": out}
}

// utcExecute runs the query through the REAL generated executor, which is the
// only way the parent field-context chain these resolvers walk gets built.
func utcExecute(t *testing.T, ctx context.Context, r *Resolver, around graphql.FieldMiddleware, ids ...model.K8sWorkloadID) map[string]interface{} {
	t.Helper()

	exec := executor.New(NewExecutableSchema(Config{Resolvers: r}))
	if around != nil {
		exec.AroundFields(around)
	}
	ctx = graphql.StartOperationTrace(ctx)
	opCtx, errs := exec.CreateOperationContext(ctx, &graphql.RawParams{Query: utcQuery, Variables: utcVariables(ids...)})
	require.Empty(t, errs)
	handler, ctx := exec.DispatchOperation(ctx, opCtx)
	res := handler(ctx)
	require.Empty(t, res.Errors)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(res.Data, &decoded))
	return decoded
}

// utcTemplatesByContainer flattens the response into
// "<namespace>/<kind>/<name>/<container>" -> the templates served for it.
func utcTemplatesByContainer(t *testing.T, decoded map[string]interface{}) map[string][]string {
	t.Helper()

	out := map[string][]string{}
	workloads, ok := decoded["workloadsByIds"].([]interface{})
	require.True(t, ok, "workloadsByIds is missing from %v", decoded)
	for _, rawWorkload := range workloads {
		workload := rawWorkload.(map[string]interface{})
		id := workload["id"].(map[string]interface{})
		containers, ok := workload["containers"].([]interface{})
		require.True(t, ok, "containers is missing from %v", workload)
		for _, rawContainer := range containers {
			container := rawContainer.(map[string]interface{})
			key := fmt.Sprintf("%s/%s/%s/%s", id["namespace"], id["kind"], id["name"], container["containerName"])

			templatization := container["urlTemplatization"].(map[string]interface{})["templatization"].(map[string]interface{})
			templates := []string{}
			for _, rawTemplate := range templatization["templates"].([]interface{}) {
				templates = append(templates, rawTemplate.(map[string]interface{})["template"].(string))
			}
			sort.Strings(templates)
			out[key] = templates
		}
	}
	return out
}

// utcRecordingHook stands in for the enterprise live-traffic implementation and
// records the identity each call was given.
type utcRecordingHook struct {
	mu    sync.Mutex
	calls []string
	err   error
}

func (h *utcRecordingHook) GetUrlTemplatizationLiveTrafficLearning(ctx context.Context, namespace, kind, name, containerName string) (*model.K8sWorkloadContainerLiveTrafficLearning, error) {
	if h.err != nil {
		return nil, h.err
	}
	key := fmt.Sprintf("%s/%s/%s/%s", namespace, kind, name, containerName)
	h.mu.Lock()
	h.calls = append(h.calls, key)
	h.mu.Unlock()
	return &model.K8sWorkloadContainerLiveTrafficLearning{
		UnmatchedServerPaths: []*model.UnmatchedURLPath{{Path: key, Count: 1}},
	}, nil
}

func (h *utcRecordingHook) recorded() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := append([]string(nil), h.calls...)
	sort.Strings(out)
	return out
}

// --- the hop counts against what gqlgen actually builds ---------------------

// utcAncestorDepths reports, for the field being resolved, how many .Parent hops
// away the nearest workload ancestor and the nearest container ancestor sit —
// measured from the live chain gqlgen built, not from reading the code.
func utcAncestorDepths(fc *graphql.FieldContext) map[string]int {
	depths := map[string]int{}
	hops := 0
	for p := fc.Parent; p != nil; p = p.Parent {
		hops++
		switch p.Result.(type) {
		case **model.K8sWorkload:
			if _, seen := depths["workload"]; !seen {
				depths["workload"] = hops
			}
		case **model.K8sWorkloadContainer:
			if _, seen := depths["container"]; !seen {
				depths["container"] = hops
			}
		}
	}
	return depths
}

// The two walkers hardcode: workload at 3 / container at 1 from urlTemplatization,
// and workload at 4 / container at 2 from liveTrafficLearning. Those six numbers
// are the whole contract, and this is what makes an incompatible schema edit a
// test failure instead of a blank drawer in the UI.
func TestTheHardcodedUrlTemplatizationAncestorDepthsMatchWhatGqlgenBuilds(t *testing.T) {
	ctx, r, _ := utcHarness(t, utcIC(utcAppNamespace, utcWorkloadName, utcContainerName))
	r.UrlTemplatizationLiveTrafficLearning = &utcRecordingHook{}

	var mu sync.Mutex
	observed := map[string]map[string]int{}
	utcExecute(t, ctx, r, func(ctx context.Context, next graphql.Resolver) (interface{}, error) {
		fc := graphql.GetFieldContext(ctx)
		switch fc.Field.Name {
		case "urlTemplatization", "liveTrafficLearning":
			mu.Lock()
			observed[fc.Field.Name] = utcAncestorDepths(fc)
			mu.Unlock()
		}
		return next(ctx)
	}, utcWorkloadID(utcAppNamespace, utcWorkloadName))

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, map[string]map[string]int{
		"urlTemplatization":   {"container": 1, "workload": 3},
		"liveTrafficLearning": {"container": 2, "workload": 4},
	}, observed)
}

// Every list in the path adds its own element field context between the list
// field and the object, which is why the two hop counts differ by one rather
// than by two. The ancestor Result is the DOUBLE pointer the generated
// marshalers store, which is the type both walkers assert on.
func TestTheWorkloadAndContainerAncestorsAreTheDoublePointersGqlgenStores(t *testing.T) {
	ctx, r, _ := utcHarness(t, utcIC(utcAppNamespace, utcWorkloadName, utcContainerName))

	var mu sync.Mutex
	var chain []string
	utcExecute(t, ctx, r, func(ctx context.Context, next graphql.Resolver) (interface{}, error) {
		fc := graphql.GetFieldContext(ctx)
		if fc.Field.Name == "urlTemplatization" {
			mu.Lock()
			for p := fc.Parent; p != nil; p = p.Parent {
				chain = append(chain, fmt.Sprintf("%T", p.Result))
			}
			mu.Unlock()
		}
		return next(ctx)
	}, utcWorkloadID(utcAppNamespace, utcWorkloadName))

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []string{
		"**model.K8sWorkloadContainer",
		"[]*model.K8sWorkloadContainer",
		"**model.K8sWorkload",
		"[]*model.K8sWorkload",
		"<nil>",
	}, chain)
}

// --- the walkers read their OWN workload and their OWN container ------------

// Four distinct (workload, container) pairs across two namespaces resolved in one
// operation. A walker that reads the first workload, the first container, or a
// fixed hop in the chain serves another pair's data, and both halves of the
// field — the templates from the InstrumentationConfig and the identity handed
// to the live-traffic hook — would say so.
func TestEveryContainerResolvesItsOwnWorkloadAndContainerThroughBothWalkers(t *testing.T) {
	ctx, r, _ := utcHarness(t,
		utcIC(utcAppNamespace, utcWorkloadName, utcContainerName, utcOtherContainer),
		utcIC(utcOtherNamespace, utcOtherWorkload, utcContainerName, utcOtherContainer),
	)
	hook := &utcRecordingHook{}
	r.UrlTemplatizationLiveTrafficLearning = hook

	decoded := utcExecute(t, ctx, r, nil,
		utcWorkloadID(utcAppNamespace, utcWorkloadName),
		utcWorkloadID(utcOtherNamespace, utcOtherWorkload),
	)

	require.Equal(t, map[string][]string{
		utcAppNamespace + "/Deployment/" + utcWorkloadName + "/" + utcContainerName:     {utcTemplate(utcAppNamespace, utcWorkloadName, utcContainerName)},
		utcAppNamespace + "/Deployment/" + utcWorkloadName + "/" + utcOtherContainer:    {utcTemplate(utcAppNamespace, utcWorkloadName, utcOtherContainer)},
		utcOtherNamespace + "/Deployment/" + utcOtherWorkload + "/" + utcContainerName:  {utcTemplate(utcOtherNamespace, utcOtherWorkload, utcContainerName)},
		utcOtherNamespace + "/Deployment/" + utcOtherWorkload + "/" + utcOtherContainer: {utcTemplate(utcOtherNamespace, utcOtherWorkload, utcOtherContainer)},
	}, utcTemplatesByContainer(t, decoded))

	require.Equal(t, []string{
		utcAppNamespace + "/Deployment/" + utcWorkloadName + "/" + utcContainerName,
		utcAppNamespace + "/Deployment/" + utcWorkloadName + "/" + utcOtherContainer,
		utcOtherNamespace + "/Deployment/" + utcOtherWorkload + "/" + utcContainerName,
		utcOtherNamespace + "/Deployment/" + utcOtherWorkload + "/" + utcOtherContainer,
	}, hook.recorded())
}

// --- the enterprise hook ---------------------------------------------------

// OSS leaves the hook unset, and the field must then resolve to null without the
// resolver walking the chain at all. The complementary half — the hook being
// called with the right identity — is covered above, so an unconditional
// `return nil, nil` cannot hide.
func TestLiveTrafficLearningIsNullWhenNoEnterpriseHookIsRegistered(t *testing.T) {
	ctx, r, _ := utcHarness(t, utcIC(utcAppNamespace, utcWorkloadName, utcContainerName))
	require.Nil(t, r.UrlTemplatizationLiveTrafficLearning)

	decoded := utcExecute(t, ctx, r, nil, utcWorkloadID(utcAppNamespace, utcWorkloadName))

	workload := decoded["workloadsByIds"].([]interface{})[0].(map[string]interface{})
	container := workload["containers"].([]interface{})[0].(map[string]interface{})
	templatization := container["urlTemplatization"].(map[string]interface{})
	require.Nil(t, templatization["liveTrafficLearning"])
	require.NotNil(t, templatization["templatization"], "the templatization half must still resolve in OSS")
}

func TestAFailingLiveTrafficHookFailsTheFieldInsteadOfReportingNoLearning(t *testing.T) {
	ctx, r, _ := utcHarness(t, utcIC(utcAppNamespace, utcWorkloadName, utcContainerName))
	r.UrlTemplatizationLiveTrafficLearning = &utcRecordingHook{err: fmt.Errorf("the cache is unreachable")}

	exec := executor.New(NewExecutableSchema(Config{Resolvers: r}))
	ctx = graphql.StartOperationTrace(ctx)
	opCtx, errs := exec.CreateOperationContext(ctx, &graphql.RawParams{
		Query:     utcQuery,
		Variables: utcVariables(utcWorkloadID(utcAppNamespace, utcWorkloadName)),
	})
	require.Empty(t, errs)
	handler, ctx := exec.DispatchOperation(ctx, opCtx)
	res := handler(ctx)

	require.Len(t, res.Errors, 1)
	require.Contains(t, res.Errors[0].Message, "the cache is unreachable")
}

// A container list that resolved fine from the loader cache while the live read
// of its InstrumentationConfig is denied must fail the field, not report the
// container as having no templatization — otherwise a cluster the UI cannot read
// looks exactly like a cluster with nothing configured, and the drawer offers to
// create rules that already exist.
func TestADeniedInstrumentationConfigReadFailsTheFieldInsteadOfEmptyingIt(t *testing.T) {
	ctx, r, clientset := utcHarness(t, utcIC(utcAppNamespace, utcWorkloadName, utcContainerName))
	clientset.PrependReactor("get", "instrumentationconfigs", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "odigos.io", Resource: "instrumentationconfigs"}, "deployment-"+utcWorkloadName, nil)
	})

	exec := executor.New(NewExecutableSchema(Config{Resolvers: r}))
	ctx = graphql.StartOperationTrace(ctx)
	opCtx, errs := exec.CreateOperationContext(ctx, &graphql.RawParams{
		Query:     utcQuery,
		Variables: utcVariables(utcWorkloadID(utcAppNamespace, utcWorkloadName)),
	})
	require.Empty(t, errs)
	handler, ctx := exec.DispatchOperation(ctx, opCtx)
	res := handler(ctx)

	require.Len(t, res.Errors, 1)
	require.Contains(t, res.Errors[0].Message, "get instrumentationconfig "+utcAppNamespace+"/deployment-"+utcWorkloadName)
}

// --- the branches the executor cannot reach --------------------------------

// utcFieldChain builds a parent chain by hand, OUTERMOST ancestor first; the last
// entry is the field context of the resolver itself.
func utcFieldChain(ctx context.Context, results ...interface{}) context.Context {
	for _, result := range results {
		ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{Result: result})
	}
	return ctx
}

func utcWorkloadResult(id *model.K8sWorkloadID) interface{} {
	w := &model.K8sWorkload{ID: id}
	return &w
}

func utcContainerResult(containerName string) interface{} {
	c := &model.K8sWorkloadContainer{ContainerName: containerName}
	return &c
}

// Both walkers refuse rather than panic or silently resolve the wrong workload
// when the chain is not the one the schema produces. The chain lengths here are
// exactly the ones the depth test above measured.
func TestTheUrlTemplatizationWalkersRejectAParentChainTheyCannotRead(t *testing.T) {
	id := utcWorkloadID(utcAppNamespace, utcWorkloadName)
	var nilContainer *model.K8sWorkloadContainer

	t.Run("workload id from the container field context", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			ctx     context.Context
			wantErr string
		}{
			{
				name:    "no field context at all",
				ctx:     context.Background(),
				wantErr: "missing parent resolver context",
			},
			{
				name:    "one hop short of the workload",
				ctx:     utcFieldChain(context.Background(), []*model.K8sWorkloadContainer{}, utcContainerResult(utcContainerName), nil),
				wantErr: "missing parent resolver context",
			},
			{
				name:    "the ancestor is a single pointer, not the double pointer gqlgen stores",
				ctx:     utcFieldChain(context.Background(), &model.K8sWorkload{ID: &id}, nil, nil, nil),
				wantErr: "parent is not a workload",
			},
			{
				name:    "the ancestor is a container, not a workload",
				ctx:     utcFieldChain(context.Background(), utcContainerResult(utcContainerName), nil, nil, nil),
				wantErr: "parent is not a workload",
			},
			{
				name:    "the workload pointer itself is nil",
				ctx:     utcFieldChain(context.Background(), (**model.K8sWorkload)(nil), nil, nil, nil),
				wantErr: "parent is not a workload",
			},
			{
				name:    "the workload carries no id",
				ctx:     utcFieldChain(context.Background(), utcWorkloadResult(nil), nil, nil, nil),
				wantErr: "parent is not a workload",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				got, err := workloadIDFromContainerFieldContext(tc.ctx)

				require.EqualError(t, err, tc.wantErr)
				require.Equal(t, model.K8sWorkloadID{}, got)
			})
		}

		accepted, err := workloadIDFromContainerFieldContext(
			utcFieldChain(context.Background(), utcWorkloadResult(&id), []*model.K8sWorkloadContainer{}, utcContainerResult(utcContainerName), nil))
		require.NoError(t, err)
		require.Equal(t, id, accepted)
	})

	t.Run("workload and container from the templatization field context", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			ctx     context.Context
			wantErr string
		}{
			{
				name:    "no field context at all",
				ctx:     context.Background(),
				wantErr: "missing parent resolver context",
			},
			{
				name:    "one hop short of the workload",
				ctx:     utcFieldChain(context.Background(), []*model.K8sWorkloadContainer{}, utcContainerResult(utcContainerName), nil, nil),
				wantErr: "missing parent resolver context",
			},
			{
				name:    "the container ancestor is a workload, not a container",
				ctx:     utcFieldChain(context.Background(), utcWorkloadResult(&id), nil, utcWorkloadResult(&id), nil, nil),
				wantErr: "parent is not a workload container",
			},
			{
				name:    "the container pointer itself is nil",
				ctx:     utcFieldChain(context.Background(), utcWorkloadResult(&id), nil, (**model.K8sWorkloadContainer)(nil), nil, nil),
				wantErr: "parent is not a workload container",
			},
			{
				name:    "the container behind the pointer is nil",
				ctx:     utcFieldChain(context.Background(), utcWorkloadResult(&id), nil, &nilContainer, nil, nil),
				wantErr: "parent is not a workload container",
			},
			{
				name:    "the container has no name",
				ctx:     utcFieldChain(context.Background(), utcWorkloadResult(&id), nil, utcContainerResult(""), nil, nil),
				wantErr: "parent is not a workload container",
			},
			{
				name:    "the container is readable but the workload ancestor is not",
				ctx:     utcFieldChain(context.Background(), utcContainerResult(utcContainerName), nil, utcContainerResult(utcContainerName), nil, nil),
				wantErr: "parent is not a workload",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				gotID, gotContainer, err := workloadContainerFromUrlTemplatizationFieldContext(tc.ctx)

				require.EqualError(t, err, tc.wantErr)
				require.Equal(t, model.K8sWorkloadID{}, gotID)
				require.Empty(t, gotContainer)
			})
		}

		acceptedID, acceptedContainer, err := workloadContainerFromUrlTemplatizationFieldContext(
			utcFieldChain(context.Background(), utcWorkloadResult(&id), []*model.K8sWorkloadContainer{}, utcContainerResult(utcContainerName), nil, nil))
		require.NoError(t, err)
		require.Equal(t, id, acceptedID)
		require.Equal(t, utcContainerName, acceptedContainer)
	})
}

// An unreadable container is reported before an unreadable workload, so the two
// error messages cannot be swapped.
func TestAnUnreadableContainerIsReportedBeforeAnUnreadableWorkload(t *testing.T) {
	ctx := utcFieldChain(context.Background(), utcContainerResult(utcContainerName), nil, utcContainerResult(""), nil, nil)

	_, _, err := workloadContainerFromUrlTemplatizationFieldContext(ctx)

	require.EqualError(t, err, "parent is not a workload container")
}

// containerNameFromFieldResult also accepts a SINGLE-pointer container result.
// gqlgen stores the double pointer for every list element (pinned above), so this
// branch is unreachable from the schema as it stands; it is kept honest here so
// that a schema change that starts producing it is not a silent behaviour change.
func TestASinglePointerContainerResultIsStillAccepted(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result interface{}
		want   string
		wantOk bool
	}{
		{name: "single pointer with a name", result: &model.K8sWorkloadContainer{ContainerName: utcContainerName}, want: utcContainerName, wantOk: true},
		{name: "single pointer without a name", result: &model.K8sWorkloadContainer{}},
		{name: "nil single pointer", result: (*model.K8sWorkloadContainer)(nil)},
		{name: "double pointer with a name", result: utcContainerResult(utcContainerName), want: utcContainerName, wantOk: true},
		{name: "nil double pointer", result: (**model.K8sWorkloadContainer)(nil)},
		{name: "not a container at all", result: "server"},
		{name: "no result", result: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := containerNameFromFieldResult(tc.result)

			assert.Equal(t, tc.wantOk, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- the resolvers' own guards ---------------------------------------------

func TestTheUrlTemplatizationResolverRefusesAContainerWithNoName(t *testing.T) {
	r := &Resolver{Logger: logr.Discard()}

	for _, obj := range []*model.K8sWorkloadContainer{nil, {}} {
		got, err := r.K8sWorkloadContainer().URLTemplatization(context.Background(), obj)

		require.EqualError(t, err, "container name is required")
		require.Nil(t, got)
	}
}

// The container name guard runs before the chain walk, so a named container with
// an unreadable parent reports the chain error instead.
func TestANamedContainerWithAnUnreadableParentReportsTheChainError(t *testing.T) {
	r := &Resolver{Logger: logr.Discard()}

	got, err := r.K8sWorkloadContainer().URLTemplatization(context.Background(), &model.K8sWorkloadContainer{ContainerName: utcContainerName})

	require.EqualError(t, err, "missing parent resolver context")
	require.Nil(t, got)
}

func TestTheLiveTrafficResolverWalksTheChainOnlyWhenAHookIsRegistered(t *testing.T) {
	withoutHook := &Resolver{Logger: logr.Discard()}
	got, err := withoutHook.K8sWorkloadContainerUrlTemplatization().LiveTrafficLearning(context.Background(), &model.K8sWorkloadContainerURLTemplatization{})
	require.NoError(t, err)
	require.Nil(t, got)

	withHook := &Resolver{Logger: logr.Discard(), UrlTemplatizationLiveTrafficLearning: &utcRecordingHook{}}
	got, err = withHook.K8sWorkloadContainerUrlTemplatization().LiveTrafficLearning(context.Background(), &model.K8sWorkloadContainerURLTemplatization{})
	require.EqualError(t, err, "missing parent resolver context")
	require.Nil(t, got)
}
