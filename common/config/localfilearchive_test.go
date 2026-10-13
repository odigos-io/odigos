package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/common"
)

func TestLocalFileArchiveConfig(t *testing.T) {
	configers, err := LoadConfigers()
	require.NoError(t, err)
	require.Contains(t, configers, common.LocalFileArchiveDestinationType)
	for _, compression := range []string{"none", "zstd"} {
		for _, format := range []string{"json", "proto"} {
			t.Run(format+"/"+compression, func(t *testing.T) {
				dest := &mockDestination{id: "archive.example", config: map[string]string{LocalFileArchivePathKey: "/archive", fileArchiveFormatKey: format, fileArchiveCompressionKey: compression}}
				cfg := Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
				pipelines, err := configers[common.LocalFileArchiveDestinationType].ModifyConfig(dest, &cfg)
				require.NoError(t, err)
				require.Len(t, pipelines, 3)
				paths := map[string]bool{}
				for _, pipeline := range pipelines {
					require.Len(t, cfg.Service.Pipelines[pipeline].Exporters, 1)
					exporter := cfg.Exporters[cfg.Service.Pipelines[pipeline].Exporters[0]].(GenericMap)
					path := exporter["path"].(string)
					assert.Equal(t, filepath.Join("/archive", dest.id), filepath.Dir(path))
					assert.False(t, paths[path])
					paths[path] = true
					assert.Equal(t, format, exporter["format"])
					assert.Equal(t, GenericMap{"max_megabytes": 100, "max_backups": 10, "max_days": 7}, exporter["rotation"])
					assert.NotContains(t, exporter, "append")
				}
			})
		}
	}
}

func TestLocalFileArchiveAppendAndPodIsolation(t *testing.T) {
	dest := &mockDestination{id: "archive", config: map[string]string{LocalFileArchivePathKey: "/archive", LocalFileArchivePVCKey: "archive-pvc", fileArchiveMaxMegabytesKey: "0"}, signals: []common.ObservabilitySignal{common.LogsObservabilitySignal}}
	cfg := Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
	pipelines, err := (&LocalFileArchive{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)
	require.Len(t, pipelines, 1)
	exporter := cfg.Exporters["file/archive-logs"].(GenericMap)
	assert.Equal(t, "/archive/archive/${env:POD_NAME}/logs.json", exporter["path"])
	assert.Equal(t, true, exporter["append"])
	assert.NotContains(t, exporter, "rotation")
}

func TestLocalFileArchiveInvalidConfigDoesNotMutate(t *testing.T) {
	for _, data := range []map[string]string{
		{}, {LocalFileArchivePathKey: "relative"}, {LocalFileArchivePathKey: "/"}, {LocalFileArchivePathKey: "/archive/../etc"},
		{LocalFileArchivePathKey: "/archive/${env:HOME}"}, {LocalFileArchivePathKey: "/archive/*"},
		{LocalFileArchivePathKey: "/archive", fileArchiveFormatKey: "csv"},
		{LocalFileArchivePathKey: "/archive", fileArchiveCompressionKey: "gzip"},
		{LocalFileArchivePathKey: "/archive", fileArchiveMaxMegabytesKey: "bad"},
		{LocalFileArchivePathKey: "/archive", fileArchiveMaxMegabytesKey: "-1"},
		{LocalFileArchivePathKey: "/archive", fileArchiveMaxBackupsKey: "0"},
		{LocalFileArchivePathKey: "/archive", fileArchiveMaxDaysKey: "-1"},
	} {
		cfg := Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
		_, err := (&LocalFileArchive{}).ModifyConfig(&mockDestination{id: "archive", config: data}, &cfg)
		require.Error(t, err)
		assert.Empty(t, cfg.Exporters)
		assert.Empty(t, cfg.Service.Pipelines)
	}
	for _, id := range []string{"", ".", "..", "../other", "a/b", "${env:HOME}"} {
		cfg := Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
		_, err := (&LocalFileArchive{}).ModifyConfig(&mockDestination{id: id, config: map[string]string{LocalFileArchivePathKey: "/archive"}}, &cfg)
		require.Error(t, err)
	}
	for _, signal := range []common.ObservabilitySignal{common.ProfilesObservabilitySignal, "unsupported"} {
		cfg := Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
		_, err := (&LocalFileArchive{}).ModifyConfig(&mockDestination{id: "archive", config: map[string]string{LocalFileArchivePathKey: "/archive"}, signals: []common.ObservabilitySignal{signal}}, &cfg)
		require.Error(t, err)
	}
}
