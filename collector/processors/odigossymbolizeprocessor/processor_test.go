package odigossymbolizeprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.uber.org/zap"
)

// fakeResolver returns a canned name for a known (module, addr) and records the
// pids it was asked about, so tests stay platform-neutral (no /proc, no ELF).
type fakeResolver struct {
	names   map[uint64]string
	gotPID  int64
	gotName string
}

func (f *fakeResolver) resolveBatch(reqs []frameRequest) []frameResult {
	out := make([]frameResult, len(reqs))
	for i, req := range reqs {
		f.gotPID = req.pid
		if n, ok := f.names[req.addr]; ok {
			f.gotName = req.mod.Name
			out[i] = frameResult{name: n, source: "symtab", ok: true}
		}
	}
	return out
}

func (f *fakeResolver) close() {}

// newTestProcessor wires a fake resolver into a processor for platform-neutral tests.
func newTestProcessor(fr resolver) *symbolizeProcessor {
	return &symbolizeProcessor{logger: zap.NewNop(), cfg: &Config{}, resolver: fr}
}

// buildProfiles constructs a one-resource batch: pid 4242, module "libfix.so",
// two native locations (one resolvable at addr 0x1000, one not at 0x2000) plus
// one already-symbolized location that must be left untouched.
func buildProfiles() (pprofile.Profiles, int /*nativeLoc*/, int /*namedLoc*/) {
	pd := pprofile.NewProfiles()
	dict := pd.Dictionary()

	st := dict.StringTable()
	st.Append("")             // 0: sentinel
	st.Append("libfix.so")    // 1: mapping filename
	st.Append("alreadyNamed") // 2: pre-existing function name

	mt := dict.MappingTable()
	m := mt.AppendEmpty()
	m.SetFilenameStrindex(1)
	m.SetMemoryStart(0x7f0000000000)
	m.SetFileOffset(0)

	// pre-existing function for the already-symbolized location.
	ft := dict.FunctionTable()
	fn := ft.AppendEmpty()
	fn.SetNameStrindex(2)

	lt := dict.LocationTable()
	// loc 0: native, resolvable (addr 0x1000)
	l0 := lt.AppendEmpty()
	l0.SetMappingIndex(0)
	l0.SetAddress(0x1000)
	// loc 1: native, NOT resolvable (addr 0x2000)
	l1 := lt.AppendEmpty()
	l1.SetMappingIndex(0)
	l1.SetAddress(0x2000)
	// loc 2: already symbolized (has a Line) — must be left untouched
	l2 := lt.AppendEmpty()
	l2.SetMappingIndex(0)
	l2.SetAddress(0x3000)
	l2.Lines().AppendEmpty().SetFunctionIndex(0)

	stk := dict.StackTable()
	s := stk.AppendEmpty()
	s.LocationIndices().Append(0, 1, 2)

	rp := pd.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutInt("process.pid", 4242)
	prof := rp.ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	sample := prof.Samples().AppendEmpty()
	sample.SetStackIndex(0)
	sample.Values().Append(1)

	return pd, 0, 2
}

func TestProcessProfilesFillsNativeLines(t *testing.T) {
	pd, nativeLoc, namedLoc := buildProfiles()

	fr := &fakeResolver{names: map[uint64]string{0x1000: "soapcommand::TransactionListCommand::onResume()"}}
	p := newTestProcessor(fr)

	out, err := p.processProfiles(context.Background(), pd)
	require.NoError(t, err)

	dict := out.Dictionary()
	lt := dict.LocationTable()

	// The resolvable native location now has exactly one Line pointing at a
	// Function whose name is the resolved symbol.
	resolved := lt.At(nativeLoc)
	require.Equal(t, 1, resolved.Lines().Len(), "native location should get a Line")
	fnIdx := resolved.Lines().At(0).FunctionIndex()
	nameIdx := dict.FunctionTable().At(int(fnIdx)).NameStrindex()
	require.Equal(t, "soapcommand::TransactionListCommand::onResume()", dict.StringTable().At(int(nameIdx)))

	// The resolved native location is tagged with odigos.symbol.source so downstream
	// tooling can treat it as an instrumentable native symbol.
	require.Equal(t, 1, resolved.AttributeIndices().Len(), "native location should be tagged with its symbol source")
	attr := dict.AttributeTable().At(int(resolved.AttributeIndices().At(0)))
	require.Equal(t, "odigos.symbol.source", dict.StringTable().At(int(attr.KeyStrindex())))
	require.Equal(t, "symtab", attr.Value().Str())

	// The unresolvable native location stays empty.
	require.Equal(t, 0, lt.At(1).Lines().Len(), "unresolvable location must stay raw")

	// The already-symbolized location is untouched (still its original Line).
	require.Equal(t, 1, lt.At(namedLoc).Lines().Len())
	require.Equal(t, int32(0), lt.At(namedLoc).Lines().At(0).FunctionIndex())

	// Correct pid was used.
	require.Equal(t, int64(4242), fr.gotPID)
	require.Equal(t, "libfix.so", fr.gotName)
}

func TestProcessProfilesNoPIDSkips(t *testing.T) {
	pd, _, _ := buildProfiles()
	// Remove the pid attribute.
	pd.ResourceProfiles().At(0).Resource().Attributes().Clear()

	fr := &fakeResolver{names: map[uint64]string{0x1000: "x"}}
	p := newTestProcessor(fr)

	out, err := p.processProfiles(context.Background(), pd)
	require.NoError(t, err)
	require.Equal(t, 0, out.Dictionary().LocationTable().At(0).Lines().Len(), "no pid => no symbolization")
	require.Equal(t, int64(0), fr.gotPID, "resolver must not be called without a pid")
}

func TestPIDFromResourceString(t *testing.T) {
	pd := pprofile.NewProfiles()
	rp := pd.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutStr("process.pid", "1234")
	require.Equal(t, int64(1234), pidFromResource(rp.Resource().Attributes(), "process.pid"))
	rp.Resource().Attributes().PutStr("process.pid", "notanint")
	require.Equal(t, int64(0), pidFromResource(rp.Resource().Attributes(), "process.pid"))
}
