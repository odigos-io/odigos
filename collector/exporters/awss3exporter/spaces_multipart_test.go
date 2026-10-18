// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package awss3exporter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.uber.org/zap"
)

func TestSpacesMultipartAndRejectedUpload(t *testing.T) {
	var mu sync.Mutex
	parts := map[int][]byte{}
	var downloaded []byte
	var uploads, completions, rejections int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path == "/telemetry/rejected" {
			rejections++
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`<Error><Code>AccessDenied</Code></Error>`))
			return
		}
		assert.Contains(t, r.Header.Get("Authorization"), "Credential=multipart-key/")
		assert.Empty(t, r.Header.Get("X-Amz-Sdk-Checksum-Algorithm"))
		q := r.URL.Query()
		switch {
		case r.Method == "POST" && q.Has("uploads"):
			uploads++
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<InitiateMultipartUploadResult><Bucket>telemetry</Bucket><Key>large</Key><UploadId>test-upload</UploadId></InitiateMultipartUploadResult>`))
		case r.Method == "PUT" && q.Has("partNumber"):
			number, err := strconv.Atoi(q.Get("partNumber"))
			assert.NoError(t, err)
			payload, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			parts[number] = payload
			w.Header().Set("ETag", fmt.Sprintf(`"part-%d"`, number))
		case r.Method == "POST" && q.Has("uploadId"):
			completions++
			numbers := []int{}
			for n := range parts {
				numbers = append(numbers, n)
			}
			sort.Ints(numbers)
			for _, n := range numbers {
				downloaded = append(downloaded, parts[n]...)
			}
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<CompleteMultipartUploadResult><Bucket>telemetry</Bucket><Key>large</Key><ETag>"complete"</ETag></CompleteMultipartUploadResult>`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
	}))
	defer server.Close()
	cfg := createDefaultConfig().(*Config)
	cfg.S3Uploader.Endpoint = server.URL
	cfg.S3Uploader.S3Bucket = "telemetry"
	cfg.S3Uploader.S3ForcePathStyle = true
	cfg.S3Uploader.DisableSSL = true
	cfg.S3Uploader.AccessKeyID = configopaque.String("multipart-key")
	cfg.S3Uploader.SecretAccessKey = configopaque.String("multipart-secret")
	cfg.S3Uploader.ChecksumWhenRequired = true
	manager, err := newUploadManager(context.Background(), cfg, zap.NewNop(), "logs", "json", false)
	require.NoError(t, err)
	payload := bytes.Repeat([]byte("telemetry"), 2*1024*1024)
	require.NoError(t, manager.Upload(context.Background(), payload, nil))
	require.Equal(t, 1, uploads)
	require.Equal(t, 1, completions)
	require.Greater(t, len(parts), 1)
	require.Equal(t, payload, downloaded)

	denied := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rejections++
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`<Error><Code>AccessDenied</Code></Error>`))
	}))
	defer denied.Close()
	cfg.S3Uploader.Endpoint = denied.URL
	manager, err = newUploadManager(context.Background(), cfg, zap.NewNop(), "logs", "json", false)
	require.NoError(t, err)
	err = manager.Upload(context.Background(), []byte("denied"), nil)
	require.ErrorContains(t, err, "AccessDenied")
	require.Equal(t, 1, rejections)
}
