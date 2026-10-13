// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package awss3exporter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.uber.org/zap"
)

func TestStaticCredentialsCannotFallBackToAWS(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.S3Uploader.S3Bucket = "archive"
	cfg.S3Uploader.UseStaticCredentials = true
	require.Error(t, cfg.Validate())
	cfg.S3Uploader.AccessKeyID = "swift-access"
	require.Error(t, cfg.Validate())
	cfg.S3Uploader.SecretAccessKey = "swift-secret"
	require.NoError(t, cfg.Validate())
}

func TestS3ClientsUseTheirOwnCredentials(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "ambient-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "ambient-secret")
	for _, access := range []string{"swift-first", "swift-second"} {
		t.Run(access, func(t *testing.T) {
			var authorization string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authorization = r.Header.Get("Authorization")
				w.Header().Set("ETag", "\"test-etag\"")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			cfg := createDefaultConfig().(*Config)
			cfg.S3Uploader.S3Bucket = "archive"
			cfg.S3Uploader.Endpoint = server.URL
			cfg.S3Uploader.S3ForcePathStyle = true
			cfg.S3Uploader.UseStaticCredentials = true
			cfg.S3Uploader.AccessKeyID = configopaque.String(access)
			cfg.S3Uploader.SecretAccessKey = "private-secret"
			manager, err := newUploadManager(t.Context(), cfg, zap.NewNop(), "logs", "json", false)
			require.NoError(t, err)
			require.NoError(t, manager.Upload(t.Context(), []byte("{}"), nil))
			require.True(t, strings.Contains(authorization, "Credential="+access+"/"))
			require.NotContains(t, authorization, "ambient-access")
		})
	}
}
