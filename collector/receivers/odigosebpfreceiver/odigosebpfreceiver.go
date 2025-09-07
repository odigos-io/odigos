package odigosebpfreceiver

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"github.com/fsnotify/fsnotify"
	"github.com/odigos-io/odigos/common/unixfd"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
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

		var (
			prevMap    *ebpf.Map
			readersWg  sync.WaitGroup
			cancelPrev context.CancelFunc
		)

		for {
			select {
			case <-ctx.Done():
				r.logger.Info("receiver supervisor exiting")
				if cancelPrev != nil {
					cancelPrev()
					readersWg.Wait()
				}
				if prevMap != nil {
					prevMap.Close()
				}
				return
			default:
			}

			r.logger.Info("supervisor requesting new FD from odiglet",
				zap.String("socket", unixfd.DefaultSocketPath))

			fd, err := unixfd.WaitForFD(unixfd.DefaultSocketPath)
			if err != nil {
				r.logger.Warn("failed to wait for FD", zap.Error(err))
				time.Sleep(2 * time.Second)
				continue
			}

			newMap, err := ebpf.NewMapFromFD(fd)
			if err != nil {
				r.logger.Warn("failed to create map from FD", zap.Error(err))
				time.Sleep(2 * time.Second)
				continue
			}
			r.logger.Info("supervisor received NEW_FD and created map")

			// Tear down old generation
			if cancelPrev != nil {
				cancelPrev()
				readersWg.Wait()
				cancelPrev = nil
				if prevMap != nil {
					prevMap.Close()
					prevMap = nil
				}
			}

			// Start readers for new map
			readersCtx, readersCancel := context.WithCancel(ctx)
			cancelPrev = readersCancel
			prevMap = newMap

			const numReaders = 1
			for i := 0; i < numReaders; i++ {
				readersWg.Add(1)
				go func(id int, m *ebpf.Map) {
					defer readersWg.Done()
					r.logger.Info("starting reader", zap.Int("reader_id", id))
					if err := r.readLoop(readersCtx, m); err != nil {
						r.logger.Error("read loop failed", zap.Int("reader_id", id), zap.Error(err))
					}
					r.logger.Info("reader exited", zap.Int("reader_id", id))
				}(i, newMap)
			}
		}
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

func (r *ebpfReceiver) waitForSocketFD(ctx context.Context, socketPath string) (*ebpf.Map, error) {
	raddr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("resolve unix addr: %w", err)
	}

	// Quick path: try to dial immediately.
	if m, err := r.tryDialAndRecv(raddr); err == nil {
		return m, nil
	} else if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ECONNREFUSED) {
		// A "real" error, not just "not ready yet"
		return nil, err
	}

	r.logger.Info("socket not ready; waiting for creation", zap.String("path", socketPath))

	dir := filepath.Dir(socketPath)

	// Try fsnotify
	watcher, wErr := fsnotify.NewWatcher()
	if wErr == nil {
		if addErr := watcher.Add(dir); addErr != nil {
			r.logger.Warn("fsnotify watcher failed; falling back to polling",
				zap.String("dir", dir), zap.Error(addErr))
			_ = watcher.Close()
			watcher = nil
		}
	} else {
		r.logger.Warn("fsnotify not available; falling back to polling", zap.Error(wErr))
	}

	// If watcher works, wait on events
	if watcher != nil {
		defer watcher.Close()
		for {
			// Try before blocking, in case we missed the event
			if m, err := r.tryDialAndRecv(raddr); err == nil {
				return m, nil
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case ev := <-watcher.Events:
				if ev.Name == socketPath && ev.Op&fsnotify.Create != 0 {
					if m, err := r.tryDialAndRecv(raddr); err == nil {
						r.logger.Info("socket connected and FD received", zap.String("path", socketPath))
						return m, nil
					}
				}
			case werr := <-watcher.Errors:
				if werr != nil {
					r.logger.Warn("fsnotify error; switching to polling", zap.Error(werr))
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

	// Fallback polling loop
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
			if m, err := r.tryDialAndRecv(raddr); err == nil {
				r.logger.Info("socket connected and FD received (polling)", zap.String("path", socketPath))
				return m, nil
			}
		}
	}
}

func (r *ebpfReceiver) tryDialAndRecv(raddr *net.UnixAddr) (*ebpf.Map, error) {
	conn, err := net.DialUnix("unix", nil, raddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	fd, err := recvFD(conn)
	if err != nil {
		return nil, err
	}
	return ebpf.NewMapFromFD(fd)
}

func recvFD(c *net.UnixConn) (int, error) {
	buf := make([]byte, 1)
	oob := make([]byte, unix.CmsgSpace(4))

	_, oobn, _, _, err := c.ReadMsgUnix(buf, oob)
	if err != nil {
		return -1, fmt.Errorf("readmsg: %w", err)
	}

	msgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return -1, fmt.Errorf("parse scm: %w", err)
	}

	if len(msgs) != 1 {
		return -1, fmt.Errorf("expected 1 control message got %d", len(msgs))
	}

	fds, err := unix.ParseUnixRights(&msgs[0])
	if err != nil {
		return -1, fmt.Errorf("parse rights: %w", err)
	}
	if len(fds) == 0 {
		return -1, fmt.Errorf("no fd received")
	}

	return fds[0], nil
}
