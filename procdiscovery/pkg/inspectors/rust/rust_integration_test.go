package rust

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestPanicStringDetectionOnRealBinary(t *testing.T) {
	strippedBinaryPath := "/tmp/rust-stripped"
	unstrippedBinaryPath := "/tmp/rust-unstripped"

	if _, err := os.Stat(strippedBinaryPath); os.IsNotExist(err) {
		t.Skip("Stripped binary not found at /tmp/rust-stripped - run test.sh first")
	}
	if _, err := os.Stat(unstrippedBinaryPath); os.IsNotExist(err) {
		t.Skip("Unstripped binary not found at /tmp/rust-unstripped - run test.sh first")
	}

	t.Run("unstripped binary should be detected via symbols", func(t *testing.T) {
		content, err := os.ReadFile(unstrippedBinaryPath)
		if err != nil {
			t.Fatalf("Failed to read unstripped binary: %v", err)
		}

		found := containsRustPanicStrings(content)
		if !found {
			t.Error("Expected to find Rust panic strings in unstripped binary")
		}
	})

	t.Run("stripped binary should be detected via panic strings", func(t *testing.T) {
		content, err := os.ReadFile(strippedBinaryPath)
		if err != nil {
			t.Fatalf("Failed to read stripped binary: %v", err)
		}

		found := containsRustPanicStrings(content)
		if !found {
			t.Error("Expected to find Rust panic strings in stripped binary - detection will fail!")
		} else {
			t.Log("✅ Stripped binary detected via panic strings")
		}
	})
}

func containsRustPanicStrings(content []byte) bool {
	rustPanicStrings := []string{
		"panicked at",
		"called `Option::unwrap()` on a `None` value",
		"called `Result::unwrap()` on an `Err` value",
		"/rustc/",
		".cargo/registry",
	}

	for _, pattern := range rustPanicStrings {
		if bytes.Contains(content, []byte(pattern)) {
			return true
		}
	}
	return false
}

type mockProcessFile struct {
	*bytes.Reader
}

func (m *mockProcessFile) Seek(offset int64, whence int) (int64, error) {
	return m.Reader.Seek(offset, whence)
}

func (m *mockProcessFile) Read(p []byte) (int, error) {
	return m.Reader.Read(p)
}

func (m *mockProcessFile) ReadAt(p []byte, off int64) (n int, err error) {
	return m.Reader.ReadAt(p, off)
}

func (m *mockProcessFile) Close() error {
	return nil
}

func TestCheckPanicStringsMethod(t *testing.T) {
	strippedBinaryPath := "/tmp/rust-stripped"

	if _, err := os.Stat(strippedBinaryPath); os.IsNotExist(err) {
		t.Skip("Stripped binary not found - run test.sh first")
	}

	content, err := os.ReadFile(strippedBinaryPath)
	if err != nil {
		t.Fatalf("Failed to read binary: %v", err)
	}

	mockFile := &mockProcessFile{Reader: bytes.NewReader(content)}

	inspector := &RustInspector{}
	found := inspector.checkPanicStrings(mockFile)

	if !found {
		t.Error("checkPanicStrings should detect stripped Rust binary")
	} else {
		t.Log("✅ checkPanicStrings successfully detects stripped Rust binary")
	}
}

func TestVersionExtraction(t *testing.T) {
	strippedBinaryPath := "/tmp/rust-stripped"

	if _, err := os.Stat(strippedBinaryPath); os.IsNotExist(err) {
		t.Skip("Binary not found - run test.sh first")
	}

	content, err := os.ReadFile(strippedBinaryPath)
	if err != nil {
		t.Fatalf("Failed to read binary: %v", err)
	}

	rustcVersionPrefix := []byte("/rustc/")
	if idx := bytes.Index(content, rustcVersionPrefix); idx != -1 {
		end := idx + len(rustcVersionPrefix) + 50
		if end > len(content) {
			end = len(content)
		}
		versionBytes := content[idx : end]
		nullIdx := bytes.IndexByte(versionBytes[len(rustcVersionPrefix):], 0)
		if nullIdx == -1 {
			nullIdx = 40
		}
		version := string(versionBytes[len(rustcVersionPrefix) : len(rustcVersionPrefix)+nullIdx])
		t.Logf("✅ Extracted rustc version/commit: %s", version)
	} else {
		t.Log("⚠️ /rustc/ path not found in binary")
	}
}

type seekerReader struct {
	io.ReadSeeker
}

func (s *seekerReader) ReadAt(p []byte, off int64) (n int, err error) {
	_, err = s.Seek(off, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return s.Read(p)
}

func (s *seekerReader) Close() error {
	return nil
}

