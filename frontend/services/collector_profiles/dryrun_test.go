package collectorprofiles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/frontend/services/collector_profiles/flamegraph"
)

func TestDryRunFlamegraphsFromCommittedTestdata(t *testing.T) {
	t.Parallel()
	cases := []struct {
		file       string
		minTicks   int64
		wantSymbol bool
	}{
		{"accounting-merged.json", 1, false},
		{"chunk-with-symbols.json", 1, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.file, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join("testdata", tc.file))
			if err != nil {
				t.Fatalf("read testdata: %v", err)
			}
			chunks := [][]byte{data}
			prof, dbg := BuildPyroscopeProfileFromChunksWithDebug(chunks)
			if dbg.ChunkCount != 1 {
				t.Errorf("chunkCount: got %d", dbg.ChunkCount)
			}
			if dbg.NumTicks < tc.minTicks {
				t.Errorf("debug.NumTicks: got %d want >= %d", dbg.NumTicks, tc.minTicks)
			}
			fb := prof.Flamebearer
			if fb.NumTicks < tc.minTicks {
				t.Errorf("flamebearer.NumTicks: got %d want >= %d", fb.NumTicks, tc.minTicks)
			}
			if len(fb.Names) == 0 || fb.Names[0] != "total" {
				t.Errorf("names[0]: got %v", fb.Names)
			}
			if len(fb.Levels) == 0 {
				t.Fatal("levels empty")
			}
			hasReal := false
			for _, n := range fb.Names {
				if n != "" && n != "total" && n != "other" && !strings.HasPrefix(n, "frame_") && !strings.HasPrefix(n, "0x") {
					hasReal = true
					break
				}
			}
			if tc.wantSymbol && !hasReal {
				t.Errorf("expected at least one non-placeholder name, got: %v", fb.Names)
			}
			if prof.Version != 1 {
				t.Errorf("version: %d", prof.Version)
			}
			if prof.Metadata.Units != "samples" {
				t.Errorf("metadata.units: %q", prof.Metadata.Units)
			}
			raw, err := json.Marshal(prof)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var m map[string]interface{}
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatalf("roundtrip: %v", err)
			}
			for _, k := range []string{"version", "flamebearer", "metadata", "timeline"} {
				if _, ok := m[k]; !ok {
					t.Errorf("missing top-level key %q", k)
				}
			}
			t.Logf("ticks=%d names=%d levels=%d pyroChunks=%d parseErr=%d chunksWithSamples=%d",
				fb.NumTicks, len(fb.Names), len(fb.Levels), dbg.ChunksViaPyroscope, dbg.ParseErrors, dbg.ChunksWithSamples)
		})
	}
}

func TestDryRunMultiChunkSameAsMergedDump(t *testing.T) {
	t.Parallel()
	dir := os.Getenv("PROFILE_DRYRUN_EXTRA_DIR")
	if dir == "" {
		t.Skip("PROFILE_DRYRUN_EXTRA_DIR not set")
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	var chunks [][]byte
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		chunks = append(chunks, b)
	}
	if len(chunks) == 0 {
		t.Skip("no json chunks")
		return
	}
	prof, dbg := BuildPyroscopeProfileFromChunksWithDebug(chunks)
	if prof.Flamebearer.NumTicks == 0 {
		t.Errorf("numTicks=0 dbg=%+v", dbg)
	}
	t.Logf("chunks=%d ticks=%d %+v", len(chunks), prof.Flamebearer.NumTicks, dbg)
}

func TestFlamebearerLevelsAreQuadruples(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("testdata", "accounting-merged.json"))
	if err != nil {
		t.Fatal(err)
	}
	prof := BuildPyroscopeProfileFromChunks([][]byte{data})
	fb := prof.Flamebearer
	for i, row := range fb.Levels {
		if len(row)%4 != 0 {
			t.Errorf("level %d len %d not multiple of 4", i, len(row))
		}
		for j := 0; j+3 < len(row); j += 4 {
			idx := row[j+3]
			if idx < 0 || int(idx) >= len(fb.Names) {
				t.Errorf("level %d node %d nameIndex %d out of range names=%d", i, j/4, idx, len(fb.Names))
			}
		}
	}
}

func TestPyroscopeJSONShapeHasHeatmapAndGroups(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("testdata", "chunk-with-symbols.json"))
	if err != nil {
		t.Fatal(err)
	}
	prof := BuildPyroscopeProfileFromChunks([][]byte{data})
	raw, err := json.Marshal(prof)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"groups", "heatmap"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing %q", k)
		}
	}
}

func TestBuildSummaryEnvDoesNotPanic(t *testing.T) {
	t.Setenv("PROFILE_BUILD_SUMMARY", "1")
	data, err := os.ReadFile(filepath.Join("testdata", "chunk-with-symbols.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = BuildPyroscopeProfileFromChunksWithDebug([][]byte{data})
}

func TestSymbolNamesFromDictionaryAppearInFlamebearer(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("testdata", "chunk-with-symbols.json"))
	if err != nil {
		t.Fatal(err)
	}
	wantSub := []string{"runtime.main", "main.foo", "main.bar"}
	prof := BuildPyroscopeProfileFromChunks([][]byte{data})
	namesJoined := strings.Join(prof.Flamebearer.Names, "\x00")
	for _, s := range wantSub {
		if !strings.Contains(namesJoined, s) {
			t.Errorf("expected name %q in flamebearer.names: %v", s, prof.Flamebearer.Names)
		}
	}
}

func TestTreeInsertMatchesFlamebearerTicks(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("testdata", "accounting-merged.json"))
	if err != nil {
		t.Fatal(err)
	}
	prof, dbg := BuildPyroscopeProfileFromChunksWithDebug([][]byte{data})
	tr := flamegraph.NewTree()
	chunks := [][]byte{data}
	for _, b := range chunks {
		samples, _ := flamegraph.SamplesFromOTLPChunk(b)
		for _, s := range samples {
			tr.InsertStack(s.Value, s.Stack...)
		}
	}
	fb2 := flamegraph.TreeToFlamebearer(tr, 1024)
	if fb2.NumTicks != prof.Flamebearer.NumTicks {
		t.Errorf("ticks mismatch tree=%d profile=%d dbg=%+v", fb2.NumTicks, prof.Flamebearer.NumTicks, dbg)
	}
}
