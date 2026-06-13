package odigosprofilesprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"

	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/collector"
)

type stubOdigosConfigExtension struct {
	managed map[string]struct{}
}

func (s *stubOdigosConfigExtension) GetFromResource(pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	return nil, false
}

func (s *stubOdigosConfigExtension) IsActiveSource(res pcommon.Resource) bool {
	attrs := res.Attributes()
	ns, _ := attrs.Get(string(semconv.K8SNamespaceNameKey))
	dep, ok := attrs.Get(string(semconv.K8SDeploymentNameKey))
	if !ok || ns.Str() == "" || dep.Str() == "" {
		return false
	}
	wk := ns.Str() + "/Deployment/" + dep.Str()
	_, found := s.managed[wk]
	return found
}

func (s *stubOdigosConfigExtension) GetWorkloadCacheKey(pcommon.Resource) (string, error) {
	return "", nil
}

func (s *stubOdigosConfigExtension) GetWorkloadIdentityFromResource(pcommon.Resource) (string, pcommon.Map, error) {
	return "", pcommon.NewMap(), nil
}

func (s *stubOdigosConfigExtension) RegisterWorkloadConfigCacheCallback(collector.WorkloadConfigCacheCallback) {
}

func (s *stubOdigosConfigExtension) UnregisterWorkloadConfigCacheCallback(collector.WorkloadConfigCacheCallback) {
}

func (s *stubOdigosConfigExtension) WaitForCacheSync(context.Context) bool { return true }

func (s *stubOdigosConfigExtension) GetDataStreamsForWorkload(pcommon.Resource) ([]string, bool) {
	return nil, false
}

func testProfilesProcessor(provider collector.OdigosConfigExtension) *odigosProfilesProcessor {
	proc := newOdigosProfilesProcessor(zap.NewNop(), componenttest.NewNopTelemetrySettings(), &Config{})
	proc.provider = provider
	return proc
}

func TestProcessProfiles_keepsManagedWorkload(t *testing.T) {
	proc := testProfilesProcessor(&stubOdigosConfigExtension{
		managed: map[string]struct{}{"ns/Deployment/app": {}},
	})

	pd := pprofile.NewProfiles()
	rp := pd.ResourceProfiles().AppendEmpty()
	attrs := rp.Resource().Attributes()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "ns")
	attrs.PutStr(string(semconv.K8SDeploymentNameKey), "app")

	out, err := proc.processProfiles(context.Background(), pd)
	require.NoError(t, err)
	require.Equal(t, 1, out.ResourceProfiles().Len())
}

func TestProcessProfiles_dropsUnmanagedWorkload(t *testing.T) {
	proc := testProfilesProcessor(&stubOdigosConfigExtension{
		managed: map[string]struct{}{},
	})

	pd := pprofile.NewProfiles()
	rp := pd.ResourceProfiles().AppendEmpty()
	attrs := rp.Resource().Attributes()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "ns")
	attrs.PutStr(string(semconv.K8SDeploymentNameKey), "other")

	out, err := proc.processProfiles(context.Background(), pd)
	require.NoError(t, err)
	require.Equal(t, 0, out.ResourceProfiles().Len())
}
