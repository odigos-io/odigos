package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func newAgentsClient(objects ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = odigosv1.AddToScheme(scheme)
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

// instrumentationConfig builds an IC whose containers carry a detected language
// and the instrumentor's agent decision, the two halves the coverage count joins.
func instrumentationConfig(name string, containers ...struct {
	name         string
	language     common.ProgrammingLanguage
	distro       string
	agentEnabled bool
}) *odigosv1.InstrumentationConfig {
	ic := &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
	}
	for _, c := range containers {
		ic.Spec.Containers = append(ic.Spec.Containers, odigosv1.ContainerAgentConfig{
			ContainerName:  c.name,
			AgentEnabled:   c.agentEnabled,
			OtelDistroName: c.distro,
		})
		ic.Status.RuntimeDetailsByContainer = append(ic.Status.RuntimeDetailsByContainer, odigosv1.RuntimeDetailsByContainer{
			ContainerName: c.name,
			Language:      c.language,
		})
	}
	return ic
}

type testContainer = struct {
	name         string
	language     common.ProgrammingLanguage
	distro       string
	agentEnabled bool
}

func communityProvider(t *testing.T) *distros.Provider {
	t.Helper()
	getter, err := distros.NewCommunityGetter()
	require.NoError(t, err)
	provider, err := distros.NewProvider(distros.NewCommunityDefaulter(), getter)
	require.NoError(t, err)
	return provider
}

func agentByLanguage(agents []*model.InstrumentationAgent, language string) *model.InstrumentationAgent {
	for _, a := range agents {
		if a.Language == language {
			return a
		}
	}
	return nil
}

func TestGetInstrumentationAgents_communityDistroMetadata(t *testing.T) {
	agents, err := GetInstrumentationAgents(context.Background(), newAgentsClient(), communityProvider(t))
	require.NoError(t, err)

	// One row per language in the community defaulter.
	require.Len(t, agents, 7)

	// Deterministic ordering, so the response doesn't reshuffle between calls.
	for i := 1; i < len(agents); i++ {
		assert.Less(t, agents[i-1].Language, agents[i].Language)
	}

	goAgent := agentByLanguage(agents, string(common.GoProgrammingLanguage))
	require.NotNil(t, goAgent)
	assert.Equal(t, "golang-community", goAgent.DistroName)
	assert.Equal(t, "go-runtime", goAgent.RuntimeEnvironment)
	assert.Equal(t, ">= 1.19", goAgent.SupportedRuntimeVersions)

	nodeAgent := agentByLanguage(agents, string(common.JavascriptProgrammingLanguage))
	require.NotNil(t, nodeAgent)
	assert.Equal(t, "nodejs-community", nodeAgent.DistroName)

	// Bounded ranges must survive verbatim — the UI needs the upper bound.
	phpAgent := agentByLanguage(agents, string(common.PhpProgrammingLanguage))
	require.NotNil(t, phpAgent)
	assert.Equal(t, ">=8.1,<8.5", phpAgent.SupportedRuntimeVersions)
}

func TestGetInstrumentationAgents_sourceCounts(t *testing.T) {
	objects := []client.Object{
		instrumentationConfig("deployment-coupon",
			testContainer{name: "coupon", language: common.JavascriptProgrammingLanguage, distro: "nodejs-community", agentEnabled: true},
		),
		// A container on a distro the page does not list (a version fallback)
		// still counts against the language's default row.
		instrumentationConfig("deployment-legacy",
			testContainer{name: "legacy", language: common.JavascriptProgrammingLanguage, distro: "nodejs-community-14", agentEnabled: true},
		),
		instrumentationConfig("deployment-currency",
			testContainer{name: "currency", language: common.PhpProgrammingLanguage, distro: "php-community", agentEnabled: true},
			// No agent available: counted against php only if detected as php;
			// nginx has no default distro so it lands on no row at all.
			testContainer{name: "nginx", language: "nginx", agentEnabled: false},
		),
		instrumentationConfig("deployment-broken",
			testContainer{name: "broken", language: common.PythonProgrammingLanguage, agentEnabled: false},
		),
	}

	agents, err := GetInstrumentationAgents(context.Background(), newAgentsClient(objects...), communityProvider(t))
	require.NoError(t, err)

	nodeAgent := agentByLanguage(agents, string(common.JavascriptProgrammingLanguage))
	require.NotNil(t, nodeAgent)
	assert.Equal(t, 2, nodeAgent.Sources)

	phpAgent := agentByLanguage(agents, string(common.PhpProgrammingLanguage))
	require.NotNil(t, phpAgent)
	assert.Equal(t, 1, phpAgent.Sources)

	// Detected but not instrumented — the row the user should look at first.
	pythonAgent := agentByLanguage(agents, string(common.PythonProgrammingLanguage))
	require.NotNil(t, pythonAgent)
	// An uninstrumented container still makes its source count.
	assert.Equal(t, 1, pythonAgent.Sources)

	// A language with nothing running stays at zero rather than disappearing.
	rubyAgent := agentByLanguage(agents, string(common.RubyProgrammingLanguage))
	require.NotNil(t, rubyAgent)
	assert.Equal(t, 0, rubyAgent.Sources)
}

func TestGetInstrumentationAgents_sourcesCountedOncePerLanguage(t *testing.T) {
	objects := []client.Object{
		// Two javascript containers in one source, plus a python sidecar: the
		// javascript row must see 2 containers but only 1 source, and the
		// source is counted again on the python row.
		instrumentationConfig("deployment-multi",
			testContainer{name: "web", language: common.JavascriptProgrammingLanguage, distro: "nodejs-community", agentEnabled: true},
			testContainer{name: "worker", language: common.JavascriptProgrammingLanguage, distro: "nodejs-community", agentEnabled: false},
			testContainer{name: "sidecar", language: common.PythonProgrammingLanguage, distro: "python-community", agentEnabled: true},
		),
	}

	agents, err := GetInstrumentationAgents(context.Background(), newAgentsClient(objects...), communityProvider(t))
	require.NoError(t, err)

	nodeAgent := agentByLanguage(agents, string(common.JavascriptProgrammingLanguage))
	require.NotNil(t, nodeAgent)
	// Two containers of the same language in one source is still one source.
	assert.Equal(t, 1, nodeAgent.Sources)

	pythonAgent := agentByLanguage(agents, string(common.PythonProgrammingLanguage))
	require.NotNil(t, pythonAgent)
	assert.Equal(t, 1, pythonAgent.Sources)
}

func TestGetInstrumentationAgents_noProvider(t *testing.T) {
	_, err := GetInstrumentationAgents(context.Background(), newAgentsClient(), nil)
	assert.Error(t, err)
}

// A defaulter shaped like the enterprise one: its default for a language differs
// from the community default, which is what gives that language two rows.
type testDefaulter struct {
	defaults map[common.ProgrammingLanguage]string
}

func (d testDefaulter) GetDefaultDistroNames() map[common.ProgrammingLanguage]string {
	return d.defaults
}

func (d testDefaulter) GetDefaultVmDistroNames() map[common.ProgrammingLanguage]string {
	return d.defaults
}

func TestGetInstrumentationAgents_listsCommunityAlternative(t *testing.T) {
	getter, err := distros.NewCommunityGetter()
	require.NoError(t, err)
	provider, err := distros.NewProvider(testDefaulter{defaults: map[common.ProgrammingLanguage]string{
		common.JavascriptProgrammingLanguage: "nodejs-community-14",
		common.GoProgrammingLanguage:         "golang-community",
	}}, getter)
	require.NoError(t, err)

	objects := []client.Object{
		instrumentationConfig("deployment-on-default",
			testContainer{name: "a", language: common.JavascriptProgrammingLanguage, distro: "nodejs-community-14", agentEnabled: true},
		),
		instrumentationConfig("deployment-on-community",
			testContainer{name: "b", language: common.JavascriptProgrammingLanguage, distro: "nodejs-community", agentEnabled: true},
		),
		instrumentationConfig("deployment-not-enabled",
			testContainer{name: "c", language: common.JavascriptProgrammingLanguage, agentEnabled: false},
		),
	}

	agents, err := GetInstrumentationAgents(context.Background(), newAgentsClient(objects...), provider)
	require.NoError(t, err)

	byName := make(map[string]*model.InstrumentationAgent, len(agents))
	for _, a := range agents {
		byName[a.DistroName] = a
	}

	// Javascript carries the tier default and the community alternative; Go's two
	// defaulters agree, so it stays a single row.
	require.Len(t, agents, 3)
	require.NotNil(t, byName["nodejs-community-14"])
	require.NotNil(t, byName["nodejs-community"])
	require.NotNil(t, byName["golang-community"])

	// Only the tier's default is flagged, and every default sorts above every
	// alternative, even one whose language sorts earlier.
	assert.True(t, byName["nodejs-community-14"].IsDefault)
	assert.False(t, byName["nodejs-community"].IsDefault)
	assert.True(t, byName["golang-community"].IsDefault)
	assert.Equal(t, []string{"golang-community", "nodejs-community-14", "nodejs-community"},
		[]string{agents[0].DistroName, agents[1].DistroName, agents[2].DistroName})

	// Counts follow the distro each container actually runs.
	assert.Equal(t, 1, byName["nodejs-community"].Sources)

	// The container with no agent has no distro of its own, so it lands on the
	// language's default row rather than disappearing: one source of its own
	// plus the one running the default.
	assert.Equal(t, 2, byName["nodejs-community-14"].Sources)
}
