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

		if pid, ok := resourcePID(attrs); ok {
			packed, registered := p.attrCache.get(pid)
			if !registered {
				p.recordDrop(ctx)
				p.logger.Debug("dropping profile resource for unregistered pid",
					zap.Uint32("pid", pid))
				continue
			}

			p.appendEnrichedResource(ctx, outRps, out.Dictionary(), rp, pid, packed, false, svcAttrIdxCache)
			continue
		}

		// The eBPF profiler sets process.pid on the RESOURCE only for host (non-container)
		// processes. For containerized workloads it groups ResourceProfiles by container.id and
		// carries process.pid on the SAMPLES instead. In that layout a resource may contain
		// samples from several PIDs, so each registered PID must be emitted as a separate
		// resource with only its own samples.
		p.appendRegisteredSamplePIDResources(ctx, profiles.Dictionary(), outRps, out.Dictionary(), rp, svcAttrIdxCache)
	}

	return out, nil
}

func resourcePID(attrs pcommon.Map) (uint32, bool) {
	if v, ok := attrs.Get(attrProcessPID); ok {
		return uint32(v.Int()), true
	}
	return 0, false
}

func (p *vmProfileAttrsProcessor) appendRegisteredSamplePIDResources(ctx context.Context, inDict pprofile.ProfilesDictionary, outRps pprofile.ResourceProfilesSlice, outDict pprofile.ProfilesDictionary, rp pprofile.ResourceProfiles, svcAttrIdxCache map[string]int32) {
	keyIdx := stringTableIndex(inDict.StringTable(), attrProcessPID)
	if keyIdx < 0 {
		p.recordDrop(ctx)
		p.logger.Debug("dropping profile resource without process.pid")
		return
	}
	attrTable := inDict.AttributeTable()
	registeredPIDs := make(map[uint32]string)
	orderedPIDs := make([]uint32, 0)
	seenAnyPID := false

	sps := rp.ScopeProfiles()
	for i := 0; i < sps.Len(); i++ {
		profs := sps.At(i).Profiles()
		for j := 0; j < profs.Len(); j++ {
			samples := profs.At(j).Samples()
			for k := 0; k < samples.Len(); k++ {
				pid, ok := samplePID(attrTable, keyIdx, samples.At(k))
				if !ok {
					continue
				}
				seenAnyPID = true
				if _, alreadyRegistered := registeredPIDs[pid]; alreadyRegistered {
					continue
				}
				packed, registered := p.attrCache.get(pid)
				if !registered {
					continue
				}
				registeredPIDs[pid] = packed
				orderedPIDs = append(orderedPIDs, pid)
			}
		}
	}

	if len(orderedPIDs) == 0 {
		p.recordDrop(ctx)
		if seenAnyPID {
			p.logger.Debug("dropping profile resource for unregistered sample pids")
		} else {
			p.logger.Debug("dropping profile resource without process.pid")
		}
		return
	}

	for _, pid := range orderedPIDs {
		p.appendEnrichedResource(ctx, outRps, outDict, rp, pid, registeredPIDs[pid], true, svcAttrIdxCache)
	}
}

func samplePID(attrTable pprofile.KeyValueAndUnitSlice, keyIdx int32, sample pprofile.Sample) (uint32, bool) {
	indices := sample.AttributeIndices()
	for a := 0; a < indices.Len(); a++ {
		idx := indices.At(a)
		if idx < 0 || int(idx) >= attrTable.Len() {
			continue
		}
		kv := attrTable.At(int(idx))
		if kv.KeyStrindex() == keyIdx && kv.Value().Type() == pcommon.ValueTypeInt {
			return uint32(kv.Value().Int()), true
		}
	}
	return 0, false
}

func (p *vmProfileAttrsProcessor) appendEnrichedResource(ctx context.Context, outRps pprofile.ResourceProfilesSlice, outDict pprofile.ProfilesDictionary, rp pprofile.ResourceProfiles, pid uint32, packed string, filterBySamplePID bool, svcAttrIdxCache map[string]int32) bool {
	dest := pprofile.NewResourceProfiles()
	rp.CopyTo(dest)

	attrs := dest.Resource().Attributes()
	if err := applyPackedResourceAttributes(attrs, packed); err != nil {
		p.recordDrop(ctx)
		p.logger.Debug("dropping profile resource after failed attribute enrichment",
			zap.Uint32("pid", pid),
			zap.Error(err))
		return false
	}
	// Hoist the PID onto the resource so downstream consumers (the in-agent sink cache key
	// and flamegraph attribution) have a stable resource-level identity even for the
	// container-grouped layout where the profiler only set container.id on the resource.
	attrs.PutInt(attrProcessPID, int64(pid))

	if filterBySamplePID && !keepOnlySamplesForPID(outDict, dest, pid) {
		p.recordDrop(ctx)
		p.logger.Debug("dropping profile resource after sample pid filtering removed all samples",
			zap.Uint32("pid", pid))
		return false
	}
	if svc, ok := dest.Resource().Attributes().Get(attrServiceName); ok {
		propagateServiceNameToSamples(outDict, dest, svc.AsString(), svcAttrIdxCache)
	}
	dest.CopyTo(outRps.AppendEmpty())
	return true
}

func keepOnlySamplesForPID(dict pprofile.ProfilesDictionary, rp pprofile.ResourceProfiles, pid uint32) bool {
	keyIdx := stringTableIndex(dict.StringTable(), attrProcessPID)
	if keyIdx < 0 {
		return false
	}
	attrTable := dict.AttributeTable()

	scopeProfiles := rp.ScopeProfiles()
	for i := 0; i < scopeProfiles.Len(); i++ {
		profiles := scopeProfiles.At(i).Profiles()
		profiles.RemoveIf(func(profile pprofile.Profile) bool {
			profile.Samples().RemoveIf(func(sample pprofile.Sample) bool {
				samplePIDValue, ok := samplePID(attrTable, keyIdx, sample)
				return !ok || samplePIDValue != pid
			})
			return profile.Samples().Len() == 0
		})
	}
	scopeProfiles.RemoveIf(func(sp pprofile.ScopeProfiles) bool {
		return sp.Profiles().Len() == 0
	})
	return scopeProfiles.Len() > 0
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
