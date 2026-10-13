package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	LocalFileArchivePathKey    = "FILE_ARCHIVE_PATH"
	LocalFileArchivePVCKey     = "FILE_ARCHIVE_PVC"
	fileArchiveFormatKey       = "FILE_ARCHIVE_FORMAT"
	fileArchiveCompressionKey  = "FILE_ARCHIVE_COMPRESSION"
	fileArchiveMaxMegabytesKey = "FILE_ARCHIVE_MAX_MEGABYTES"
	fileArchiveMaxBackupsKey   = "FILE_ARCHIVE_MAX_BACKUPS"
	fileArchiveMaxDaysKey      = "FILE_ARCHIVE_MAX_DAYS"
)

type LocalFileArchive struct{}

func (f *LocalFileArchive) DestType() common.DestinationType {
	return common.LocalFileArchiveDestinationType
}

func LocalFileArchiveDirectory(data map[string]string) (string, error) {
	directory := data[LocalFileArchivePathKey]
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory ||
		directory == string(filepath.Separator) || strings.ContainsAny(directory, "$*?\\\n\r\x00") {
		return "", fmt.Errorf("%s must be an absolute, clean, non-root directory without environment substitutions or wildcards", LocalFileArchivePathKey)
	}
	return directory, nil
}

func (f *LocalFileArchive) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if isProfilingEnabled(dest) || (!isTracingEnabled(dest) && !isMetricsEnabled(dest) && !isLoggingEnabled(dest)) {
		return nil, fmt.Errorf("local file archive supports traces, metrics and logs only")
	}
	directory, err := LocalFileArchiveDirectory(dest.GetConfig())
	if err != nil {
		return nil, err
	}
	id := dest.GetID()
	if id == "" || id == "." || id == ".." || strings.ContainsAny(id, "/\\$*?\n\r\x00") {
		return nil, fmt.Errorf("invalid local file archive destination ID")
	}
	data := dest.GetConfig()
	format := data[fileArchiveFormatKey]
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "proto" {
		return nil, fmt.Errorf("%s must be json or proto", fileArchiveFormatKey)
	}
	compression := data[fileArchiveCompressionKey]
	if compression == "none" {
		compression = ""
	}
	if compression != "" && compression != "zstd" {
		return nil, fmt.Errorf("%s must be none or zstd", fileArchiveCompressionKey)
	}
	rotation := GenericMap{}
	for _, setting := range []struct {
		key, field            string
		defaultValue, minimum int
	}{
		{fileArchiveMaxMegabytesKey, "max_megabytes", 100, 0},
		{fileArchiveMaxBackupsKey, "max_backups", 10, 1},
		{fileArchiveMaxDaysKey, "max_days", 7, 0},
	} {
		value := setting.defaultValue
		if raw := data[setting.key]; raw != "" {
			value, err = parseInt(setting.key, raw)
			if err != nil {
				return nil, err
			}
		}
		if value < setting.minimum {
			return nil, fmt.Errorf("%s must be at least %d", setting.key, setting.minimum)
		}
		rotation[setting.field] = value
	}
	directory = filepath.Join(directory, id)
	if data[LocalFileArchivePVCKey] != "" {
		directory = filepath.Join(directory, "${env:POD_NAME}")
	}
	var pipelines []string
	for _, signal := range []common.ObservabilitySignal{
		common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal,
	} {
		if !isSignalExists(dest, signal) {
			continue
		}
		signalName := strings.ToLower(string(signal))
		name := "file/" + id + "-" + signalName
		extension := ".json"
		if format == "proto" {
			extension = ".bin"
		}
		if compression != "" {
			extension += ".zst"
		}
		exporter := GenericMap{
			"path":                  filepath.Join(directory, signalName+extension),
			"format":                format,
			"compression":           compression,
			"create_directory":      true,
			"directory_permissions": "0750",
		}
		if rotation["max_megabytes"].(int) == 0 {
			exporter["append"] = true
		} else {
			exporter["rotation"] = rotation
		}
		currentConfig.Exporters[name] = exporter
		pipeline := signalName + "/localfilearchive-" + id
		currentConfig.Service.Pipelines[pipeline] = Pipeline{Exporters: []string{name}}
		pipelines = append(pipelines, pipeline)
	}
	return pipelines, nil
}
