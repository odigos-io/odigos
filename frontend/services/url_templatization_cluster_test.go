package services

import (
	"context"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	actionsv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	commonapi "github.com/odigos-io/odigos/common/api"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/api/agentsignalconfig"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/urltemplate"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"
)

// The workload every test in this file resolves. The InstrumentationConfig is
// stored under the runtime object name, not under the workload name — the two
// are deliberately different so a resolver reading the wrong one is visible.
const (
	utzNamespace     = "app-ns"
	utzKind          = "Deployment"
	utzWorkload      = "api"
	utzICName        = "deployment-api"
	utzContainer     = "server"
	utzOtherContaine = "proxy"
)

// --- fixtures -------------------------------------------------------------

// utzCluster points the package-level kube client at a fake odigos clientset
// holding the given objects, and pins the namespace the Actions are read from.
func utzCluster(t *testing.T, objects ...runtime.Object) *odigosfake.Clientset {
	t.Helper()

	t.Setenv(consts.CurrentNamespaceEnvVar, consts.DefaultOdigosNamespace)

	clientset := odigosfake.NewSimpleClientset(objects...)
	previous := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{OdigosClient: clientset.OdigosV1alpha1()})
	t.Cleanup(func() { kube.SetDefaultClient(previous) })
	return clientset
}

func utzResolve(t *testing.T, containerName string) (*model.LiveTrafficLearningTemplatization, []urltemplate.PathRule) {
	t.Helper()

	templatization, rules, err := GetContainerUrlTemplatization(context.Background(), utzNamespace, utzKind, utzWorkload, containerName)
	require.NoError(t, err)
	require.NotNil(t, templatization)
	return templatization, rules
}

func utzIC(mutators ...func(*odigosv1.InstrumentationConfig)) *odigosv1.InstrumentationConfig {
	ic := &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: utzICName, Namespace: utzNamespace},
	}
	for _, mutate := range mutators {
		mutate(ic)
	}
	return ic
}

// utzCollectorConfig puts a templatization config on the workload COLLECTOR half
// of the InstrumentationConfig spec.
func utzCollectorConfig(containerName string, templates ...string) func(*odigosv1.InstrumentationConfig) {
	return func(ic *odigosv1.InstrumentationConfig) {
		ic.Spec.WorkloadCollectorConfig = append(ic.Spec.WorkloadCollectorConfig, commonapi.ContainerCollectorConfig{
			ContainerName: containerName,
			SourceCollectorConfig: commonapi.SourceCollectorConfig{
				UrlTemplatization: &actionsapi.UrlTemplatizationConfig{Templates: templates},
			},
		})
	}
}

// utzAgentConfig puts a templatization config on the per-container AGENT traces
// half of the InstrumentationConfig spec.
func utzAgentConfig(containerName string, templates ...string) func(*odigosv1.InstrumentationConfig) {
	return func(ic *odigosv1.InstrumentationConfig) {
		ic.Spec.Containers = append(ic.Spec.Containers, odigosv1.ContainerAgentConfig{
			ContainerName: containerName,
			Traces: &agentsignalconfig.AgentTracesConfig{
				UrlTemplatization: &actionsapi.UrlTemplatizationConfig{Templates: templates},
			},
		})
	}
}

// utzContainerWithoutTemplatization adds a container that carries no
// templatization at all on either half, under either an absent Traces signal or
// a Traces signal with no templatization in it.
func utzContainerWithoutTemplatization(containerName string, withTraces bool) func(*odigosv1.InstrumentationConfig) {
	return func(ic *odigosv1.InstrumentationConfig) {
		container := odigosv1.ContainerAgentConfig{ContainerName: containerName}
		if withTraces {
			container.Traces = &agentsignalconfig.AgentTracesConfig{}
		}
		ic.Spec.Containers = append(ic.Spec.Containers, container)
		ic.Spec.WorkloadCollectorConfig = append(ic.Spec.WorkloadCollectorConfig, commonapi.ContainerCollectorConfig{
			ContainerName: containerName,
		})
	}
}

func utzDetectedLanguage(containerName string, language common.ProgrammingLanguage) func(*odigosv1.InstrumentationConfig) {
	return func(ic *odigosv1.InstrumentationConfig) {
		ic.Status.RuntimeDetailsByContainer = append(ic.Status.RuntimeDetailsByContainer, odigosv1.RuntimeDetailsByContainer{
			ContainerName: containerName,
			Language:      language,
		})
	}
}

func utzAction(name string, rules ...actionsv1.UrlTemplatizationRule) *odigosv1.Action {
	return &odigosv1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: consts.DefaultOdigosNamespace,
			Labels:    map[string]string{k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosLiveTrafficLearningManagedByValue},
		},
		Spec: odigosv1.ActionSpec{
			ActionName:        "action display " + name,
			URLTemplatization: &actionsv1.URLTemplatizationConfig{Rules: rules},
		},
	}
}

func utzTemplateStrings(templates []*model.LiveTrafficLearningTemplate) []string {
	out := make([]string, 0, len(templates))
	for _, template := range templates {
		out = append(out, template.Template)
	}
	return out
}

func utzTemplate(t *testing.T, templatization *model.LiveTrafficLearningTemplatization, template string) *model.LiveTrafficLearningTemplate {
	t.Helper()

	for _, candidate := range templatization.Templates {
		if candidate.Template == template {
			return candidate
		}
	}
	t.Fatalf("template %q is not in %v", template, utzTemplateStrings(templatization.Templates))
	return nil
}

// --- which half of the InstrumentationConfig the container config comes from --

// The collector half and the agent half can both carry a templatization config
// for the same container, and they can disagree. Nothing but the order of the
// two loops decides which one the UI reports.
func TestTheWorkloadCollectorTemplatizationWinsOverThePerContainerAgentOne(t *testing.T) {
	ic := utzIC(
		utzCollectorConfig(utzContainer, "/from-the-collector/{id}"),
		utzAgentConfig(utzContainer, "/from-the-agent/{id}"),
	)

	byContainer := collectUrlTemplatizationConfigsByContainer(ic)

	require.Equal(t, map[string]*actionsapi.UrlTemplatizationConfig{
		utzContainer: {Templates: []string{"/from-the-collector/{id}"}},
	}, byContainer)
}

// Each container must be looked up in both halves on its own: one container
// configured only on the collector half and another configured only on the
// agent half must both be found, with the config that belongs to it.
func TestEachContainerResolvesItsOwnTemplatizationFromEitherHalf(t *testing.T) {
	ic := utzIC(
		utzCollectorConfig(utzContainer, "/collector-only/{id}"),
		utzAgentConfig(utzOtherContaine, "/agent-only/{id}"),
	)

	byContainer := collectUrlTemplatizationConfigsByContainer(ic)

	require.Equal(t, map[string]*actionsapi.UrlTemplatizationConfig{
		utzContainer:     {Templates: []string{"/collector-only/{id}"}},
		utzOtherContaine: {Templates: []string{"/agent-only/{id}"}},
	}, byContainer)
}

// A container that carries no templatization must be ABSENT from the map rather
// than present with a nil config — GetContainerUrlTemplatization distinguishes
// the two only by the nil check on the looked-up value.
func TestAContainerWithoutTemplatizationIsAbsentFromTheConfigMap(t *testing.T) {
	for _, withTraces := range []bool{false, true} {
		ic := utzIC(utzContainerWithoutTemplatization(utzContainer, withTraces))

		byContainer := collectUrlTemplatizationConfigsByContainer(ic)

		require.Empty(t, byContainer, "withTraces=%v", withTraces)
	}
}

// --- the InstrumentationConfig read ---------------------------------------

// Without a cluster client the resolver must still answer with an empty, non-nil
// template list: the GraphQL field is `templates: [LiveTrafficLearningTemplate!]!`,
// so a nil list would serve null for a non-null field.
func TestTheTemplatizationIsEmptyButNotNullWithoutAClusterClient(t *testing.T) {
	previous := kube.DefaultClient
	t.Cleanup(func() { kube.SetDefaultClient(previous) })

	for name, client := range map[string]*kube.Client{
		"no client at all":            nil,
		"a client with no odigos API": {},
	} {
		kube.SetDefaultClient(client)

		templatization, rules, err := GetContainerUrlTemplatization(context.Background(), utzNamespace, utzKind, utzWorkload, utzContainer)

		require.NoError(t, err, name)
		require.NotNil(t, templatization, name)
		require.NotNil(t, templatization.Templates, name)
		require.Empty(t, templatization.Templates, name)
		require.Nil(t, rules, name)
	}
}

// An uninstrumented workload has no InstrumentationConfig at all. That is normal
// cluster state, not a failure, and must not fail the GraphQL field.
func TestAMissingInstrumentationConfigResolvesToAnEmptyTemplatization(t *testing.T) {
	utzCluster(t)

	templatization, rules, err := GetContainerUrlTemplatization(context.Background(), utzNamespace, utzKind, utzWorkload, utzContainer)

	require.NoError(t, err)
	require.NotNil(t, templatization.Templates)
	require.Empty(t, templatization.Templates)
	require.Nil(t, rules)
}

// Any other read failure — an RBAC denial, an unreachable API server — must NOT
// be collapsed into "this workload has no templatization", or a cluster the UI
// cannot read looks like a cluster with nothing configured.
func TestAFailedInstrumentationConfigReadIsReportedAsAnError(t *testing.T) {
	clientset := utzCluster(t)
	clientset.PrependReactor("get", "instrumentationconfigs", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "odigos.io", Resource: "instrumentationconfigs"}, utzICName, nil)
	})

	templatization, rules, err := GetContainerUrlTemplatization(context.Background(), utzNamespace, utzKind, utzWorkload, utzContainer)

	require.Error(t, err)
	require.Nil(t, templatization)
	require.Nil(t, rules)
	// one ordered phrase, so swapping the namespace and the name is visible
	require.ErrorContains(t, err, "get instrumentationconfig "+utzNamespace+"/"+utzICName)
	require.True(t, apierrors.IsForbidden(err), "the k8s error must stay unwrappable: %v", err)
}

// The InstrumentationConfig is named after the workload RUNTIME object, not after
// the workload. A decoy object under the bare workload name, holding different
// templates, proves which one is read.
func TestTheInstrumentationConfigIsReadUnderTheWorkloadRuntimeObjectName(t *testing.T) {
	decoy := &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: utzWorkload, Namespace: utzNamespace},
	}
	utzCollectorConfig(utzContainer, "/from-the-decoy/{id}")(decoy)
	utzCluster(t, utzIC(utzCollectorConfig(utzContainer, "/from-the-real-config/{id}")), decoy)

	templatization, _ := utzResolve(t, utzContainer)

	require.Equal(t, []string{"/from-the-real-config/{id}"}, utzTemplateStrings(templatization.Templates))
}

// A container that is in the workload but has no templatization config of its own
// must not inherit a sibling container's templates.
func TestAContainerWithNoTemplatizationDoesNotInheritItsSiblings(t *testing.T) {
	utzCluster(t, utzIC(
		utzCollectorConfig(utzContainer, "/only-on-the-server/{id}"),
		utzContainerWithoutTemplatization(utzOtherContaine, true),
	))

	templatization, rules := utzResolve(t, utzOtherContaine)

	require.Empty(t, templatization.Templates)
	require.Nil(t, rules)
	require.Nil(t, templatization.PatternBasedTemplatization)
}

// --- which Actions are considered -----------------------------------------

// Three of these four Actions must be ignored, each for exactly one reason, so
// dropping any single filter is visible.
func TestOnlyEnabledUrlTemplatizationActionsContributeTemplates(t *testing.T) {
	rules := []actionsv1.UrlTemplatizationRule{{Templates: []string{"/from-this-action/{id}"}}}

	enabled := utzAction("enabled", rules...)

	disabled := utzAction("disabled", actionsv1.UrlTemplatizationRule{Templates: []string{"/from-a-disabled-action/{id}"}})
	disabled.Spec.Disabled = true

	otherType := utzAction("other-type", actionsv1.UrlTemplatizationRule{Templates: []string{"/from-another-action-type/{id}"}})
	otherType.Spec.URLTemplatization = nil
	otherType.Spec.PiiMasking = &actionsv1.PiiMaskingConfig{}

	disabledOtherType := utzAction("disabled-other-type")
	disabledOtherType.Spec.Disabled = true
	disabledOtherType.Spec.URLTemplatization = nil

	utzCluster(t, utzIC(utzCollectorConfig(utzContainer)), enabled, disabled, otherType, disabledOtherType)

	considered := listUrlTemplatizationActions(context.Background())

	names := make([]string, 0, len(considered))
	for _, action := range considered {
		names = append(names, action.Name)
	}
	require.Equal(t, []string{"enabled"}, names)

	templatization, _ := utzResolve(t, utzContainer)
	require.Equal(t, []string{"/from-this-action/{id}"}, utzTemplateStrings(templatization.Templates))
}

// Actions live in the odigos namespace, not in the workload's namespace. An
// Action sitting next to the workload must not be picked up.
func TestActionsAreReadFromTheOdigosNamespaceAndNotTheWorkloadNamespace(t *testing.T) {
	nextToTheWorkload := utzAction("next-to-the-workload", actionsv1.UrlTemplatizationRule{Templates: []string{"/from-the-workload-namespace/{id}"}})
	nextToTheWorkload.Namespace = utzNamespace

	utzCluster(t, utzIC(utzCollectorConfig(utzContainer)), nextToTheWorkload)

	require.Empty(t, listUrlTemplatizationActions(context.Background()))

	templatization, _ := utzResolve(t, utzContainer)
	require.Empty(t, templatization.Templates)
}

// listUrlTemplatizationActions swallows a failed List and returns no actions, so
// a transient API-server error silently strips the ownership metadata off every
// template while still reporting the templates themselves. Pinned, not endorsed:
// the UI then shows an Action-managed template as unmanaged and editable.
func TestAFailedActionsListStripsOwnershipInsteadOfFailingTheQuery(t *testing.T) {
	action := utzAction("owner", actionsv1.UrlTemplatizationRule{Templates: []string{"/users/{id}"}})
	clientset := utzCluster(t, utzIC(utzCollectorConfig(utzContainer, "/users/{id}")), action)

	healthy, _ := utzResolve(t, utzContainer)
	require.Equal(t, "owner", *utzTemplate(t, healthy, "/users/{id}").ActionID)
	require.Equal(t, model.ManagedByLiveTrafficLearning, utzTemplate(t, healthy, "/users/{id}").ManagedBy)

	clientset.PrependReactor("list", "actions", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewServiceUnavailable("the apiserver is down")
	})

	require.Nil(t, listUrlTemplatizationActions(context.Background()))

	degraded, _ := utzResolve(t, utzContainer)
	require.Equal(t, []string{"/users/{id}"}, utzTemplateStrings(degraded.Templates))
	require.Nil(t, utzTemplate(t, degraded, "/users/{id}").ActionID)
	require.Equal(t, model.ManagedByUnknown, utzTemplate(t, degraded, "/users/{id}").ManagedBy)
}

// --- per-container language scoping ----------------------------------------

// An Action rule scoped to a language must be matched against the language
// detected for THAT container, not against the first language in the workload.
func TestAnActionLanguageScopeIsMatchedAgainstTheResolvedContainersOwnLanguage(t *testing.T) {
	goOnly := utzAction("go-only", actionsv1.UrlTemplatizationRule{
		Scopes:    &k8sconsts.SourcesScopes{Languages: []common.ProgrammingLanguage{common.GoProgrammingLanguage}},
		Templates: []string{"/go-only/{id}"},
	})

	utzCluster(t, utzIC(
		utzCollectorConfig(utzContainer),
		utzCollectorConfig(utzOtherContaine),
		utzDetectedLanguage(utzContainer, common.GoProgrammingLanguage),
		utzDetectedLanguage(utzOtherContaine, common.JavaProgrammingLanguage),
	), goOnly)

	goContainer, _ := utzResolve(t, utzContainer)
	require.Equal(t, []string{"/go-only/{id}"}, utzTemplateStrings(goContainer.Templates))

	javaContainer, _ := utzResolve(t, utzOtherContaine)
	require.Empty(t, utzTemplateStrings(javaContainer.Templates))
}

// The same for the workload scope: the resolved workload identity is what the
// rule scope is matched against.
func TestAnActionSourceScopeIsMatchedAgainstTheResolvedWorkloadIdentity(t *testing.T) {
	thisWorkload := k8sconsts.PodWorkload{Namespace: utzNamespace, Kind: k8sconsts.WorkloadKindDeployment, Name: utzWorkload}
	otherWorkload := k8sconsts.PodWorkload{Namespace: utzNamespace, Kind: k8sconsts.WorkloadKindDeployment, Name: "other"}

	utzCluster(t, utzIC(utzCollectorConfig(utzContainer)),
		utzAction("matching", actionsv1.UrlTemplatizationRule{
			Scopes:    &k8sconsts.SourcesScopes{Sources: []k8sconsts.PodWorkload{thisWorkload}},
			Templates: []string{"/matching/{id}"},
		}),
		utzAction("other-workload", actionsv1.UrlTemplatizationRule{
			Scopes:    &k8sconsts.SourcesScopes{Sources: []k8sconsts.PodWorkload{otherWorkload}},
			Templates: []string{"/other-workload/{id}"},
		}),
		utzAction("other-namespace", actionsv1.UrlTemplatizationRule{
			Scopes:    &k8sconsts.SourcesScopes{Namespaces: []string{"somewhere-else"}},
			Templates: []string{"/other-namespace/{id}"},
		}),
	)

	templatization, _ := utzResolve(t, utzContainer)

	require.Equal(t, []string{"/matching/{id}"}, utzTemplateStrings(templatization.Templates))
}

// --- template ownership and de-duplication ---------------------------------

// The same template can be reached from an Action's documented templates, from
// its plain templates and from the resolved config. The first writer owns the
// metadata the UI renders, so the order of the three passes is a contract.
func TestADocumentedTemplateOwnsATemplateThatIsAlsoListedPlainly(t *testing.T) {
	utzCluster(t, utzIC(utzCollectorConfig(utzContainer, "/users/{id}")),
		utzAction("owner", actionsv1.UrlTemplatizationRule{
			DocumentedTemplates: []actionsv1.UrlTemplatizationDocumentedTemplate{
				{Template: "/users/{id}", Examples: []string{"/users/7"}, Notes: "learned from live traffic"},
			},
			Templates: []string{"/users/{id}"},
		}),
	)

	templatization, _ := utzResolve(t, utzContainer)

	require.Equal(t, []string{"/users/{id}"}, utzTemplateStrings(templatization.Templates))
	only := templatization.Templates[0]
	require.Equal(t, []string{"/users/7"}, only.Examples)
	require.Equal(t, "learned from live traffic", *only.Notes)
	require.Equal(t, "owner", *only.ActionID)
	require.Equal(t, "action display owner", *only.ActionName)
	require.Equal(t, model.ManagedByLiveTrafficLearning, only.ManagedBy)
}

// When two Actions document the same template, the first one in the list owns it.
func TestTheFirstActionToClaimATemplateOwnsIt(t *testing.T) {
	first := utzAction("aaa-first", actionsv1.UrlTemplatizationRule{
		DocumentedTemplates: []actionsv1.UrlTemplatizationDocumentedTemplate{
			{Template: "/users/{id}", Notes: "from the first action"},
		},
	})
	second := utzAction("zzz-second", actionsv1.UrlTemplatizationRule{
		DocumentedTemplates: []actionsv1.UrlTemplatizationDocumentedTemplate{
			{Template: "/users/{id}", Notes: "from the second action"},
		},
	})

	utzCluster(t, utzIC(utzCollectorConfig(utzContainer)), first, second)

	templatization, _ := utzResolve(t, utzContainer)

	require.Len(t, templatization.Templates, 1)
	require.Equal(t, "aaa-first", *templatization.Templates[0].ActionID)
	require.Equal(t, "from the first action", *templatization.Templates[0].Notes)
}

// A template that is in the resolved config but matches no Action rule any more
// is reported with no owner, which is what tells the UI it cannot be edited
// through an Action.
func TestATemplateNoActionClaimsIsReportedWithoutAnOwner(t *testing.T) {
	utzCluster(t, utzIC(utzCollectorConfig(utzContainer, "/orphan/{id}")),
		utzAction("owner", actionsv1.UrlTemplatizationRule{Templates: []string{"/owned/{id}"}}),
	)

	templatization, _ := utzResolve(t, utzContainer)

	require.Equal(t, []string{"/orphan/{id}", "/owned/{id}"}, utzTemplateStrings(templatization.Templates))

	orphan := utzTemplate(t, templatization, "/orphan/{id}")
	require.Nil(t, orphan.ActionID)
	require.Nil(t, orphan.ActionName)
	require.Equal(t, model.ManagedByUnknown, orphan.ManagedBy)
	require.NotNil(t, orphan.Examples)
	require.Empty(t, orphan.Examples)

	owned := utzTemplate(t, templatization, "/owned/{id}")
	require.Equal(t, "owner", *owned.ActionID)
	require.Equal(t, model.ManagedByLiveTrafficLearning, owned.ManagedBy)
}

// An Action with no display name must report a null actionName rather than an
// empty string, so the UI falls back to the action id instead of rendering "".
func TestAnActionWithNoDisplayNameReportsANullActionName(t *testing.T) {
	nameless := utzAction("nameless", actionsv1.UrlTemplatizationRule{
		DocumentedTemplates: []actionsv1.UrlTemplatizationDocumentedTemplate{{Template: "/users/{id}", Notes: "  "}},
	})
	nameless.Spec.ActionName = "   "

	utzCluster(t, utzIC(utzCollectorConfig(utzContainer)), nameless)

	templatization, _ := utzResolve(t, utzContainer)

	only := utzTemplate(t, templatization, "/users/{id}")
	require.Equal(t, "nameless", *only.ActionID)
	require.Nil(t, only.ActionName)
	require.Nil(t, only.Notes)
	// examples: [String!]! is non-null, so a documented template with no
	// examples has to serve an empty list rather than null
	require.NotNil(t, only.Examples)
	require.Empty(t, only.Examples)
}

// templates: [LiveTrafficLearningTemplate!]! is a non-null list, so the
// converter has to turn a nil list into an empty one. No production caller
// passes nil today — ResolveExistingUrlTemplatizationTemplates always allocates
// — so this is the only thing holding that guard in place.
func TestTheTemplatizationModelNeverCarriesANullTemplateList(t *testing.T) {
	out := UrlTemplatizationConfigToModel(&actionsapi.UrlTemplatizationConfig{}, nil)

	require.NotNil(t, out.Templates)
	require.Empty(t, out.Templates)
	require.Nil(t, out.PatternBasedTemplatization)
}

// Surrounding whitespace is trimmed before the template is de-duplicated, so a
// padded Action template and a clean config template are one entry, not two.
func TestTemplatesAreTrimmedBeforeTheyAreDeduplicated(t *testing.T) {
	utzCluster(t, utzIC(utzCollectorConfig(utzContainer, "/users/{id}", "/orders/{id}", "   ")),
		utzAction("owner", actionsv1.UrlTemplatizationRule{
			Templates: []string{"  /users/{id}  "},
			DocumentedTemplates: []actionsv1.UrlTemplatizationDocumentedTemplate{
				{Template: "\t/orders/{id}\n"},
			},
		}),
	)

	templatization, _ := utzResolve(t, utzContainer)

	require.Equal(t, []string{"/orders/{id}", "/users/{id}"}, utzTemplateStrings(templatization.Templates))
	for _, template := range templatization.Templates {
		require.Equal(t, "owner", *template.ActionID, "%q lost its owner to a duplicate", template.Template)
	}
}

// --- the path rules the enterprise caller filters unmatched traffic with -----

// The second return value must be parsed from the RESOLVED templates — the union
// of the config and the matching Actions — and not from the config alone. An
// Action template missing from the rules would make live traffic that the
// Action already templatizes show up as "unmatched" and get recommended again.
func TestThePathRulesCoverTheActionTemplatesAndNotJustTheConfiguredOnes(t *testing.T) {
	utzCluster(t, utzIC(utzCollectorConfig(utzContainer, "/from-the-config/{id}")),
		utzAction("owner", actionsv1.UrlTemplatizationRule{Templates: []string{"/from-the-action/{id}"}}),
	)

	templatization, rules := utzResolve(t, utzContainer)

	require.Equal(t, []string{"/from-the-action/{id}", "/from-the-config/{id}"}, utzTemplateStrings(templatization.Templates))
	require.Len(t, rules, 2)
	require.True(t, utzMatchesAnyRule("/from-the-action/42", rules))
	require.True(t, utzMatchesAnyRule("/from-the-config/42", rules))
	require.False(t, utzMatchesAnyRule("/something-else/42", rules))
}

// ParseTemplatizationPathRules re-implements urltemplate.ParseRulesBySegmentCount
// with a flat return shape, and the package's only matcher takes the bucketed one.
// Nothing links the two parsers, so this pins that they still agree rule for rule
// — a parser fix applied to one of them has to be applied to the other.
func TestTheTwoTemplateParsersStillAgreeRuleForRule(t *testing.T) {
	templates := []string{"/users/{id}", "/health", "/a/*/b", "/users/{id}/orders/{orderId}"}

	flat := ParseTemplatizationPathRules(templates)
	bucketed := urltemplate.ParseRulesBySegmentCount(templates)

	total := 0
	for segmentCount, bucket := range bucketed {
		total += len(bucket)
		for _, rule := range bucket {
			require.Len(t, rule.Segments, segmentCount, "a rule is filed under the wrong segment count")
			require.Contains(t, flat, rule)
		}
	}
	require.Equal(t, len(flat), total)
	require.Len(t, flat, len(templates))

	for _, path := range []string{"/users/42", "/health", "/a/anything/b", "/users/42/orders/7", "/nope"} {
		require.Equal(t, urltemplate.PathMatchesAnyRule(path, bucketed), utzMatchesAnyRule(path, flat), "the two parsers disagree on %q", path)
	}
}

// utzMatchesAnyRule is how a consumer of the flat rule list has to match a path.
func utzMatchesAnyRule(path string, rules []urltemplate.PathRule) bool {
	for _, rule := range rules {
		if rule.IsPathMatching(path) {
			return true
		}
	}
	return false
}

// ParseUserInputRuleString has a single `return rule, nil`, so the error branch
// in ParseTemplatizationPathRules is unreachable: every template becomes a rule,
// including ones a user would call malformed. Pinned so that a parser that
// starts rejecting input does not silently shrink the rule set instead.
func TestEveryTemplateBecomesAPathRuleIncludingMalformedOnes(t *testing.T) {
	templates := []string{"/users/{id}", "/a/*/b", "unanchored", "{", "//", "", "   "}

	rules := ParseTemplatizationPathRules(templates)

	require.Len(t, rules, len(templates))
	require.Equal(t, []urltemplate.RulePathSegment{
		{StaticString: "users"},
		{TemplateName: "id"},
	}, rules[0].Segments)
	require.Equal(t, []urltemplate.RulePathSegment{
		{StaticString: "a"},
		{Wildcard: true},
		{StaticString: "b"},
	}, rules[1].Segments)
}

func TestTheRuleListIsEmptyButAllocatedForAContainerWithNoTemplates(t *testing.T) {
	require.Empty(t, ParseTemplatizationPathRules(nil))
}

// --- the pattern-based defaults --------------------------------------------

// The pattern-based block is optional and its skip policy is optional inside it;
// each absent level must stay absent rather than materialise as a zero value.
func TestThePatternBasedTemplatizationFollowsTheConfigLevelByLevel(t *testing.T) {
	for _, tc := range []struct {
		name            string
		defaults        *actionsapi.DefaultTemplatizationConfig
		wantPatternNil  bool
		wantPolicyNil   bool
		wantDisabled    bool
		wantNonSuccess  bool
		wantStatusCodes []int
	}{
		{name: "no defaults at all", defaults: nil, wantPatternNil: true},
		{
			name:          "defaults without a skip policy",
			defaults:      &actionsapi.DefaultTemplatizationConfig{Disabled: true},
			wantPolicyNil: true,
			wantDisabled:  true,
		},
		{
			name: "defaults with a skip policy",
			defaults: &actionsapi.DefaultTemplatizationConfig{
				SkipPolicy: &actionsapi.DefaultTemplatizationSkipPolicyConfig{
					SkipForNonSuccessCodes: true,
					SkipHttpStatusCodes:    []int{404, 418},
				},
			},
			wantNonSuccess:  true,
			wantStatusCodes: []int{404, 418},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ic := utzIC(utzCollectorConfig(utzContainer, "/users/{id}"))
			ic.Spec.WorkloadCollectorConfig[0].UrlTemplatization.Default = tc.defaults
			utzCluster(t, ic)

			templatization, _ := utzResolve(t, utzContainer)

			if tc.wantPatternNil {
				require.Nil(t, templatization.PatternBasedTemplatization)
				return
			}
			require.NotNil(t, templatization.PatternBasedTemplatization)
			require.Equal(t, tc.wantDisabled, templatization.PatternBasedTemplatization.Disabled)
			if tc.wantPolicyNil {
				require.Nil(t, templatization.PatternBasedTemplatization.SkipPolicy)
				return
			}
			require.NotNil(t, templatization.PatternBasedTemplatization.SkipPolicy)
			require.Equal(t, tc.wantNonSuccess, *templatization.PatternBasedTemplatization.SkipPolicy.SkipForNonSuccessCodes)
			require.Equal(t, tc.wantStatusCodes, templatization.PatternBasedTemplatization.SkipPolicy.SkipHTTPStatusCodes)
		})
	}
}
