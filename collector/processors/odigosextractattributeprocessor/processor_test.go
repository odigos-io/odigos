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

func TestBuildExtractionRegexes_JSON(t *testing.T) {
	tests := []struct {
		name             string
		key              string
		input            string
		extracted_string string // empty string means "expect no match"
	}{
		{
			name:             "json double-quoted",
			key:              "study_id",
			input:            `{"study_id": "abc-123"}`,
			extracted_string: "abc-123",
		},
		{
			name:             "json double-quoted, no space",
			key:              "study_id",
			input:            `{"study_id":"abc-123"}`,
			extracted_string: "abc-123",
		},
		{
			name:             "json with sibling keys",
			key:              "study_id",
			input:            `{"study_id": "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228","cooking_status": "Completed"}`,
			extracted_string: "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228",
		},
		{
			name:             "nested json object",
			key:              "study_id",
			input:            `{"outer":{"study_id":"x"}}`,
			extracted_string: "x",
		},
		{
			name:             "sql single-quoted",
			key:              "study_id",
			input:            `WHERE study_id = '1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228' RETURNING id`,
			extracted_string: "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228",
		},
		{
			name:             "sql single-quoted, tight",
			key:              "study_id",
			input:            `WHERE study_id='abc'`,
			extracted_string: "abc",
		},
		{
			name: "sql multi-line statement",
			key:  "study_id",
			input: "UPDATE orders\n      SET study_caching_status = 'Completed', study_location_code = 'cloud'\n" +
				"      WHERE study_id = '1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228'\n" +
				"      RETURNING id",
			extracted_string: "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228",
		},
		{
			name:             "unquoted value with equals",
			key:              "study_id",
			input:            `study_id=abc`,
			extracted_string: "abc",
		},
		{
			name:             "unquoted value with colon and space",
			key:              "study_id",
			input:            `study_id: abc`,
			extracted_string: "abc",
		},
		{
			name:             "unquoted value with spaced equals",
			key:              "study_id",
			input:            `study_id = abc-123`,
			extracted_string: "abc-123",
		},
		{
			name:             "numeric value",
			key:              "study_id",
			input:            `{"study_id": 123}`,
			extracted_string: "123",
		},
		{
			name:             "key at start of string",
			key:              "study_id",
			input:            `study_id: "xyz"`,
			extracted_string: "xyz",
		},

		// False positives -- these should NOT match.
		{
			name:             "substring key prefix",
			key:              "study_id",
			input:            `{"mystudy_id": "nope"}`,
			extracted_string: "",
		},
		{
			name:             "substring key suffix",
			key:              "study_id",
			input:            `{"study_id_v2": "nope"}`,
			extracted_string: "",
		},
		{
			name:             "underscore-joined key",
			key:              "study_id",
			input:            `my_study_id = 'nope'`,
			extracted_string: "",
		},
		{
			name:             "unrelated key",
			key:              "study_id",
			input:            `{"other":"value"}`,
			extracted_string: "",
		},
		{
			name:             "empty content",
			key:              "study_id",
			input:            ``,
			extracted_string: "",
		},

		// Ensure regex metacharacters in the key are treated literally.
		{
			name:             "key with dot, literal match",
			key:              "a.b",
			input:            `{"a.b": "matched"}`,
			extracted_string: "matched",
		},
		{
			name:             "key with dot, metachar does not cross-match",
			key:              "a.b",
			input:            `{"aXb": "nope"}`,
			extracted_string: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jsonRe, _ := buildExtractionRegexes(tc.key)
			got := firstCapture(jsonRe.FindStringSubmatch(tc.input))

			assert.Equal(t, tc.extracted_string, got, "jsonRe(key=%q) on %q", tc.key, tc.input)
		})
	}
}

func TestBuildExtractionRegexes_URL(t *testing.T) {
	tests := []struct {
		name             string
		key              string
		input            string
		extracted_string string
	}{
		{
			name:             "basic segment in middle of path",
			key:              "studies",
			input:            `/studies/abc123/more`,
			extracted_string: "abc123",
		},
		{
			name:             "segment at end of string",
			key:              "studies",
			input:            `/studies/abc123`,
			extracted_string: "abc123",
		},
		{
			name:             "segment followed by trailing slash",
			key:              "studies",
			input:            `/studies/abc123/`,
			extracted_string: "abc123",
		},
		{
			name:             "value terminated by query string",
			key:              "studies",
			input:            `/studies/abc123?foo=bar`,
			extracted_string: "abc123",
		},
		{
			name:             "value terminated by fragment",
			key:              "studies",
			input:            `/studies/abc123#frag`,
			extracted_string: "abc123",
		},
		{
			name:             "dicom-style deep path",
			key:              "studies",
			input:            `projects/qaidg-workflowfg-90/locations/northamerica-northeast1/datasets/qaidg-workflowfg-90/dicomStores/dicom_workflowfac1_qa/dicomWeb/studies/1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228/series/1.2.276.0.28.3/instances/1.2`,
			extracted_string: "1.3.6.1.4.1.40744.71.65797265067703624152858272792653363228",
		},

		// False positives -- these should NOT match.
		{
			name:             "segment is a substring prefix",
			key:              "studies",
			input:            `/mystudies/abc`,
			extracted_string: "",
		},
		{
			name:             "segment is a substring suffix",
			key:              "studies",
			input:            `/studies_archive/abc`,
			extracted_string: "",
		},
		{
			name:             "missing leading slash",
			key:              "studies",
			input:            `studies/abc`,
			extracted_string: "",
		},
		{
			name:             "missing separator slash",
			key:              "studies",
			input:            `/studiesabc`,
			extracted_string: "",
		},
		{
			name:             "different segment name",
			key:              "studies",
			input:            `/series/abc`,
			extracted_string: "",
		},
		{
			name:             "empty value between slashes",
			key:              "studies",
			input:            `/studies/`,
			extracted_string: "",
		},

		// Ensure regex metacharacters in the key are treated literally.
		{
			name:             "key with dot, literal match",
			key:              "a.b",
			input:            `/a.b/matched`,
			extracted_string: "matched",
		},
		{
			name:             "key with dot, metachar does not cross-match",
			key:              "a.b",
			input:            `/aXb/nope`,
			extracted_string: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, urlRe := buildExtractionRegexes(tc.key)
			got := firstCapture(urlRe.FindStringSubmatch(tc.input))

			assert.Equal(t, tc.extracted_string, got, "urlRe(key=%q) on %q", tc.key, tc.input)
		})
	}
}
