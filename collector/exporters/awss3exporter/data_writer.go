package awss3exporter

import "context"

type DataWriter interface {
	WriteBuffer(ctx context.Context, buf []byte, config *Config, metadata string, format string) error
}
