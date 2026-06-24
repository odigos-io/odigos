//go:build linux

package symbolizeserver

import (
	"bytes"
	"context"
	"debug/elf"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestServerSymbolizesOverSocket is the end-to-end proof: a real compiled .so is
// "mapped" by a fake process, the server is asked over its unix socket to
// symbolize an address in it, and returns the demangled name + source — the exact
// path the collector takes, with no ELF work in the collector.
func TestServerSymbolizesOverSocket(t *testing.T) {
	if _, err := exec.LookPath("g++"); err != nil {
		t.Skip("g++ not available")
	}
	// 1. Build a .so with a known C++ function + a GNU build-id.
	dir := t.TempDir()
	src := filepath.Join(dir, "lib.cpp")
	so := filepath.Join(dir, "libfix.so")
	const code = `namespace soapcommand { struct C { int onResume(); };
	  int C::onResume(){ volatile int s=0; for(int i=0;i<5;i++) s+=i; return s; } }
	  extern "C" int entry(){ soapcommand::C c; return c.onResume(); }`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("g++", "-shared", "-fPIC", "-O0", "-g0", "-Wl,--build-id", "-o", so, src).CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}
	name, vaddr := elfFacts(t, so)

	// 2. Fake /proc: pid 5555 maps libfix.so. memStart=0 ⇒ addr is treated as a vaddr.
	pid := 5555
	root := filepath.Join(dir, "proc")
	if err := os.MkdirAll(filepath.Join(root, fmt.Sprint(pid)), 0o755); err != nil {
		t.Fatal(err)
	}
	maps := "0-1000 r-xp 00000000 08:01 1 " + so + "\n"
	if err := os.WriteFile(filepath.Join(root, fmt.Sprint(pid), "maps"), []byte(maps), 0o644); err != nil {
		t.Fatal(err)
	}
	// /proc/<pid>/root → "/" so the recorded absolute path resolves.
	if err := os.Symlink("/", filepath.Join(root, fmt.Sprint(pid), "root")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOST_PROC", root)

	// 3. Start the server on a temp socket.
	sock := filepath.Join(dir, "sym.sock")
	srv := New(sock, zap.NewNop())
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Close(context.Background())

	client := socketClient(sock)
	// BuildID "" skips verification for the resolve check; the wrong-id refuse below
	// exercises strict verification against the .so's real build-id.
	frame := Frame{PID: pid, Module: "libfix.so", MemoryStart: 0, FileOffset: 0, BuildID: "", Addr: vaddr}

	// 4. Poll: parsing is async server-side, so the first call may be "pending".
	deadline := time.Now().Add(5 * time.Second)
	for {
		got := postSymbolize(t, client, sock, frame)
		if got.Name != "" {
			if got.Name != name && got.Name != "soapcommand::C::onResume()" {
				t.Fatalf("name = %q, want %q", got.Name, name)
			}
			if got.Source != "symtab" && got.Source != "dynsym" {
				t.Fatalf("source = %q, want symtab/dynsym", got.Source)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server never resolved the frame")
		}
		time.Sleep(20 * time.Millisecond)
	}

	// 5. A wrong build-id must be refused (strict verification through the server).
	bad := frame
	bad.BuildID = "00bad00"
	if got := postSymbolize(t, client, sock, bad); got.Name != "" {
		t.Errorf("build-id mismatch must not resolve, got %q", got.Name)
	}
}

func postSymbolize(t *testing.T, c *http.Client, sock string, f Frame) Resolved {
	t.Helper()
	body, _ := json.Marshal(symbolizeRequest{Frames: []Frame{f}})
	resp, err := c.Post("http://unix/symbolize", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out symbolizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Frames) != 1 {
		t.Fatalf("want 1 frame, got %d", len(out.Frames))
	}
	return out.Frames[0]
}

func socketClient(sock string) *http.Client {
	return &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", sock)
		},
	}}
}

// elfFacts returns a function's demangled name and its virtual address.
func elfFacts(t *testing.T, path string) (name string, vaddr uint64) {
	t.Helper()
	f, err := elf.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	for _, read := range []func() ([]elf.Symbol, error){f.Symbols, f.DynamicSymbols} {
		syms, _ := read()
		for _, s := range syms {
			if elf.ST_TYPE(s.Info) == elf.STT_FUNC && s.Value != 0 && bytesContains(s.Name, "onResume") {
				vaddr = s.Value
			}
		}
	}
	if vaddr == 0 {
		t.Fatal("onResume symbol not found")
	}
	return "soapcommand::C::onResume()", vaddr
}

func bytesContains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
