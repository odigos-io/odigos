package utils

import (
	"bytes"
	"testing"
)

func TestIsMapsFileContainsBinary(t *testing.T) {
	data := buildSyntheticMaps(20, "libpython3.11.so.1.0", 7)

	t.Run("hit", func(t *testing.T) {
		f := memProcessFile{bytes.NewReader(data)}
		if !IsMapsFileContainsBinary(f, []string{"libjvm.so", "libpython"}) {
			t.Fatal("expected hit for libpython")
		}
	})

	t.Run("miss", func(t *testing.T) {
		f := memProcessFile{bytes.NewReader(data)}
		if IsMapsFileContainsBinary(f, []string{"libjvm.so", "libruby.so"}) {
			t.Fatal("expected miss")
		}
	})

	t.Run("empty_binaries", func(t *testing.T) {
		f := memProcessFile{bytes.NewReader(data)}
		if IsMapsFileContainsBinary(f, nil) {
			t.Fatal("expected false for empty binaries")
		}
	})
}
