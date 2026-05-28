package actions

import (
	"testing"

	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/stretchr/testify/require"
)

func TestExtractAttributeConfigRejectsInvalidRegex(t *testing.T) {
	_, err := extractAttributeConfig(&odigosactions.ExtractAttributeConfig{
		Extractions: []odigosactions.Extraction{
			{
				TargetAttributeName: "request.id",
				Regex:               "[invalid",
			},
		},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid regex")
}

func TestExtractAttributeConfigAllowsValidRegex(t *testing.T) {
	cfg, err := extractAttributeConfig(&odigosactions.ExtractAttributeConfig{
		Extractions: []odigosactions.Extraction{
			{
				TargetAttributeName: "request.id",
				Regex:               `request_id=([A-Za-z0-9-]+)`,
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, cfg.Extractions, 1)
	require.Equal(t, `request_id=([A-Za-z0-9-]+)`, cfg.Extractions[0].Regex)
}
