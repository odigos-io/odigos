package odigospiimaskingprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/processor/processortest"

	"github.com/odigos-io/odigos/common/api/actions"
)

func TestMaskPiiData_CategoriesAndCustom(t *testing.T) {
	proc, err := newPiiMaskingProcessor(processortest.NewNopSettings(processortest.NopType), &Config{
		PiiMaskingConfig: actions.PiiMaskingConfig{
			PiiCategories: []actions.PiiCategory{actions.EmailMasking},
			CustomFormatMaskings: []actions.CustomFormatMasking{
				{LookupKey: "ssn", DataFormat: actions.FormatJSON},
			},
			CustomRegexMaskings: []actions.CustomRegexMasking{
				{Regex: `api[_-]?key=([^\s&]+)`},
			},
		},
	})
	require.NoError(t, err)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "email category",
			input: "contact me at user@example.com please",
			want:  "contact me at ***EMAIL*** please",
		},
		{
			name:  "json format masking",
			input: `{"ssn": "123-45-6789", "name": "alice"}`,
			want:  `{"ssn": "****", "name": "alice"}`,
		},
		{
			name:  "custom regex masking",
			input: "auth api_key=super-secret-value next",
			want:  "auth api_key=**** next",
		},
		{
			name:  "combined",
			input: `email=user@example.com payload={"ssn":"999"} api-key=abc123`,
			want:  `email=***EMAIL*** payload={"ssn":"****"} api-key=****`,
		},
		{
			name:  "no match",
			input: "nothing sensitive here",
			want:  "nothing sensitive here",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, changed := proc.maskPiiData(tc.input)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.input != tc.want, changed)
		})
	}
}

func TestBuildFormatMaskingRegex(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		format actions.DataFormat
		input  string
		want   string
	}{
		{
			name:   "json",
			key:    "user_id",
			format: actions.FormatJSON,
			input:  `{"user_id": "abc123", "name": "foo"}`,
			want:   `{"user_id": "****", "name": "foo"}`,
		},
		{
			name:   "sql",
			key:    "password",
			format: actions.FormatSQL,
			input:  `WHERE password = 'hunter2' AND status = 'ok'`,
			want:   `WHERE password = '****' AND status = 'ok'`,
		},
		{
			name:   "resource_path",
			key:    "orders",
			format: actions.FormatResourcePath,
			input:  `/api/v1/orders/abc-123/items`,
			want:   `/api/v1/orders/****/items`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			re, err := buildFormatMaskingRegex(tc.key, tc.format)
			require.NoError(t, err)
			got, changed := maskCaptureGroups(re, tc.input)
			assert.True(t, changed)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "valid",
			cfg: Config{PiiMaskingConfig: actions.PiiMaskingConfig{
				PiiCategories: []actions.PiiCategory{actions.EmailMasking},
				CustomFormatMaskings: []actions.CustomFormatMasking{
					{LookupKey: "ssn", DataFormat: actions.FormatJSON},
				},
				CustomRegexMaskings: []actions.CustomRegexMasking{
					{Regex: `(secret)`},
				},
			}},
		},
		{
			name: "invalid category",
			cfg: Config{PiiMaskingConfig: actions.PiiMaskingConfig{
				PiiCategories: []actions.PiiCategory{"PHONE"},
			}},
			wantErr: "unsupported category",
		},
		{
			name: "format missing lookupKey",
			cfg: Config{PiiMaskingConfig: actions.PiiMaskingConfig{
				CustomFormatMaskings: []actions.CustomFormatMasking{
					{DataFormat: actions.FormatJSON},
				},
			}},
			wantErr: "lookupKey is required",
		},
		{
			name: "regex without capture group",
			cfg: Config{PiiMaskingConfig: actions.PiiMaskingConfig{
				CustomRegexMaskings: []actions.CustomRegexMasking{
					{Regex: `abc`},
				},
			}},
			wantErr: "capture group",
		},
		{
			name: "invalid regex",
			cfg: Config{PiiMaskingConfig: actions.PiiMaskingConfig{
				CustomRegexMaskings: []actions.CustomRegexMasking{
					{Regex: `(`},
				},
			}},
			wantErr: "invalid regex",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
