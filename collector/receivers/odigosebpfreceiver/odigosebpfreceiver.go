package odigosebpfreceiver

import (
	"context"
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"github.com/fsnotify/fsnotify"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	numOfPages      = 2048
	PollingInterval = 10 * time.Second
)

type ebpfReceiver struct {
	config  *Config
	cancel  context.CancelFunc
	logger  *zap.Logger
	mapPath string

	// Consumers
	nextTraces  consumer.Traces
	nextMetrics consumer.Metrics
	nextLogs    consumer.Logs

	// WaitGroup to wait for all goroutines to finish
	wg sync.WaitGroup
}

func (r *ebpfReceiver) Start(ctx context.Context, host component.Host) error {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		// Step 1: wait for the map once
		m, err := r.waitForPinnedMap(ctx)
		if err != nil {
			r.logger.Warn("odigos-ebpf: failed to wait for pinned map", zap.Error(err))
			return
		}

		// Step 2: start N concurrent readers using this map
		const numReaders = 1

		var readersWg sync.WaitGroup

		for i := range numReaders {
			readersWg.Add(1)
			go func(id int) {
				defer readersWg.Done()
				if err := r.readLoop(ctx, m); err != nil {
					r.logger.Error("read loop failed", zap.Int("reader_id", id), zap.Error(err))
				}
			}(i)
		}

		// Step 3: wait for all readers to finish before closing the map
		readersWg.Wait()
		m.Close()
		r.logger.Info("odigos-ebpf: all readers finished, map closed", zap.String("mapPath", r.mapPath))
	}()

	return nil
}

func (r *ebpfReceiver) Shutdown(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	done := make(chan struct{})

	// WaitGroup wait in a goroutine to support context cancellation
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		r.logger.Warn("odigos-ebpf: receiver shutdown did not finish before context was canceled",
			zap.String("mapPath", r.mapPath))
		return ctx.Err()
	case <-done:
		r.logger.Info("odigos-ebpf: receiver shutdown complete", zap.String("mapPath", r.mapPath))
		return nil // all goroutines exited gracefully
	}
}

// tryToLoadPinnedMap attempts to load the pinned map from bpffs.
func (r *ebpfReceiver) tryToLoadPinnedMap() (*ebpf.Map, error) {
	m, err := ebpf.LoadPinnedMap(r.mapPath, nil)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// waitForPinnedMap waits until the pinned map exists.
// If fsnotify is available on the directory, it waits on events.
// If fsnotify setup fails (or later errors), it falls back to 10s polling forever.
func (r *ebpfReceiver) waitForPinnedMap(ctx context.Context) (*ebpf.Map, error) {
	// Quick path.
	if m, err := r.tryToLoadPinnedMap(); err == nil {
		return m, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		// Real error, not just "doesn't exist yet".
		return nil, err
	}

	r.logger.Info("pinned map not found; waiting for creation", zap.String("path", r.mapPath))

	// We watch the root bpffs mount (/sys/fs/bpf) instead of the exact pinned map path
	// because fsnotify cannot watch a file that does not yet exist. The pinned map file
	// (/sys/fs/bpf/odiglet/traces) will be created dynamically by the odiglet.
	// By watching the root directory, we can receive a fsnotify.Create event when the
	// target file is eventually pinned anywhere under it, and then check if the path
	// matches our target map before attempting to load it.
	dir := "/sys/fs/bpf"

	// Try to set up fsnotify.
	watcher, wErr := fsnotify.NewWatcher()
	if wErr == nil {
		if addErr := watcher.Add(dir); addErr != nil {
			r.logger.Warn("fsnotify watcher failed; falling back to 10s polling", zap.String("dir", dir), zap.Error(addErr))
			_ = watcher.Close()
			watcher = nil
		}
	} else {
		r.logger.Warn("fsnotify not available; falling back to 10s polling", zap.Error(wErr))
	}

	// If watcher works, wait on events; otherwise, poll every 10s.
	if watcher != nil {
		defer watcher.Close()
		for {
			// Try before blocking, in case we missed the event race.
			if m, err := r.tryToLoadPinnedMap(); err == nil {
				return m, nil
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case ev := <-watcher.Events:
				// React when the map file is created, otherwise waiting for the next event.
				if ev.Name == filepath.Dir(r.mapPath) && (ev.Op&(fsnotify.Create)) != 0 {
					if _, err := os.Stat(r.mapPath); err == nil {
						if m, err := r.tryToLoadPinnedMap(); err == nil {
							r.logger.Info("pinned map found and loaded", zap.String("path", r.mapPath))
							return m, nil
						}
					}
				}
			case werr := <-watcher.Errors:
				if werr != nil {
					r.logger.Warn("fsnotify error; switching to 10s polling", zap.Error(werr))
					// break to polling loop below
					watcher.Close()
					watcher = nil
					break
				}
			}
			if watcher == nil {
				break
			}
		}
	}

	// Polling fallback: 10s interval, indefinitely.
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(PollingInterval):
			if m, err := r.tryToLoadPinnedMap(); err == nil {
				r.logger.Info("pinned map found and loaded", zap.String("path", r.mapPath))
				return m, nil
			}
		}
	}
}
func (r *ebpfReceiver) readLoop(ctx context.Context, m *ebpf.Map) error {
	reader, err := perf.NewReader(m, numOfPages*os.Getpagesize())
	if err != nil {
		r.logger.Error("failed to open perf reader", zap.Error(err))
		return err
	}
	defer reader.Close()

	var record perf.Record

	go func() {
		<-ctx.Done()
		// This will unblock the blocking call to reader.ReadInto()
		reader.Close()
	}()

	for {
		// This blocks until data is available or reader is closed
		err := reader.ReadInto(&record)
		if err != nil {
			if errors.Is(err, perf.ErrClosed) {
				// Closed due to shutdown signal — exit gracefully
				return nil
			}
			r.logger.Error("error reading from perf reader", zap.Error(err))
			continue
		}

		if record.LostSamples != 0 {
			r.logger.Error("lost samples", zap.Int("lost", int(record.LostSamples)))
			continue
		}

		if len(record.RawSample) < 8 {
			continue
		}

		acceptedLength := binary.NativeEndian.Uint64(record.RawSample[:8])
		if len(record.RawSample) < (8 + int(acceptedLength)) {
			continue // incomplete record
		}

		// Attempt to unmarshal into OpenTelemetry Traces format
		protoUnmarshaler := ptrace.ProtoUnmarshaler{}
		td, err := protoUnmarshaler.UnmarshalTraces(record.RawSample[8 : 8+acceptedLength])
		if err != nil {
			// Fallback: try unmarshalling legacy trace format
			var span tracepb.ResourceSpans
			err = proto.Unmarshal(record.RawSample[8:8+acceptedLength], &span)
			if err != nil {
				r.logger.Error("error unmarshalling span", zap.Error(err))
				continue
			}
			td = convertResourceSpansToPdata(&span)
		}

		err = r.nextTraces.ConsumeTraces(ctx, td)
		if err != nil {
			r.logger.Error("err consuming traces", zap.Error(err))
			continue
		}
	}
}

// convertResourceSpansToPdata converts a single ResourceSpans to pdata Traces
// This is here to support the old version of the agent that doesn't support the new format.
// TODO: remove this once we are sure that all the agents are updated to the new format.
func convertResourceSpansToPdata(resourceSpans *tracepb.ResourceSpans) ptrace.Traces {
	// Wrap single ResourceSpans in TracesData
	tracesData := &tracepb.TracesData{
		ResourceSpans: []*tracepb.ResourceSpans{resourceSpans},
	}

	// Marshal to bytes
	data, err := proto.Marshal(tracesData)
	if err != nil {
		return ptrace.NewTraces()
	}

	// Convert to pdata
	unmarshaler := &ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(data)
	if err != nil {
		return ptrace.NewTraces()
	}

	return traces
}
