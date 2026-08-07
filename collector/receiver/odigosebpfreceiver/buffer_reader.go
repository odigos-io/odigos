package odigosebpfreceiver

import (
	"errors"
	"os"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf/ringbuf"
	"go.uber.org/zap"
)

// BufferRecord represents a record read from either perf or ring buffer
type BufferRecord struct {
	RawSample   []byte
	LostSamples uint64
}

// BufferReader provides a unified interface for reading from eBPF maps
// regardless of whether they are perf buffers or ring buffers
type BufferReader interface {
	// ReadInto reads the next record from the buffer
	ReadInto(record *BufferRecord) error
	// Close closes the reader and releases resources
	Close() error
}

// perfBufferReader implements BufferReader for perf event arrays
type perfBufferReader struct {
	reader *perf.Reader
}

func (p *perfBufferReader) ReadInto(record *BufferRecord) error {
	var perfRecord perf.Record
	err := p.reader.ReadInto(&perfRecord)
	if err != nil {
		return err
	}

	record.RawSample = perfRecord.RawSample
	record.LostSamples = perfRecord.LostSamples
	return nil
}

func (p *perfBufferReader) Close() error {
	return p.reader.Close()
}

// ringBufferReader implements BufferReader for ring buffers
type ringBufferReader struct {
	reader *ringbuf.Reader
}

func (r *ringBufferReader) ReadInto(record *BufferRecord) error {
	ringRecord, err := r.reader.Read()
	if err != nil {
		return err
	}

	record.RawSample = ringRecord.RawSample
	record.LostSamples = 0 // Ring buffers don't have lost samples in the same way
	return nil
}

func (r *ringBufferReader) Close() error {
	return r.reader.Close()
}

// NewBufferReader creates the appropriate BufferReader based on the eBPF map type
func NewBufferReader(m *ebpf.Map, logger *zap.Logger) (BufferReader, error) {
	info, err := m.Info()
	if err != nil {
		return nil, err
	}

	switch info.Type {
	case ebpf.PerfEventArray:
		logger.Debug("Creating perf buffer reader")
		reader, err := perf.NewReader(m, numOfPages*os.Getpagesize())
		if err != nil {
			return nil, err
		}
		return &perfBufferReader{reader: reader}, nil

	case ebpf.RingBuf:
		logger.Debug("Creating ring buffer reader")
		reader, err := ringbuf.NewReader(m)
		if err != nil {
			return nil, err
		}
		return &ringBufferReader{reader: reader}, nil

	default:
		return nil, errors.New("unsupported map type for buffer reading")
	}
}

// Helper function to check if an error indicates the reader was closed
func IsClosedError(err error) bool {
	return errors.Is(err, perf.ErrClosed) ||
		errors.Is(err, ringbuf.ErrClosed)
}
