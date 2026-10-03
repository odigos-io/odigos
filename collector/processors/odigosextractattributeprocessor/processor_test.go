package odigosextractattributeprocessor

import (
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
)

// firstCapture returns the first captured group produced by regexes, trying them in
// the same order the processor does, or "" if none matched. Encoded as a helper so
// table-driven tests can use the empty string to mean "expect no match".
func firstCapture(regexes []*regexp.Regexp, input string) string {
	for _, re := range regexes {
		if m := re.FindStringSubmatch(input); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

func TestBuildExtractionRegex_JSON(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		input     string
		extracted string // empty string means "expect no match"
	}{
		{
			name:      "double-quoted",
			key:       "study_id",
			input:     `{"study_id": "abc-123"}`,
			extracted: "abc-123",
		},
		{
			name:      "double-quoted, no space",
			key:       "study_id",
			input:     `{"study_id":"abc-123"}`,
			extracted: "abc-123",
		},
		{
			name:      "with sibling keys",
			key:       "study_id",
			input:     `{"study_id": "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228","cooking_status": "Completed"}`,
			extracted: "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228",
		},
		{
			name:      "nested object",
			key:       "study_id",
			input:     `{"outer":{"study_id":"x"}}`,
			extracted: "x",
		},
		{
			name:      "quoted value containing spaces",
			key:       "full_name",
			input:     `{"full_name": "Jane Q Public", "id": 7}`,
			extracted: "Jane Q Public",
		},
		{
			name:      "quoted value containing a comma",
			key:       "full_name",
			input:     `{"full_name": "Public, Jane"}`,
			extracted: "Public, Jane",
		},
		{
			name:      "quoted value containing escaped quotes",
			key:       "note",
			input:     `{"note": "say \"hi\" now", "x": 1}`,
			extracted: `say \"hi\" now`,
		},
		{
			name:      "quoted timestamp value",
			key:       "created_at",
			input:     `{"created_at": "2026-09-15 10:03:49", "id": 7}`,
			extracted: "2026-09-15 10:03:49",
		},
		{
			name:      "quoted value is not truncated at a closing brace",
			key:       "address",
			input:     `{"address": "742 Evergreen Terrace}"}`,
			extracted: "742 Evergreen Terrace}",
		},
		{
			name:      "unquoted value with colon and space",
			key:       "study_id",
			input:     `study_id: abc`,
			extracted: "abc",
		},
		{
			name:      "numeric value",
			key:       "study_id",
			input:     `{"study_id": 123}`,
			extracted: "123",
		},
		{
			name:      "key at start of string",
			key:       "study_id",
			input:     `study_id: "xyz"`,
			extracted: "xyz",
		},

		// False positives -- these should NOT match.
		{
			name:      "substring key prefix",
			key:       "study_id",
			input:     `{"mystudy_id": "nope"}`,
			extracted: "",
		},
		{
			name:      "substring key suffix",
			key:       "study_id",
			input:     `{"study_id_v2": "nope"}`,
			extracted: "",
		},
		{
			name:      "unrelated key",
			key:       "study_id",
			input:     `{"other":"value"}`,
			extracted: "",
		},
		{
			name:      "empty content",
			key:       "study_id",
			input:     ``,
			extracted: "",
		},
		{
			name:      "sql equals does not match json",
			key:       "study_id",
			input:     `WHERE study_id = 'abc'`,
			extracted: "",
		},

		// Ensure regex metacharacters in the key are treated literally.
		{
			name:      "key with dot, literal match",
			key:       "a.b",
			input:     `{"a.b": "matched"}`,
			extracted: "matched",
		},
		{
			name:      "key with dot, metachar does not cross-match",
			key:       "a.b",
			input:     `{"aXb": "nope"}`,
			extracted: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			regexes, err := buildExtractionRegexes(tc.key, FormatJSON)
			assert.NoError(t, err)
			got := firstCapture(regexes, tc.input)
			assert.Equal(t, tc.extracted, got, "json regex(key=%q) on %q", tc.key, tc.input)
		})
	}
}

func TestBuildExtractionRegex_SQL(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		input     string
		extracted string // empty string means "expect no match"
	}{
		{
			name:      "single-quoted value",
			key:       "study_id",
			input:     `WHERE study_id = '1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228' RETURNING id`,
			extracted: "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228",
		},
		{
			name:      "single-quoted, tight",
			key:       "study_id",
			input:     `WHERE study_id='abc'`,
			extracted: "abc",
		},
		{
			name: "multi-line statement",
			key:  "study_id",
			input: "UPDATE orders\n      SET study_caching_status = 'Completed', study_location_code = 'cloud'\n" +
				"      WHERE study_id = '1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228'\n" +
				"      RETURNING id",
			extracted: "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228",
		},
		{
			name:      "unquoted value at start",
			key:       "study_id",
			input:     `study_id=abc`,
			extracted: "abc",
		},
		{
			name:      "unquoted value with spaced equals",
			key:       "study_id",
			input:     `study_id = abc-123`,
			extracted: "abc-123",
		},
		{
			name:      "numeric value",
			key:       "study_id",
			input:     `WHERE study_id=42`,
			extracted: "42",
		},
		{
			name:      "key inside parenthesized predicate",
			key:       "study_id",
			input:     `WHERE (study_id='abc') AND status='ok'`,
			extracted: "abc",
		},
		{
			name:      "value terminated by trailing semicolon",
			key:       "study_id",
			input:     `WHERE study_id=42;`,
			extracted: "42",
		},
		{
			name:      "quoted value containing spaces",
			key:       "full_name",
			input:     `UPDATE t SET x=1 WHERE full_name = 'Jane Q Public' RETURNING id`,
			extracted: "Jane Q Public",
		},
		{
			name:      "quoted value containing a comma",
			key:       "full_name",
			input:     `WHERE full_name = 'Public, Jane'`,
			extracted: "Public, Jane",
		},
		{
			name:      "quoted value containing a semicolon",
			key:       "note",
			input:     `WHERE note = 'first; second' AND x = 1`,
			extracted: "first; second",
		},

		// False positives -- these should NOT match.
		{
			name:      "underscore-joined key",
			key:       "study_id",
			input:     `my_study_id = 'nope'`,
			extracted: "",
		},
		{
			name:      "substring key suffix",
			key:       "study_id",
			input:     `WHERE study_id_v2 = 'nope'`,
			extracted: "",
		},
		{
			name:      "json colon does not match sql",
			key:       "study_id",
			input:     `{"study_id": "abc"}`,
			extracted: "",
		},
		{
			name:      "empty content",
			key:       "study_id",
			input:     ``,
			extracted: "",
		},

		// Ensure regex metacharacters in the key are treated literally.
		{
			name:      "key with dot, literal match",
			key:       "a.b",
			input:     `WHERE a.b = 'matched'`,
			extracted: "matched",
		},
		{
			name:      "key with dot, metachar does not cross-match",
			key:       "a.b",
			input:     `WHERE aXb = 'nope'`,
			extracted: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			regexes, err := buildExtractionRegexes(tc.key, FormatSQL)
			assert.NoError(t, err)
			got := firstCapture(regexes, tc.input)
			assert.Equal(t, tc.extracted, got, "sql regex(key=%q) on %q", tc.key, tc.input)
		})
	}
}

// TestProcessTraces_QuotedValueIsExtractedInFull drives the whole processor so the
// extracted span attribute, not just the regex, is covered.
func TestProcessTraces_QuotedValueIsExtractedInFull(t *testing.T) {
	tests := []struct {
		name          string
		dataFormat    DataFormat
		payloadAttr   string
		payload       string
		wantAttribute string
	}{
		{
			name:          "json payload",
			dataFormat:    FormatJSON,
			payloadAttr:   "http.request.payload",
			payload:       `{"full_name": "Jane Q Public", "id": 7}`,
			wantAttribute: "Jane Q Public",
		},
		{
			name:          "sql statement",
			dataFormat:    FormatSQL,
			payloadAttr:   "db.query.text",
			payload:       `SELECT id FROM users WHERE full_name = 'Jane Q Public'`,
			wantAttribute: "Jane Q Public",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{Extractions: []Extraction{{
				TargetAttributeName: "user.full_name",
				LookupKey:           "full_name",
				DataFormat:          tc.dataFormat,
			}}}

			proc, err := newExtractAttributeProcessor(processortest.NewNopSettings(component.MustNewType("odigosextractattribute")), cfg)
			require.NoError(t, err)

			traces := ptrace.NewTraces()
			span := traces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
			span.Attributes().PutStr(tc.payloadAttr, tc.payload)

			_, err = proc.processTraces(context.Background(), traces)
			require.NoError(t, err)

			got, ok := span.Attributes().Get("user.full_name")
			require.True(t, ok, "expected the target attribute to be set")
			assert.Equal(t, tc.wantAttribute, got.Str())
		})
	}
}

func TestBuildExtractionRegex_URL(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		input     string
		extracted string
	}{
		{
			name:      "basic segment in middle of path",
			key:       "studies",
			input:     `/studies/abc123/more`,
			extracted: "abc123",
		},
		{
			name:      "segment at end of string",
			key:       "studies",
			input:     `/studies/abc123`,
			extracted: "abc123",
		},
		{
			name:      "segment followed by trailing slash",
			key:       "studies",
			input:     `/studies/abc123/`,
			extracted: "abc123",
		},
		{
			name:      "value terminated by query string",
			key:       "studies",
			input:     `/studies/abc123?foo=bar`,
			extracted: "abc123",
		},
		{
			name:      "value terminated by fragment",
			key:       "studies",
			input:     `/studies/abc123#frag`,
			extracted: "abc123",
		},
		{
			name:      "dicom-style deep path",
			key:       "studies",
			input:     `projects/qaidg-workflowfg-90/locations/northamerica-northeast1/datasets/qaidg-workflowfg-90/dicomStores/dicom_workflowfac1_qa/dicomWeb/studies/1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228/series/1.2.276.0.28.3/instances/1.2`,
			extracted: "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228",
		},

		// False positives -- these should NOT match.
		{
			name:      "segment is a substring prefix",
			key:       "studies",
			input:     `/mystudies/abc`,
			extracted: "",
		},
		{
			name:      "segment is a substring suffix",
			key:       "studies",
			input:     `/studies_archive/abc`,
			extracted: "",
		},
		{
			name:      "no leading slash (start of string)",
			key:       "studies",
			input:     `studies/abc`,
			extracted: "abc",
		},
		{
			name:      "missing separator slash",
			key:       "studies",
			input:     `/studiesabc`,
			extracted: "",
		},
		{
			name:      "different segment name",
			key:       "studies",
			input:     `/series/abc`,
			extracted: "",
		},
		{
			name:      "empty value between slashes",
			key:       "studies",
			input:     `/studies/`,
			extracted: "",
		},

		// Ensure regex metacharacters in the key are treated literally.
		{
			name:      "key with dot, literal match",
			key:       "a.b",
			input:     `/a.b/matched`,
			extracted: "matched",
		},
		{
			name:      "key with dot, metachar does not cross-match",
			key:       "a.b",
			input:     `/aXb/nope`,
			extracted: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			regexes, err := buildExtractionRegexes(tc.key, FormatResourcePath)
			assert.NoError(t, err)
			got := firstCapture(regexes, tc.input)
			assert.Equal(t, tc.extracted, got, "url regex(key=%q) on %q", tc.key, tc.input)
		})
	}
}
