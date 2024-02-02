package config_test

import (
	"context"
	"testing"

	"github.com/keyval-dev/odigos/autoscaler/controllers/gateway/config"
	"github.com/stretchr/testify/require"
)

func TestElasticsearch_SanitizeURL(t *testing.T) {
	tt := []struct {
		_           struct{}
		URL         string
		ExpectedURL string
	}{
		{
			URL:         "http://localhost:9200/",
			ExpectedURL: "http://localhost:9200/",
		},
		{
			URL:         "http://localhost",
			ExpectedURL: "http://localhost",
		},
		{
			URL:         "localhost",
			ExpectedURL: "localhost",
		},
		{
			URL:         "http://user:pass@localhost:9200",
			ExpectedURL: "http://user:pass@localhost:9200",
		},
	}

	var es config.Elasticsearch
	ctx := context.Background()

	for i := range tt {
		tc := tt[i]
		t.Run(tc.URL, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)

			actualURL, actualErr := es.SanitizeURL(ctx, tc.URL)
			r.NoError(actualErr)
			r.Equal(tc.ExpectedURL, actualURL)

		})
	}
}
