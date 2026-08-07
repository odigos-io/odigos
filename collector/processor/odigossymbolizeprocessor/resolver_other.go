//go:build !linux

package odigossymbolizeprocessor

import "go.uber.org/zap"

// noopResolver is used on non-Linux platforms, where /proc and the target
// binaries aren't available. The processor still builds and passes profiles
// through unchanged.
type noopResolver struct{}

func newResolver(*Config, *zap.Logger) resolver { return noopResolver{} }

func (noopResolver) resolve(int64, moduleRef, uint64) (name, source string, ok bool) {
	return "", "", false
}
func (noopResolver) prewarm(int64) {}
func (noopResolver) close()        {}
