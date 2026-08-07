package odigossymbolizeprocessor

// moduleRef is the subset of an OTLP profile Mapping the resolver needs to open
// and address the on-disk binary.
type moduleRef struct {
	Name        string // module basename (e.g. "libCXOPSX00.so")
	MemoryStart uint64
	FileOffset  uint64
	BuildID     string // expected GNU build-id (hex), or "" to skip verification
}

// resolver turns a native instruction address into a function name and the ELF
// symbol table it came from ("symtab"/"dynsym"). It is implemented by the on-host
// symbolizer on Linux and a no-op elsewhere, so the processor compiles on every
// platform while only symbolizing where /proc and the target binaries are available.
//
// The source is emitted as a Location attribute so downstream tooling can tell a
// native symbol resolved from the *live binary* (instrumentable — OBI can attach a
// uprobe by signature) apart from a frame the profiler named (kernel/interpreted).
//
// Symbol-table parsing happens on background workers inside the symbolizer, so
// resolve never blocks the pipeline; a brand-new binary resolves on a later
// batch. prewarm primes that cache for a freshly-seen process.
type resolver interface {
	resolve(pid int64, m moduleRef, addr uint64) (name, source string, ok bool)
	prewarm(pid int64)
	close()
}
