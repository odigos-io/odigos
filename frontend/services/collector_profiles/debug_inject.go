package collectorprofiles

import (
	_ "embed"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/services/common"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

//go:embed testdata/chunk-with-symbols.json
var embeddedChunkWithSymbols []byte

// InjectDebugSample writes a synthetic OTLP profile chunk (from testdata) into the store for the given
// workload so the Profiler UI / sourceProfiling can render a non-empty flame graph without relying on
// live ebpf → collector → gateway traffic. Only for local/dev when ODIGOS_PROFILE_DEBUG_INJECT is enabled.
func (s *ProfileStore) InjectDebugSample(namespace, kindStr, name string) error {
	id, err := SourceIDFromStrings(namespace, kindStr, name)
	if err != nil {
		return err
	}
	key := SourceKeyFromSourceID(id)

	var unmarshaler pprofile.JSONUnmarshaler
	pd, err := unmarshaler.UnmarshalProfiles(embeddedChunkWithSymbols)
	if err != nil {
		return fmt.Errorf("unmarshal embedded chunk: %w", err)
	}
	if pd.ResourceProfiles().Len() < 1 {
		return fmt.Errorf("embedded chunk has no resource profiles")
	}

	attrs := pd.ResourceProfiles().At(0).Resource().Attributes()
	applyWorkloadAttrsForSource(attrs, id)

	var marshaler pprofile.JSONMarshaler
	out, err := marshaler.MarshalProfiles(pd)
	if err != nil {
		return fmt.Errorf("marshal profiles: %w", err)
	}

	s.StartViewing(key)
	s.AddProfileData(key, out)
	return nil
}

func applyWorkloadAttrsForSource(attrs pcommon.Map, id common.SourceID) {
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), id.Namespace)
	switch id.Kind {
	case k8sconsts.WorkloadKindDeployment:
		attrs.PutStr(string(semconv.K8SDeploymentNameKey), id.Name)
	case k8sconsts.WorkloadKindStatefulSet:
		attrs.PutStr(string(semconv.K8SStatefulSetNameKey), id.Name)
	case k8sconsts.WorkloadKindDaemonSet:
		attrs.PutStr(string(semconv.K8SDaemonSetNameKey), id.Name)
	case k8sconsts.WorkloadKindCronJob:
		attrs.PutStr(string(semconv.K8SCronJobNameKey), id.Name)
	case k8sconsts.WorkloadKindJob:
		attrs.PutStr(string(semconv.K8SJobNameKey), id.Name)
	case k8sconsts.WorkloadKindArgoRollout:
		attrs.PutStr(k8sconsts.K8SArgoRolloutNameAttribute, id.Name)
	default:
		attrs.PutStr(string(semconv.K8SDeploymentNameKey), id.Name)
	}
}
