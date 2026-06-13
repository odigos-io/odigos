package serviceioconnector

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"

	commonapi "github.com/odigos-io/odigos/common/api"
	odigoscollector "github.com/odigos-io/odigos/common/collector"

	"github.com/odigos-io/odigos/collector/pkg/completetrace"
)

var odigosConfigExtensionID = component.MustNewID("odigos_config_k8s")

type testHost struct {
	extensions map[component.ID]component.Component
}

func (h *testHost) GetExtensions() map[component.ID]component.Component {
	return h.extensions
}

func (h *testHost) GetFactory(component.Kind, component.Type) component.Factory {
	return nil
}

func startConnectorWithMockExtension(t *testing.T, connector component.Component, ext *mockOdigosConfigExtension) {
	t.Helper()
	host := &testHost{
		extensions: map[component.ID]component.Component{
			odigosConfigExtensionID: ext,
		},
	}
	require.NoError(t, connector.Start(t.Context(), host))
	t.Cleanup(func() {
		require.NoError(t, connector.Shutdown(t.Context()))
	})
}

type mockOdigosConfigExtension struct {
	activeSources map[string]struct{}
}

func (m *mockOdigosConfigExtension) GetFromResource(_ pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	return nil, false
}

func (m *mockOdigosConfigExtension) IsActiveSource(res pcommon.Resource) bool {
	serviceName, ok := res.Attributes().Get("service.name")
	if !ok {
		return false
	}
	_, active := m.activeSources[serviceName.Str()]
	return active
}

func (m *mockOdigosConfigExtension) GetWorkloadCacheKey(_ pcommon.Resource) (string, error) {
	return "", nil
}

func (m *mockOdigosConfigExtension) GetWorkloadIdentityFromResource(res pcommon.Resource) (string, pcommon.Map, error) {
	serviceName, ok := res.Attributes().Get("service.name")
	if !ok {
		return "", pcommon.NewMap(), fmt.Errorf("missing service.name")
	}
	attrs := pcommon.NewMap()
	serviceName.CopyTo(attrs.PutEmpty("service.name"))
	return serviceName.Str(), attrs, nil
}

func (m *mockOdigosConfigExtension) RegisterWorkloadConfigCacheCallback(_ odigoscollector.WorkloadConfigCacheCallback) {
}

func (m *mockOdigosConfigExtension) UnregisterWorkloadConfigCacheCallback(_ odigoscollector.WorkloadConfigCacheCallback) {
}

func (m *mockOdigosConfigExtension) WaitForCacheSync(_ context.Context) bool {
	return true
}

func (m *mockOdigosConfigExtension) GetDataStreamsForWorkload(_ pcommon.Resource) ([]string, bool) {
	return nil, false
}

func (m *mockOdigosConfigExtension) Start(context.Context, component.Host) error {
	return nil
}

func (m *mockOdigosConfigExtension) Shutdown(context.Context) error {
	return nil
}

func TestAggregateConnectionsFromTree_FiltersInactiveSources(t *testing.T) {
	td := buildServiceIOTestTrace(t)

	tree, err := completetrace.BuildTraceTree(td, nil)
	require.NoError(t, err)

	connector := &serviceioConnector{
		keyToMetric:          make(map[uint64]metricSeries),
		inputSpanAttributes:  []string{"http.route"},
		outputSpanAttributes: []string{"rpc.service"},
		odigosConfig: &mockOdigosConfigExtension{
			activeSources: map[string]struct{}{"svc-1": {}},
		},
	}

	require.True(t, connector.aggregateConnectionsFromTree(tree))
	require.Len(t, connector.keyToMetric, 2)
}

func TestAggregateConnectionsFromTree_SkipsInactiveSources(t *testing.T) {
	td := buildServiceIOTestTrace(t)

	tree, err := completetrace.BuildTraceTree(td, nil)
	require.NoError(t, err)

	connector := &serviceioConnector{
		keyToMetric:          make(map[uint64]metricSeries),
		inputSpanAttributes:  []string{"http.route"},
		outputSpanAttributes: []string{"rpc.service"},
		odigosConfig: &mockOdigosConfigExtension{
			activeSources: map[string]struct{}{},
		},
	}

	require.False(t, connector.aggregateConnectionsFromTree(tree))
	require.Empty(t, connector.keyToMetric)
}
