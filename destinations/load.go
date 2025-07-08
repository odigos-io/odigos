package destinations

import (
	"embed"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed data/*
var destsFS embed.FS

// array of all destinations configs
var loadedDestinations []Destination

// map from destination type to destination config object
var destinationsByType map[string]Destination

func Load() error {
	return load(destsFS)
}

func Get() []Destination {
	return loadedDestinations
}

func GetDestinationByType(destType string) Destination {
	return destinationsByType[destType]
}

func load(fs embed.FS) error {
	var dests []Destination
	var destsByTypeMap = make(map[string]Destination)

	// load all files in the data directory
	files, err := fs.ReadDir("data")
	if err != nil {
		return err
	}

	for _, file := range files {
		fileName := file.Name()

		// check if fileName ends with .yaml or .yml
		if !strings.HasSuffix(fileName, ".yaml") && !strings.HasSuffix(fileName, ".yml") {
			continue
		}

		// load each file
		bytesData, err := fs.ReadFile("data/" + file.Name())
		if err != nil {
			return err
		}

		var dest Destination
		err = yaml.Unmarshal(bytesData, &dest)
		if err != nil {
			return err
		}

		destsByTypeMap[string(dest.Metadata.Type)] = dest
		dests = append(dests, dest)
	}

	destinationsByType = destsByTypeMap
	loadedDestinations = dests
	return nil
}
