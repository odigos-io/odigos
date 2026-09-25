package services

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every level of the offsets file is a slice of pointers, so a null anywhere in
// it reaches the walker as a nil element. One null per level in a single fixture
// keeps a dropped guard from turning a malformed file into a panic in the UI.
func TestGoOffsetsToModel_skipsNilEntriesAtEveryLevel(t *testing.T) {
	file := `{"timestamp":"2026-07-29T00:00:00Z","mods":[
		null,
		{"module":"example.com/mod","packages":[
			null,
			{"package":"example.com/mod/pkg","structs":[
				null,
				{"struct":"Foo","fields":[
					null,
					{"field":"Bar","offsets":[null,{"offset":8,"versions":["1.0.0"]}]}
				]}
			]}
		]}
	]}`

	parsed := mustParseOffsetsFile(t, file)
	require.Len(t, parsed.Mods, 2)
	require.Nil(t, parsed.Mods[0], "fixture must actually contain a nil module")

	offsets := goOffsetsToModel(parsed)
	require.Len(t, offsets.Mods, 1)
	assert.Equal(t, "example.com/mod", offsets.Mods[0].Module)
	assert.Equal(t, "1.0.0", offsets.Mods[0].MinVersion)
	assert.Equal(t, "1.0.0", offsets.Mods[0].MaxVersion)
	require.Len(t, offsets.Mods[0].MinorVersions, 1)
	assert.Equal(t, []string{"1.0.0"}, offsets.Mods[0].MinorVersions[0].Versions)
}

// A version string the library cannot read is dropped rather than propagated:
// it has no place in the range, and listing it would offer the UI a version
// odiglet has no offsets for.
func TestGoOffsetsToModel_dropsUnparseableVersions(t *testing.T) {
	file := goffFileJSON("2026-07-29T00:00:00Z", goffMod{
		name:     "example.com/mod",
		versions: []string{"1.0.0", "not-a-version", "2.0.0", ""},
	})

	offsets := goOffsetsToModel(mustParseOffsetsFile(t, file))
	require.Len(t, offsets.Mods, 1)
	mod := offsets.Mods[0]
	assert.Equal(t, "1.0.0", mod.MinVersion)
	assert.Equal(t, "2.0.0", mod.MaxVersion)

	var listed []string
	for _, minor := range mod.MinorVersions {
		listed = append(listed, minor.Versions...)
	}
	assert.ElementsMatch(t, []string{"1.0.0", "2.0.0"}, listed)
}

// A module whose versions are all unreadable still has to render: an empty range
// and an empty — but non-null — list of minor versions.
func TestGoOffsetsToModel_moduleWithNoReadableVersions(t *testing.T) {
	file := goffFileJSON("2026-07-29T00:00:00Z", goffMod{
		name:     "example.com/mod",
		versions: []string{"not-a-version", "also-bad"},
	})

	offsets := goOffsetsToModel(mustParseOffsetsFile(t, file))
	require.Len(t, offsets.Mods, 1)
	assert.Equal(t, "", offsets.Mods[0].MinVersion)
	assert.Equal(t, "", offsets.Mods[0].MaxVersion)
	assert.NotNil(t, offsets.Mods[0].MinorVersions)
	assert.Empty(t, offsets.Mods[0].MinorVersions)
}

// Consecutive offsets overlap on the versions where a struct layout did not
// change, so the same version reaches the walker more than once and must be
// listed once.
func TestGoOffsetsToModel_dedupesVersionsSharedBetweenOffsets(t *testing.T) {
	file := `{"timestamp":"2026-07-29T00:00:00Z","mods":[{"module":"example.com/mod","packages":[{"package":"p","structs":[{"struct":"Foo","fields":[{"field":"Bar","offsets":[` +
		`{"offset":8,"versions":["1.1.0","1.1.1"]},` +
		`{"offset":16,"versions":["1.1.1","1.1.2"]}` +
		`]}]}]}]}]}`

	offsets := goOffsetsToModel(mustParseOffsetsFile(t, file))
	require.Len(t, offsets.Mods, 1)
	require.Len(t, offsets.Mods[0].MinorVersions, 1)
	assert.Equal(t, []string{"1.1.0", "1.1.1", "1.1.2"}, offsets.Mods[0].MinorVersions[0].Versions)
}

// An offsets file with no generation time renders as an empty string rather than
// Go's zero time, which the UI would display as a date in year 1.
func TestGoOffsetsToModel_absentTimestampRendersEmpty(t *testing.T) {
	file := goffFileJSON("", goffMod{name: "example.com/mod", versions: []string{"1.0.0"}})
	require.NotContains(t, file, "timestamp")

	offsets := goOffsetsToModel(mustParseOffsetsFile(t, file))
	assert.Equal(t, "", offsets.Timestamp)
	require.Len(t, offsets.Mods, 1)
}

// Module paths are listed case-insensitively; a byte-wise sort groups every
// capitalised path ahead of every lowercase one instead of alphabetising them.
func TestGoOffsetsToModel_sortsModulesCaseInsensitively(t *testing.T) {
	file := goffFileJSON("2026-07-29T00:00:00Z",
		goffMod{name: "Zebra.example.com/mod", versions: []string{"1.0.0"}},
		goffMod{name: "apple.example.com/mod", versions: []string{"1.0.0"}},
		goffMod{name: "Banana.example.com/mod", versions: []string{"1.0.0"}},
	)

	offsets := goOffsetsToModel(mustParseOffsetsFile(t, file))
	got := make([]string, 0, len(offsets.Mods))
	for _, mod := range offsets.Mods {
		got = append(got, mod.Module)
	}
	assert.Equal(t, []string{"apple.example.com/mod", "Banana.example.com/mod", "Zebra.example.com/mod"}, got)
}

// Neither the module order in the file nor the version order inside an offset is
// meaningful, and the table must not reshuffle between two reads of equivalent
// files.
func TestGoOffsetsToModel_outputIsIndependentOfInputOrder(t *testing.T) {
	forward := goffFileJSON("2026-07-29T00:00:00Z",
		goffMod{name: "example.com/a", versions: []string{"1.0.0", "1.1.0", "2.0.0"}},
		goffMod{name: "example.com/b", versions: []string{"1.2.0", "1.10.0"}},
	)
	reversed := goffFileJSON("2026-07-29T00:00:00Z",
		goffMod{name: "example.com/b", versions: []string{"1.10.0", "1.2.0"}},
		goffMod{name: "example.com/a", versions: []string{"2.0.0", "1.1.0", "1.0.0"}},
	)
	require.NotEqual(t, forward, reversed)

	assert.Equal(t, goOffsetsToModel(mustParseOffsetsFile(t, forward)), goOffsetsToModel(mustParseOffsetsFile(t, reversed)))
}

// The ConfigMap holds the offsets file inside a JSON string while the offsets URL
// serves it bare, so the two parsers are not interchangeable. Calling the wrong
// one turns a healthy cluster into "invalid go offsets JSON".
func TestParseGoOffsets_contentAndFileDifferByJSONStringWrapping(t *testing.T) {
	bare := goffFileJSON("2026-07-29T00:00:00Z", goffMod{name: "example.com/mod", versions: []string{"1.0.0"}})
	wrapped := goffPayload(t, bare)

	t.Run("file accepts the bare document", func(t *testing.T) {
		parsed, err := parseGoOffsetsFile(bare)
		require.NoError(t, err)
		assert.Len(t, parsed.Mods, 1)
	})

	t.Run("content accepts the wrapped document", func(t *testing.T) {
		parsed, err := parseGoOffsetsContent(wrapped)
		require.NoError(t, err)
		assert.Len(t, parsed.Mods, 1)
	})

	t.Run("content rejects the bare document", func(t *testing.T) {
		_, err := parseGoOffsetsContent(bare)
		assert.ErrorContains(t, err, "invalid go offsets JSON")
	})

	t.Run("file rejects the wrapped document", func(t *testing.T) {
		_, err := parseGoOffsetsFile(wrapped)
		assert.ErrorContains(t, err, "invalid go offsets JSON")
	})
}

// mods is a non-null GraphQL list, so a document that simply omits the key has
// to normalise to an empty slice: nil marshals as null and fails the query.
func TestParseGoOffsetsFile_absentModsKeyBecomesAnEmptyList(t *testing.T) {
	parsed, err := parseGoOffsetsFile(`{"timestamp":"2026-07-29T00:00:00Z"}`)
	require.NoError(t, err)
	assert.NotNil(t, parsed.Mods)
	assert.Empty(t, parsed.Mods)

	offsets := goOffsetsToModel(parsed)
	assert.NotNil(t, offsets.Mods)
	assert.Empty(t, offsets.Mods)
}

// Everything after the delimiter is a detached signature, not offsets data.
func TestStripGoOffsetsSignature(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "no signature", in: `{"mods":[]}`, want: `{"mods":[]}`},
		{name: "signature trailer", in: "{\"mods\":[]}\n---SIGNATURE---\nZGVhZA==\n", want: `{"mods":[]}`},
		{name: "signature only", in: "---SIGNATURE---ZGVhZA==", want: ""},
		{
			// A signature block that itself contains the delimiter must not
			// resurrect the tail as offsets data.
			name: "delimiter appears twice",
			in:   `{"mods":[]}---SIGNATURE---a---SIGNATURE---b`,
			want: `{"mods":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, stripGoOffsetsSignature(tt.in))
		})
	}
}

// Version ordering is numeric, and the lexical fallback only applies to strings
// the library cannot read. The two unreadable rows order the other way round
// from the readable ones, so a fallback that fires too eagerly is visible.
func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{name: "numeric ordering beats lexical", a: "1.10.0", b: "1.2.0", want: false},
		{name: "numeric ordering ascending", a: "1.2.0", b: "1.10.0", want: true},
		{name: "equal versions are not less", a: "1.2.0", b: "1.2.0", want: false},
		{name: "normalised forms compare equal", a: "1.2", b: "1.2.0", want: false},
		{name: "both unreadable falls back to lexical", a: "bogus-1.10", b: "bogus-1.2", want: true},
		{name: "left unreadable falls back to lexical", a: "bogus", b: "1.0.0", want: false},
		{name: "right unreadable falls back to lexical", a: "1.0.0", b: "bogus", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, compareVersions(tt.a, tt.b))
		})
	}
}

// Inside one minor the diff lists patch versions ascending and numerically: a
// lexical sort files 1.1.10 between 1.1.0 and 1.1.2.
func TestCompareGoOffsets_ordersVersionsWithinAMinorNumerically(t *testing.T) {
	current := mustParseOffsetsFile(t, goffFileJSON("2026-07-01T00:00:00Z",
		goffMod{name: "example.com/mod", versions: []string{"1.1.2"}}))
	proposed := mustParseOffsetsFile(t, goffFileJSON("2026-07-30T00:00:00Z",
		goffMod{name: "example.com/mod", versions: []string{"1.1.10", "1.1.2", "1.1.0"}}))

	result := compareGoOffsets(current, proposed)
	require.Len(t, result.Mods, 1)
	require.Len(t, result.Mods[0].MinorVersions, 1)

	got := make([]string, 0, 3)
	isNew := make(map[string]bool, 3)
	for _, ver := range result.Mods[0].MinorVersions[0].Versions {
		got = append(got, ver.Version)
		isNew[ver.Version] = ver.IsNew
	}
	assert.Equal(t, []string{"1.1.0", "1.1.2", "1.1.10"}, got)
	// The one version already installed is the only one not flagged as new, so
	// the ordering assertion cannot be satisfied by a list that lost an entry.
	assert.Equal(t, map[string]bool{"1.1.0": true, "1.1.2": false, "1.1.10": true}, isNew)
	// A minor that gains versions but keeps one is not itself new.
	assert.False(t, result.Mods[0].MinorVersions[0].IsNew)
}

// Narrowing a module the candidate still lists is an update in its own right.
// Nothing is added and no module disappears here, so the dropped version is the
// only thing that can raise the flag the UI gates its warning on.
func TestCompareGoOffsets_droppingVersionsFromAKeptModuleIsAnUpdate(t *testing.T) {
	current := mustParseOffsetsFile(t, goffFileJSON("2026-07-01T00:00:00Z",
		goffMod{name: "example.com/mod", versions: []string{"1.0.0", "1.1.0"}}))
	proposed := mustParseOffsetsFile(t, goffFileJSON("2026-07-30T00:00:00Z",
		goffMod{name: "example.com/mod", versions: []string{"1.0.0"}}))

	result := compareGoOffsets(current, proposed)
	require.Len(t, result.Mods, 1)
	assert.True(t, result.HasUpdates)
	assert.False(t, result.Mods[0].IsNew)
	assert.False(t, result.Mods[0].IsRemoved)
}

// A module with no path cannot be matched between the two sides, so it is left
// out of the diff entirely rather than added as a nameless row.
func TestCompareGoOffsets_skipsUnnamedModules(t *testing.T) {
	current := mustParseOffsetsFile(t, goffFileJSON("2026-07-01T00:00:00Z",
		goffMod{name: "", versions: []string{"1.0.0"}},
		goffMod{name: "example.com/mod", versions: []string{"1.0.0"}},
	))
	proposed := mustParseOffsetsFile(t, goffFileJSON("2026-07-30T00:00:00Z",
		goffMod{name: "", versions: []string{"9.0.0"}},
		goffMod{name: "example.com/mod", versions: []string{"1.0.0"}},
	))
	require.Len(t, current.Mods, 2, "fixture must actually contain an unnamed module")

	result := compareGoOffsets(current, proposed)
	require.Len(t, result.Mods, 1)
	assert.Equal(t, "example.com/mod", result.Mods[0].Module)
	assert.False(t, result.HasUpdates)
}

// Every list in the diff is non-null in the schema, including on a comparison
// that finds nothing at all to report.
func TestCompareGoOffsets_listsAreNonNilWhenThereIsNothingToReport(t *testing.T) {
	empty := mustParseOffsetsFile(t, `{"mods":[]}`)

	result := compareGoOffsets(empty, empty)
	assert.False(t, result.HasUpdates)
	assert.NotNil(t, result.Mods)
	assert.Empty(t, result.Mods)
	assert.Equal(t, "", result.CurrentTimestamp)
	assert.Equal(t, "", result.ProposedTimestamp)

	single := mustParseOffsetsFile(t, goffFileJSON("2026-07-01T00:00:00Z",
		goffMod{name: "example.com/mod", versions: []string{"1.0.0"}}))
	result = compareGoOffsets(single, single)
	require.Len(t, result.Mods, 1)
	require.Len(t, result.Mods[0].MinorVersions, 1)
	assert.NotNil(t, result.Mods[0].MinorVersions[0].Versions)
}

// The ConfigMap payload is a JSON string of the whole file, so anything the file
// may legally contain — quotes, newlines, a signature trailer — has to survive
// the encode/decode pair the writer and reader form.
func TestEncodeGoOffsets_roundTripsThroughTheContentParser(t *testing.T) {
	file := goffFileJSON("2026-07-29T00:00:00Z",
		goffMod{name: "example.com/mod", versions: []string{"1.0.0"}}) + "\n---SIGNATURE---\n\"quoted\"\tand\ttabbed\n"

	encoded, err := encodeGoOffsets([]byte(file))
	require.NoError(t, err)

	var decoded string
	require.NoError(t, json.Unmarshal([]byte(encoded), &decoded))
	assert.Equal(t, file, decoded)

	parsed, err := parseGoOffsetsContent(encoded)
	require.NoError(t, err)
	require.Len(t, parsed.Mods, 1)
	assert.Equal(t, "example.com/mod", parsed.Mods[0].Module)
}
