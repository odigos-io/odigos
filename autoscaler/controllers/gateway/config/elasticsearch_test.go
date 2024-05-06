package config_test

import (
	"testing"

	"github.com/odigos-io/odigos/autoscaler/controllers/gateway/config"
	"github.com/stretchr/testify/require"
)

func TestElasticsearch_SanitizeURL(t *testing.T) {
	tt := []struct {
		_           struct{}
		URL         string
		ExpectedURL string
		ExpectedErr string
	}{
		{
			URL:         "http://localhost:9200/",
			ExpectedURL: "http://localhost:9200/",
			ExpectedErr: "",
		},
		{
			URL:         "http://localhost",
			ExpectedURL: "http://localhost:9200",
			ExpectedErr: "",
		},
		{
			URL:         "http:///localhost",
			ExpectedURL: "",
			ExpectedErr: "invalid URL",
		},
		{
			URL:         "localhost",
			ExpectedURL: "",
			ExpectedErr: "invalid URI for request",
		},
		{
			URL:         "http://user:pass@localhost:9200",
			ExpectedURL: "http://user:pass@localhost:9200",
			ExpectedErr: "",
		},
		{
			URL:         "http://user:pass@localhost:80",
			ExpectedURL: "http://user:pass@localhost:80",
			ExpectedErr: "",
		},
		{
			URL:         "https://foobar.com:8443",
			ExpectedURL: "https://foobar.com:8443",
			ExpectedErr: "",
		},
		// IPs
		{
			URL:         "127.0.0.1:8080",
			ExpectedURL: "",
			ExpectedErr: "invalid URI for request",
		},
		{
			URL:         "http://127.0.0.1:8080",
			ExpectedURL: "http://127.0.0.1:8080",
			ExpectedErr: "",
		},
		{
			URL:         "[::1]:8080",
			ExpectedURL: "",
			ExpectedErr: "invalid URI for request",
		},
		{
			URL:         "http://[::1]:8080",
			ExpectedURL: "http://[::1]:8080",
			ExpectedErr: "",
		},
	}

	var es config.Elasticsearch
	for i := range tt {
		tc := tt[i]
		t.Run(tc.URL, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)

			actualURL, actualErr := es.SanitizeURL(tc.URL)
			if tc.ExpectedErr != "" {
				r.Error(actualErr)
				r.Contains(actualErr.Error(), tc.ExpectedErr)
			} else {
				r.NoError(actualErr)
				r.Equal(tc.ExpectedURL, actualURL)
			}

		})
	}
}
