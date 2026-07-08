package collectorconfig

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
)

// boolOrDefault dereferences an optional bool toggle, falling back to def when unset.
func boolOrDefault(p *bool, def bool) bool {
	if p != nil {
		return *p
	}
	return def
}

// memoryReceiverConfig renders the "memory" block of the profiling (ebpf-profiler)
// receiver from the control-plane Profiling.Memory knob. It resolves every default
// explicitly so behavior is deterministic regardless of agent-side defaults. Keys
// mirror the agent's collector/config.MemoryConfig mapstructure tags.
func memoryReceiverConfig(m *common.ProfilingMemoryConfiguration) config.GenericMap {
	sample := m.SampleSizeBytes
	if sample == 0 {
		sample = 262144 // 256KiB
	}
	intervalSec := m.ReportIntervalSeconds
	if intervalSec == 0 {
		intervalSec = 15
	}
	nativeMode := m.NativeMode
	if nativeMode == "" {
		nativeMode = "off"
	}
	javaMode := m.JavaMode
	if javaMode == "" {
		javaMode = "jfr" // Tier-1 default: the JVM's own Flight Recorder
	}
	goOn, javaOn, nativeOn, dotnetOn, nodeOn := true, true, false, false, false
	if l := m.Languages; l != nil {
		goOn = boolOrDefault(l.Go, true)
		javaOn = boolOrDefault(l.Java, true)
		nativeOn = boolOrDefault(l.Native, false)
		dotnetOn = boolOrDefault(l.Dotnet, false)
		nodeOn = boolOrDefault(l.Node, false)
	}
	return config.GenericMap{
		"enabled":           true,
		"sample_size_bytes": sample,
		"report_interval":   fmt.Sprintf("%ds", intervalSec),
		"inuse_tracking":    boolOrDefault(m.InuseTracking, true),
		// inject is the no-restart enablement mechanism (ptrace; Go heap sampling
		// today). Off by default and independent of native.mode — the agent's
		// memory subsystem reads it as the top-level "inject" mapstructure key.
		"inject": boolOrDefault(m.Inject, false),
		"languages": config.GenericMap{
			"go": goOn, "java": javaOn, "native": nativeOn, "dotnet": dotnetOn, "node": nodeOn,
		},
		"java":   config.GenericMap{"mode": javaMode},
		"native": config.GenericMap{"mode": nativeMode},
		// metrics turns on the memory subsystem's internal perf counters (spike-free,
		// batched). Off by default; the scrape source for the memprof dashboard.
		"metrics": boolOrDefault(m.Metrics, false),
	}
}

// memoryNativeEnabled reports whether native (C/C++/Rust) memory profiling is on,
// which requires central symbolization of the resulting address-based frames.
func memoryNativeEnabled(profiling *common.ProfilingConfiguration) bool {
	if !common.MemoryProfilingActive(profiling) || profiling.Memory.Languages == nil {
		return false
	}
	return boolOrDefault(profiling.Memory.Languages.Native, false)
}

// ProfilingPipelineConfig builds the node collector profiles domain when profiling is enabled.
func ProfilingPipelineConfig(odigosNamespace string, profiling *common.ProfilingConfiguration) config.Config {
	if !common.ProfilingPipelineActive(profiling) {
		return config.Config{}
	}

	endpoint := k8sconsts.OtlpGrpcDNSEndpoint(k8sconsts.OdigosClusterCollectorServiceName, odigosNamespace, odigosconsts.OTLPPort)
	exp := commonconf.MergeProfilingOtlpExporter(config.GenericMap{
		"endpoint":    endpoint,
		"tls":         config.GenericMap{"insecure": true},
		"compression": "none",
	}, profiling.Exporter)

	processors := config.GenericMap{
		commonconf.ProfilingNodeFilterProcessor:         commonconf.ProfilingFilterProcessorConfig(),
		commonconf.ProfilingNodeK8sAttributesProcessor:  commonconf.K8sAttributesProfilesProcessorConfig(),
		commonconf.ProfilingNodeOdigosProfilesProcessor: commonconf.OdigosProfilesProcessorConfig(),
		commonconf.ProfilingNodeServiceNameProcessor:    commonconf.ProfilingServiceNameTransformConfig(),
	}
	pipelineProcessors := []string{
		commonconf.ProfilingNodeFilterProcessor,
		commonconf.ProfilingNodeK8sAttributesProcessor,
		commonconf.ProfilingNodeOdigosProfilesProcessor,
	}
	// Native symbolization is opt-in (profiling.symbolization.native), but it is
	// REQUIRED whenever native memory profiling is on (those frames arrive as
	// module+offset and must be named centrally). When on, the symbolize processor
	// runs after the keep-filter (only retained profiles are symbolized) and before
	// service-name enrichment.
	if profiling.NativeSymbolizationEnabled() || memoryNativeEnabled(profiling) {
		processors[commonconf.ProfilingNodeSymbolizeProcessor] = commonconf.OdigosSymbolizeProcessorConfig()
		pipelineProcessors = append(pipelineProcessors, commonconf.ProfilingNodeSymbolizeProcessor)
	}
	pipelineProcessors = append(pipelineProcessors, commonconf.ProfilingNodeServiceNameProcessor)

	// The profiling receiver is the node-wide ebpf-profiler. CPU profiling is its
	// default; memory profiling is layered on via the "memory" config block when
	// Profiling.Memory is enabled (same receiver, same pipeline as CPU).
	receiverConfig := config.GenericMap{}
	if common.MemoryProfilingActive(profiling) {
		receiverConfig["memory"] = memoryReceiverConfig(profiling.Memory)
	}

	return config.Config{
		Receivers: config.GenericMap{
			commonconf.ProfilingReceiver: receiverConfig,
		},
		Processors: processors,
		Exporters: config.GenericMap{
			commonconf.ProfilingNodeToGatewayExporter: exp,
		},
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				"profiles": {
					Receivers:  []string{commonconf.ProfilingReceiver},
					Processors: pipelineProcessors,
					Exporters:  []string{commonconf.ProfilingNodeToGatewayExporter},
				},
			},
		},
	}
}
