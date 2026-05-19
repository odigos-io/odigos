package transport

import (
	"encoding/binary"
	"fmt"
	"io"
)

const maxFrameSize = 4 << 20 // 4 MiB

// ReadFrame reads a length-prefixed protobuf payload from r.
func ReadFrame(r io.Reader) ([]byte, error) {
	var lengthBuf [4]byte
	if _, err := io.ReadFull(r, lengthBuf[:]); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lengthBuf[:])
	if length == 0 {
		return nil, fmt.Errorf("empty frame")
	}
	if length > maxFrameSize {
		return nil, fmt.Errorf("frame size %d exceeds max %d", length, maxFrameSize)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// WriteFrame writes a length-prefixed payload to w.
func WriteFrame(w io.Writer, payload []byte) error {
	if len(payload) > maxFrameSize {
		return fmt.Errorf("payload size %d exceeds max %d", len(payload), maxFrameSize)
	}
	var lengthBuf [4]byte
	binary.BigEndian.PutUint32(lengthBuf[:], uint32(len(payload)))
	if _, err := w.Write(lengthBuf[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
