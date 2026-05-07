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
			name: "entry missing target",
			cfg: Config{Extractions: []Extraction{
				{Source: "study_id", DataFormat: FormatJSON},
			}},
			expectError: true,
			errContains: "target is required",
		},
		{
			name: "entry with neither source nor regex",
			cfg: Config{Extractions: []Extraction{
				{Target: "study.id"},
			}},
			expectError: true,
			errContains: "must set either source or regex",
		},
		{
			name: "entry with both source and regex",
			cfg: Config{Extractions: []Extraction{
				{Source: "study_id", Regex: `(.+)`, DataFormat: FormatJSON, Target: "study.id"},
			}},
			expectError: true,
			errContains: "cannot set both source and regex",
		},
		{
			name: "source without data_format",
			cfg: Config{Extractions: []Extraction{
				{Source: "study_id", Target: "study.id"},
			}},
			expectError: true,
			errContains: "data_format is required when source is set",
		},
		{
			name: "source with invalid data_format",
			cfg: Config{Extractions: []Extraction{
				{Source: "study_id", DataFormat: DataFormat("xml"), Target: "study.id"},
			}},
			expectError: true,
			errContains: "invalid data_format",
		},
		{
			name: "regex with stray data_format",
			cfg: Config{Extractions: []Extraction{
				{Regex: `(.+)`, DataFormat: FormatJSON, Target: "out"},
			}},
			expectError: true,
			errContains: "data_format must not be set when using regex",
		},
		{
			name: "duplicate target across entries",
			cfg: Config{Extractions: []Extraction{
				{Source: "study_id", DataFormat: FormatJSON, Target: "study.id"},
				{Source: "study_uuid", DataFormat: FormatJSON, Target: "study.id"},
			}},
			expectError: true,
			errContains: "duplicate target",
		},
		{
			name: "valid: all preset entries",
			cfg: Config{Extractions: []Extraction{
				{Source: "study_id", DataFormat: FormatJSON, Target: "study.id"},
				{Source: "studies", DataFormat: FormatURL, Target: "study.id.url"},
			}},
			expectError: false,
		},
		{
			name: "valid: all regex entries",
			cfg: Config{Extractions: []Extraction{
				{Regex: `study_id=([^\s]+)`, Target: "study.id"},
				{Regex: `request_id=([0-9a-f-]+)`, Target: "request.id"},
			}},
			expectError: false,
		},
		{
			name: "valid: mixed preset and regex entries",
			cfg: Config{Extractions: []Extraction{
				{Source: "study_id", DataFormat: FormatJSON, Target: "study.id"},
				{Regex: `request_id=([0-9a-f-]+)`, Target: "request.id"},
			}},
			expectError: false,
		},
		{
			name: "valid: same source key with different data_formats and targets",
			cfg: Config{Extractions: []Extraction{
				{Source: "study_id", DataFormat: FormatJSON, Target: "study.id.json"},
				{Source: "study_id", DataFormat: FormatURL, Target: "study.id.url"},
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
