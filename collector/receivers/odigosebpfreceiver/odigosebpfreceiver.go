package odigosebpfreceiver

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/ringbuf"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
)

const TestMapPath = "/sys/fs/bpf/odigos/test_map"

type receiverImpl struct {
	config *Config
	cancel context.CancelFunc
	next   consumer.Traces
}

func (r *receiverImpl) Start(ctx context.Context, host component.Host) error {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	go func() {
		if err := r.readLoop(ctx); err != nil {
			// TODO: handle error
		}
	}()
	return nil
}

func (r *receiverImpl) Shutdown(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

func (r *receiverImpl) readLoop(ctx context.Context) error {
	m, err := ebpf.LoadPinnedMap(TestMapPath, nil)
	if err != nil {
		return fmt.Errorf("failed to load pinned map: %w", err)
	}
	defer m.Close()

	reader, err := ringbuf.NewReader(m)
	if err != nil {
		return fmt.Errorf("failed to open perf reader: %w", err)
	}
	defer reader.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			rec, err := reader.Read()
			if err != nil {
				if errors.Is(err, ringbuf.ErrClosed) {
					return nil
				}
				continue
			}

			if len(rec.RawSample) < 8 {
				continue
			}

			spanLen := binary.NativeEndian.Uint64(rec.RawSample[:8])
			if len(rec.RawSample) < int(8+spanLen) {
				continue
			}

			var span tracepb.ResourceSpans
			err = proto.Unmarshal(rec.RawSample[8:8+spanLen], &span)
			if err != nil {
				continue
			}

			err = r.next.ConsumeTraces(ctx, traces)
			if err != nil {
				continue
			}
		}
	}
}
