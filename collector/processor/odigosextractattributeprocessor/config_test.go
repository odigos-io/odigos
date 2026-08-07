package odigosextractattributeprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		expectError bool
		errContains string
	}{
		{
			name:        "empty extractions list",
			cfg:         Config{},
			expectError: true,
			errContains: "extractions must not be empty",
		},
		{
			name: "entry missing target_attribute_name",
			cfg: Config{Extractions: []Extraction{
				{LookupKey: "study_id", DataFormat: FormatJSON},
			}},
			expectError: true,
			errContains: "target_attribute_name is required",
		},
		{
			name: "entry with neither lookup_key nor regex",
			cfg: Config{Extractions: []Extraction{
				{TargetAttributeName: "study.id"},
			}},
			expectError: true,
			errContains: "must set either lookup_key or regex",
		},
		{
			name: "entry with both lookup_key and regex",
			cfg: Config{Extractions: []Extraction{
				{LookupKey: "study_id", Regex: `(.+)`, DataFormat: FormatJSON, TargetAttributeName: "study.id"},
			}},
			expectError: true,
			errContains: "cannot set both lookup_key and regex",
		},
		{
			name: "lookup_key without data_format",
			cfg: Config{Extractions: []Extraction{
				{LookupKey: "study_id", TargetAttributeName: "study.id"},
			}},
			expectError: true,
			errContains: "data_format is required when lookup_key is set",
		},
		{
			name: "lookup_key with invalid data_format",
			cfg: Config{Extractions: []Extraction{
				{LookupKey: "study_id", DataFormat: DataFormat("xml"), TargetAttributeName: "study.id"},
			}},
			expectError: true,
			errContains: "invalid data_format",
		},
		{
			name: "regex with stray data_format",
			cfg: Config{Extractions: []Extraction{
				{Regex: `(.+)`, DataFormat: FormatJSON, TargetAttributeName: "out"},
			}},
			expectError: true,
			errContains: "data_format must not be set when using regex",
		},
		{
			name: "duplicate target_attribute_name across entries",
			cfg: Config{Extractions: []Extraction{
				{LookupKey: "study_id", DataFormat: FormatJSON, TargetAttributeName: "study.id"},
				{LookupKey: "study_uuid", DataFormat: FormatJSON, TargetAttributeName: "study.id"},
			}},
			expectError: true,
			errContains: "duplicate target_attribute_name",
		},
		{
			name: "valid: all preset entries",
			cfg: Config{Extractions: []Extraction{
				{LookupKey: "study_id", DataFormat: FormatJSON, TargetAttributeName: "study.id"},
				{LookupKey: "studies", DataFormat: FormatResourcePath, TargetAttributeName: "study.id.url"},
				{LookupKey: "study_id", DataFormat: FormatSQL, TargetAttributeName: "study.id.sql"},
			}},
			expectError: false,
		},
		{
			name: "valid: all regex entries",
			cfg: Config{Extractions: []Extraction{
				{Regex: `study_id=([^\s]+)`, TargetAttributeName: "study.id"},
				{Regex: `request_id=([0-9a-f-]+)`, TargetAttributeName: "request.id"},
			}},
			expectError: false,
		},
		{
			name: "valid: mixed preset and regex entries",
			cfg: Config{Extractions: []Extraction{
				{LookupKey: "study_id", DataFormat: FormatJSON, TargetAttributeName: "study.id"},
				{Regex: `request_id=([0-9a-f-]+)`, TargetAttributeName: "request.id"},
			}},
			expectError: false,
		},
		{
			name: "valid: same lookup_key with different data_formats and target_attribute_names",
			cfg: Config{Extractions: []Extraction{
				{LookupKey: "study_id", DataFormat: FormatJSON, TargetAttributeName: "study.id.json"},
				{LookupKey: "study_id", DataFormat: FormatResourcePath, TargetAttributeName: "study.id.url"},
			}},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.expectError {
				assert.Error(t, err)
				if err != nil && tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
