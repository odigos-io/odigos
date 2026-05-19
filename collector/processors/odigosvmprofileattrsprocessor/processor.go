package odigosvmprofileattrsprocessor

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/odigos-io/odigos/common/unixfd"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.uber.org/zap"
)

const (
	attrProcessPID       = "process.pid"
	attrServiceName      = "service.name"
	unknownServicePrefix = "unknown_service"
)

type vmProfileAttrsProcessor struct {
	logger    *zap.Logger
	cfg       *Config
	attrCache *profileAttrCache

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func (p *vmProfileAttrsProcessor) capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (p *vmProfileAttrsProcessor) start(ctx context.Context, _ component.Host) error {
	runCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		err := unixfd.ConnectAndListenProfileAttrs(runCtx, p.cfg.SocketPath, p.logger, func(line string) {
			p.attrCache.applyEvent(line)
		})
		if err != nil && runCtx.Err() == nil {
			p.logger.Error("profiles attr unix client stopped", zap.Error(err))
		}
	}()

	return nil
}

func (p *vmProfileAttrsProcessor) shutdown(context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	return nil
}

// processProfiles keeps only resource profiles whose process.pid is registered in the VM agent
// cache, enriches them from the streamed attribute map, and drops everything else.
func (p *vmProfileAttrsProcessor) processProfiles(_ context.Context, profiles pprofile.Profiles) (pprofile.Profiles, error) {
	rps := profiles.ResourceProfiles()
	if rps.Len() == 0 {
		return profiles, nil
	}

	out := pprofile.NewProfiles()
	profiles.Dictionary().CopyTo(out.Dictionary())
	outRps := out.ResourceProfiles()

	for i := 0; i < rps.Len(); i++ {
		rp := rps.At(i)
		attrs := rp.Resource().Attributes()

		pidVal, ok := attrs.Get(attrProcessPID)
		if !ok {
			p.logger.Debug("dropping profile resource without process.pid")
			continue
		}
		pid := uint32(pidVal.Int())

		packed, registered := p.attrCache.get(pid)
		if !registered {
			p.logger.Debug("dropping profile resource for unregistered pid",
				zap.Uint32("pid", pid))
			continue
		}

		if err := applyPackedResourceAttributes(attrs, packed); err != nil {
			p.logger.Debug("dropping profile resource after failed attribute enrichment",
				zap.Uint32("pid", pid),
				zap.Error(err))
			continue
		}

		dest := outRps.AppendEmpty()
		rp.CopyTo(dest)
		if svc, ok := dest.Resource().Attributes().Get(attrServiceName); ok {
			propagateServiceNameToSamples(out.Dictionary(), dest, svc.AsString())
		}
	}

	return out, nil
}

// applyPackedResourceAttributes parses "key:value,key:value" into resource attributes.
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
		if key == attrServiceName {
			if existing, ok := resourceAttrs.Get(attrServiceName); ok {
				name := existing.AsString()
				if name != "" && !strings.HasPrefix(name, unknownServicePrefix) {
					continue
				}
			}
		}
		resourceAttrs.PutStr(key, val)
		parsed = true
	}
	if !parsed {
		return fmt.Errorf("no valid attributes parsed from: %s", attributesStr)
	}
	return nil
}
