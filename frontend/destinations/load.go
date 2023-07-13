package destinations

import (
	"embed"

	"gopkg.in/yaml.v3"
)

//go:embed data/*
var destsFS embed.FS

var loadedDestinations []Destination

func Load() error {
	return load(destsFS)
}

func Get() []Destination {
	return loadedDestinations
}

func load(fs embed.FS) error {
	var dests []Destination

	// load all files in the data directory
	files, err := fs.ReadDir("data")
	if err != nil {
		return err
	}

	for _, file := range files {
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

		dests = append(dests, dest)
	}

	loadedDestinations = dests
	return nil
}
