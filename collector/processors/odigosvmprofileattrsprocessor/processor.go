package odigosvmprofileattrsprocessor

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/collector/processors/odigosvmprofileattrsprocessor/internal/metadata"
)

// Resource attribute keys, sourced from semconv so they stay aligned with the eBPF profiler
// receiver, which emits these same keys (semconv.ProcessPIDKey / semconv.ServiceNameKey).
const (
	attrProcessPID  = string(semconv.ProcessPIDKey)
	attrServiceName = string(semconv.ServiceNameKey)
)

// vmProfileAttrsProcessor enriches eBPF profile batches with workload identity from the VM agent
// and drops any resource it cannot identify.
type vmProfileAttrsProcessor struct {
	logger           *zap.Logger
	cfg              *Config
	attrCache        *profileAttrCache
	telemetryBuilder *metadata.TelemetryBuilder
}

// capabilities reports that this processor mutates pprofile data.
func (p *vmProfileAttrsProcessor) capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// start binds this processor to the process-global PID→attrs cache (and starts the
// single unixfd client on first call). The cache + client are intentionally NOT
// owned by the processor: a config reload destroys and rebuilds the processor, but
// the singleton survives, so the cache stays warm and no profiles are dropped during
// the rebuild. See shared_cache.go.
func (p *vmProfileAttrsProcessor) start(_ context.Context, _ component.Host) error {
	p.attrCache = sharedProfileAttrCache(p.cfg.SocketPath, p.logger)
	return nil
}

// shutdown releases the telemetry builder. It deliberately does NOT stop the unixfd
// client or clear the cache — those are process-global (shared_cache.go) and must
// survive config reloads (which call shutdown on the retiring processor).
func (p *vmProfileAttrsProcessor) shutdown(context.Context) error {
	if p.telemetryBuilder != nil {
		p.telemetryBuilder.Shutdown()
	}
	return nil
}

// processProfiles keeps only resource profiles whose process.pid is registered in the VM agent
// cache, enriches them from the streamed attribute map, and drops everything else. Every drop
// increments the dropped_resource_profiles counter with a reason label.
func (p *vmProfileAttrsProcessor) processProfiles(ctx context.Context, profiles pprofile.Profiles) (pprofile.Profiles, error) {
	rps := profiles.ResourceProfiles()
	if rps.Len() == 0 {
		return profiles, nil
	}

	out := pprofile.NewProfiles()
	profiles.Dictionary().CopyTo(out.Dictionary())
	outRps := out.ResourceProfiles()

	// Per-batch cache: serviceName -> attribute index in out's dictionary. Scoped to this call
	// only — the index is meaningless against any other batch's dictionary, so it must not escape.
	svcAttrIdxCache := make(map[string]int32)

	for i := 0; i < rps.Len(); i++ {
		rp := rps.At(i)
		attrs := rp.Resource().Attributes()

		pidVal, ok := attrs.Get(attrProcessPID)
		if !ok {
			p.recordDrop(ctx)
			p.logger.Debug("dropping profile resource without process.pid")
			continue
		}
		pid := uint32(pidVal.Int())

		packed, registered := p.attrCache.get(pid)
		if !registered {
			p.recordDrop(ctx)
			p.logger.Debug("dropping profile resource for unregistered pid",
				zap.Uint32("pid", pid))
			continue
		}

		if err := applyPackedResourceAttributes(attrs, packed); err != nil {
			p.recordDrop(ctx)
			p.logger.Debug("dropping profile resource after failed attribute enrichment",
				zap.Uint32("pid", pid),
				zap.Error(err))
			continue
		}

		dest := outRps.AppendEmpty()
		rp.CopyTo(dest)
		if svc, ok := dest.Resource().Attributes().Get(attrServiceName); ok {
			propagateServiceNameToSamples(out.Dictionary(), dest, svc.AsString(), svcAttrIdxCache)
		}
	}

	return out, nil
}

// recordDrop increments the dropped_resource_profiles counter.
// Safe to call when the telemetry builder failed to initialize.
func (p *vmProfileAttrsProcessor) recordDrop(ctx context.Context) {
	if p.telemetryBuilder == nil || p.telemetryBuilder.OdigosVMProfileAttrsProcessorDroppedResourceProfiles == nil {
		return
	}
	p.telemetryBuilder.OdigosVMProfileAttrsProcessorDroppedResourceProfiles.Add(ctx, 1)
}

// applyPackedResourceAttributes parses "key:value,key:value" into resource attributes,
// unconditionally overwriting existing values. The VM agent is the authoritative source
// of workload identity for the eBPF profiler — the receiver-emitted service.name
// (typically "unknown_service:<exe>") is always replaced.
func applyPackedResourceAttributes(resourceAttrs pcommon.Map, attributesStr string) error {
	if strings.TrimSpace(attributesStr) == "" {
		return fmt.Errorf("empty attributes string")
	}

	parsed := false
	for _, part := range strings.Split(attributesStr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		if key == "" || val == "" {
			continue
		}
		resourceAttrs.PutStr(key, val)
		parsed = true
	}
	if !parsed {
		return fmt.Errorf("no valid attributes parsed from: %s", attributesStr)
	}
	return nil
}
