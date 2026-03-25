package config

import (
	"embed"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed data/*
var configFS embed.FS

var loadedConfigurations []Configuration

func Load() error {
	return load(configFS)
}

func Get() []Configuration {
	return loadedConfigurations
}

func load(fs embed.FS) error {
	var configs []Configuration

	files, err := fs.ReadDir("data")
	if err != nil {
		return err
	}

	for _, file := range files {
		fileName := file.Name()
		if !strings.HasSuffix(fileName, ".yaml") && !strings.HasSuffix(fileName, ".yml") {
			continue
		}

		bytesData, err := fs.ReadFile("data/" + fileName)
		if err != nil {
			return err
		}

		var cfg Configuration
		err = yaml.Unmarshal(bytesData, &cfg)
		if err != nil {
			return err
		}

		configs = append(configs, cfg)
	}

	loadedConfigurations = configs
	return nil
}
