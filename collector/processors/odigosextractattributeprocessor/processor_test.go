package odigosextractattributeprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// firstCapture returns the first captured group from m, or "" if there was no
// match. Encoded as a helper so table-driven tests can use the empty string to
// mean "expect no match".
func firstCapture(m []string) string {
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func TestBuildExtractionRegex_JSON(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		input     string
		extracted string // empty string means "expect no match"
	}{
		{
			name:      "json double-quoted",
			key:       "study_id",
			input:     `{"study_id": "abc-123"}`,
			extracted: "abc-123",
		},
		{
			name:      "json double-quoted, no space",
			key:       "study_id",
			input:     `{"study_id":"abc-123"}`,
			extracted: "abc-123",
		},
		{
			name:      "json with sibling keys",
			key:       "study_id",
			input:     `{"study_id": "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228","cooking_status": "Completed"}`,
			extracted: "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228",
		},
		{
			name:      "nested json object",
			key:       "study_id",
			input:     `{"outer":{"study_id":"x"}}`,
			extracted: "x",
		},
		{
			name:      "sql single-quoted",
			key:       "study_id",
			input:     `WHERE study_id = '1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228' RETURNING id`,
			extracted: "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228",
		},
		{
			name:      "sql single-quoted, tight",
			key:       "study_id",
			input:     `WHERE study_id='abc'`,
			extracted: "abc",
		},
		{
			name: "sql multi-line statement",
			key:  "study_id",
			input: "UPDATE orders\n      SET study_caching_status = 'Completed', study_location_code = 'cloud'\n" +
				"      WHERE study_id = '1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228'\n" +
				"      RETURNING id",
			extracted: "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228",
		},
		{
			name:      "unquoted value with equals",
			key:       "study_id",
			input:     `study_id=abc`,
			extracted: "abc",
		},
		{
			name:      "unquoted value with colon and space",
			key:       "study_id",
			input:     `study_id: abc`,
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
			name:      "underscore-joined key",
			key:       "study_id",
			input:     `my_study_id = 'nope'`,
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
			re, err := buildExtractionRegex(tc.key, FormatJSON)
			assert.NoError(t, err)
			got := firstCapture(re.FindStringSubmatch(tc.input))
			assert.Equal(t, tc.extracted, got, "json regex(key=%q) on %q", tc.key, tc.input)
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
			re, err := buildExtractionRegex(tc.key, FormatURL)
			assert.NoError(t, err)
			got := firstCapture(re.FindStringSubmatch(tc.input))
			assert.Equal(t, tc.extracted, got, "url regex(key=%q) on %q", tc.key, tc.input)
		})
	}
}
