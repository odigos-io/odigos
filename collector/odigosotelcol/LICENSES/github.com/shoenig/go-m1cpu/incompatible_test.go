//go:build !darwin || !arm64 || !cgo

package m1cpu

import (
	"testing"

	"github.com/shoenig/test/must"
)

const (
	message = "m1cpu: not a darwin/arm64 system"
)

func panics(f func()) (message string) {
	defer func() {
		if r := recover(); r != nil {
			message = r.(string)
		}
	}()
	f()
	return
}

func Test_IsAppleSilicon(t *testing.T) {
	result := IsAppleSilicon()
	must.False(t, result)
}

func check(t *testing.T, f func()) {
	t.Helper()
	result := panics(f)
	must.Eq(t, result, message)
}

func Test_PCoreHz(t *testing.T) {
	check(t, func() { _ = PCoreHz() })
}

func Test_ECoreHz(t *testing.T) {
	check(t, func() { _ = ECoreHz() })
}

func Test_PCoreGHz(t *testing.T) {
	check(t, func() { _ = PCoreGHz() })
}

func Test_ECoreGHz(t *testing.T) {
	check(t, func() { _ = ECoreGHz() })
}

func Test_PCoreCount(t *testing.T) {
	check(t, func() { _ = PCoreCount() })
}

func Test_ECoreCount(t *testing.T) {
	check(t, func() { _ = ECoreCount() })
}

func Test_PCoreCache(t *testing.T) {
	check(t, func() { _, _, _ = PCoreCache() })
}

func Test_ECoreCache(t *testing.T) {
	check(t, func() { _, _, _ = ECoreCache() })
}

func Test_ModelName(t *testing.T) {
	check(t, func() { _ = ModelName() })
}
