package symbolize

// This file is the exported API surface over the ported, unexported symbolizer
// logic in symbolize.go (copied verbatim from vm-agent so its proven behavior
// stays identical). The processor package consumes only these exported wrappers.

// Symbolizer resolves native (C/C++/Rust) instruction addresses to function
// names by reading the on-disk binary's symbol sources. It is safe for
// concurrent use: the underlying parse results live in a process-global cache.
type Symbolizer struct {
	inner *symbolizer
}

// New builds a Symbolizer from the given options. With Native set, resolve()
// reads .symtab/.dynsym/MiniDebugInfo/local-debuginfo from on-disk binaries;
// with Native off, Resolve is a no-op that leaves frames raw.
func New(opts SymbolizeOptions) *Symbolizer {
	return &Symbolizer{inner: newSymbolizer(opts)}
}

// Resolve maps a process instruction address (with the mapping's memory-start
// and file-offset) to a fully-described frame in exePath. ok is false when no
// symbol source could name the address — callers must then leave the frame raw.
func (s *Symbolizer) Resolve(exePath string, memStart, fileOffset, addr uint64) (ResolvedFrame, bool) {
	return s.inner.resolve(exePath, memStart, fileOffset, addr)
}
