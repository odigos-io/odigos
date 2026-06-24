//go:build linux

// resolver_linux.go is the Linux resolver: it adapts the internal symbolize
// package (the real on-host symbolizer) to the processor's resolver interface.
package odigossymbolizeprocessor

import (
	"time"

	"go.uber.org/zap"

	"github.com/odigos-io/odigos/collector/processors/odigossymbolizeprocessor/internal/symbolize"
)

// nativeResolver symbolizes native frames against the on-disk ELF of a
// process's loaded modules, via the internal symbolize package. Symbol parsing
// is asynchronous (background workers), so resolve never blocks the pipeline.
type nativeResolver struct {
	s *symbolize.Symbolizer
}

func newResolver(cfg *Config, logger *zap.Logger) resolver {
	opts := []symbolize.Option{symbolize.WithLogger(logger)}
	if cfg != nil {
		opts = append(opts,
			symbolize.WithMaxSymbolCache(cfg.MaxSymbolCache),
			symbolize.WithMaxSymbolBytes(cfg.MaxSymbolBytes),
			symbolize.WithMaxMapsCache(cfg.MaxMapsCache),
			symbolize.WithParseWorkers(cfg.ParseWorkers),
		)
		if cfg.MapsTTLSeconds > 0 {
			opts = append(opts, symbolize.WithMapsTTL(time.Duration(cfg.MapsTTLSeconds)*time.Second))
		}
	}
	return &nativeResolver{s: symbolize.New(opts...)}
}

func (r *nativeResolver) resolve(pid int64, m moduleRef, addr uint64) (name, source string, ok bool) {
	f, found := r.s.Resolve(int(pid), symbolize.Mapping{
		Name:        m.Name,
		MemoryStart: m.MemoryStart,
		FileOffset:  m.FileOffset,
		BuildID:     m.BuildID,
	}, addr)
	if !found {
		return "", "", false
	}
	return f.Name, f.Source, true
}

func (r *nativeResolver) prewarm(pid int64) { r.s.PreWarm(int(pid)) }
func (r *nativeResolver) close()            { r.s.Close() }
