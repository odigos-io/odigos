package actions

// In-process end-to-end: drive real Actions through convertActionToProcessor, the OrderHint
// filter/sort, bucketing and both the gateway and node render, asserting the rendered configs.
// Ties together what the unit tests cover in isolation, including OrderHint ordering and the fact
// that the tier-1 actions are cluster-gateway scoped.

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/autoscaler/controllers/nodecollector/collectorconfig"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/pipelinegen"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pyroscopeDest is a Pyroscope-typed destination so the real Pyroscope configer registers a
// "profiles/pyroscope-<id>" pipeline via addProfilesPipeline.
type pyroscopeDest struct{ id string }

func (d pyroscopeDest) GetID() string                   { return d.id }
func (d pyroscopeDest) GetType() common.DestinationType { return common.PyroscopeDestinationType }
func (d pyroscopeDest) GetConfig() map[string]string {
	return map[string]string{"PYROSCOPE_URL": "pyroscope.example.com:4040"}
}
func (d pyroscopeDest) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.ProfilesObservabilitySignal}
}

func mkAction(name string, spec odigosv1.ActionSpec) *odigosv1.Action {
	return &odigosv1.Action{
		TypeMeta:   metav1.TypeMeta{APIVersion: "odigos.io/v1alpha1", Kind: "Action"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "odigos-system", UID: types.UID(name + "-uid")},
		Spec:       spec,
	}
}

func TestE2E_ProfilesActions_ActionToRenderedConfig(t *testing.T) {
	ctx := context.Background()
	str := func(s string) *string { return &s }
	// Each action selects PROFILES *and* TRACES, so a single generated processor config carries both
	// profile_statements and trace_statements — the shared-across-pipelines case.
	sigs := []common.ObservabilitySignal{common.ProfilesObservabilitySignal, common.TracesObservabilitySignal}

	actions := []*odigosv1.Action{
		mkAction("add-cluster", odigosv1.ActionSpec{ActionName: "add cluster info", Signals: sigs,
			AddClusterInfo: &actionsv1.AddClusterInfoConfig{ClusterAttributes: []actionsv1.OtelAttributeWithValue{
				{AttributeName: "k8s.cluster.name", AttributeStringValue: str("prod-eu")}}}}),
		mkAction("rename-attr", odigosv1.ActionSpec{ActionName: "rename attr", Signals: sigs,
			RenameAttribute: &actionsv1.RenameAttributeConfig{Renames: map[string]string{"old.name": "new.name"}}}),
		mkAction("delete-attr", odigosv1.ActionSpec{ActionName: "delete attr", Signals: sigs,
			DeleteAttribute: &actionsv1.DeleteAttributeConfig{AttributeNamesToDelete: []string{"secret.token"}}}),
	}

	// 1) Real reconciler conversion: Action -> Processor CRD.
	procList := &odigosv1.ProcessorList{}
	for _, a := range actions {
		p, err := convertActionToProcessor(ctx, nil, a)
		require.NoError(t, err, "convertActionToProcessor(%s)", a.Name)
		procList.Items = append(procList.Items, *p)
	}

	// 2) Production filter+sort by OrderHint, per collector role. All three tier-1 actions declare
	// only the ClusterGateway role, so the gateway gets all three (ordered) and the node gets none.
	gwProcs := commonconf.FilterAndSortProcessorsByOrderHint(procList, odigosv1.CollectorsGroupRoleClusterGateway)
	nodeProcs := commonconf.FilterAndSortProcessorsByOrderHint(procList, odigosv1.CollectorsGroupRoleNodeCollector)
	require.Len(t, gwProcs, 3)
	require.Len(t, nodeProcs, 0)

	// 3) Bucketing: all three are admitted to profiles, ordered by OrderHint (delete -100, rename -50,
	// addclusterinfo 1), and also to traces — with no cross-contamination.
	results := config.CrdProcessorToConfig(commonconf.ToProcessorConfigurerArray(gwProcs))
	require.Empty(t, results.Errs)
	wantOrder := []string{"transform/delete-attr", "transform/rename-attr", "resource/add-cluster"}
	assert.Equal(t, wantOrder, results.ProfilesProcessors)
	assert.Len(t, results.TracesProcessors, 3)

	// --- GATEWAY render ---
	gwOpts := pipelinegen.GatewayConfigOptions{OdigosNamespace: "odigos-system"}
	gw, err, statuses, signals := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{pyroscopeDest{id: "p1"}},
		commonconf.ToProcessorConfigurerArray(gwProcs),
		func(c *config.Config, _ []string, _ []string) error { return nil },
		nil, &gwOpts)
	require.NoError(t, err)
	require.NoError(t, statuses.Destination["p1"])
	require.Contains(t, signals, common.ProfilesObservabilitySignal)

	// Profiles flow through the router: the "profiles/in" root pipeline runs the processors once
	// (OrderHint order, after resource/odigos-version) and exports to odigosrouterconnector/profiles.
	root, ok := gw.Service.Pipelines["profiles/in"]
	require.True(t, ok)
	assert.Equal(t, append([]string{"resource/odigos-version"}, wantOrder...), root.Processors, "profiles root processor order")
	assert.Equal(t, []string{"odigosrouterconnector/profiles"}, root.Exporters)
	// The destination pipeline receives from its forward connector, no batch.
	dest, ok := gw.Service.Pipelines["profiles/pyroscope-p1"]
	require.True(t, ok)
	assert.Contains(t, dest.Receivers, "forward/profiles/pyroscope-p1")
	assert.NotContains(t, dest.Processors, "batch")
	// The transform processors carry profile_statements with exactly the collector-safe contexts.
	assertProfileStatementContexts(t, gw.Processors, "transform/rename-attr")
	assertProfileStatementContexts(t, gw.Processors, "transform/delete-attr")

	// --- NODE render --- gateway-scoped actions => node profiles pipeline is unchanged (built-in chain).
	on := true
	nodeResults := config.CrdProcessorToConfig(commonconf.ToProcessorConfigurerArray(nodeProcs))
	require.Empty(t, nodeResults.ProfilesProcessors)
	nodeCfg, err := config.MergeConfigs(map[string]config.Config{
		"processors": nodeResults.ProcessorsConfig,
		"profiling":  collectorconfig.ProfilingPipelineConfig("odigos-system", &common.ProfilingConfiguration{Enabled: &on}, nodeResults.ProfilesProcessors),
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		commonconf.ProfilingNodeFilterProcessor,
		commonconf.ProfilingNodeK8sAttributesProcessor,
		commonconf.ProfilingNodeOdigosProfilesProcessor,
		commonconf.ProfilingNodeSymbolizeProcessor,
		commonconf.ProfilingNodeServiceNameProcessor,
	}, nodeCfg.Service.Pipelines["profiles"].Processors)
}

// assertProfileStatementContexts asserts the processor carries profile_statements over exactly the
// resource/scope/profile contexts — the only ones the transformprocessor accepts for profiles.
// The config has been through a JSON round-trip (Processor CRD RawExtension -> GetConfig), so the
// statements are generic maps rather than typed OttlStatementConfig structs.
func assertProfileStatementContexts(t *testing.T, procs config.GenericMap, key string) {
	t.Helper()
	m, ok := procs[key].(config.GenericMap)
	require.True(t, ok, "processor %q config missing/not a map", key)
	ps, ok := m["profile_statements"].([]interface{})
	require.True(t, ok, "processor %q missing profile_statements (got %T)", key, m["profile_statements"])
	got := make([]string, len(ps))
	for i, s := range ps {
		sm, ok := s.(map[string]interface{})
		require.True(t, ok, "profile_statements[%d] not a map: %T", i, s)
		got[i], _ = sm["context"].(string)
	}
	assert.Equal(t, []string{"resource", "scope", "profile"}, got, "processor %q profile_statements contexts", key)
}
