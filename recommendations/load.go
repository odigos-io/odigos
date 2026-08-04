package recommendations

import (
	"embed"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed manifests/*
var recommendationsFS embed.FS

var loadedRecommendations []Recommendation

var recommendationsByType map[RecommendationType]Recommendation

func Load() error {
	return load(recommendationsFS)
}

func Get() []Recommendation {
	return loadedRecommendations
}

func GetByType(recommendationType RecommendationType) (Recommendation, bool) {
	recommendation, ok := recommendationsByType[recommendationType]
	return recommendation, ok
}

func load(fs embed.FS) error {
	var recs []Recommendation
	recsByType := make(map[RecommendationType]Recommendation)

	files, err := fs.ReadDir("manifests")
	if err != nil {
		return err
	}

	for _, file := range files {
		fileName := file.Name()
		if !strings.HasSuffix(fileName, ".yaml") && !strings.HasSuffix(fileName, ".yml") {
			continue
		}

		bytesData, err := fs.ReadFile("manifests/" + fileName)
		if err != nil {
			return err
		}

		var rec Recommendation
		if err := yaml.Unmarshal(bytesData, &rec); err != nil {
			return err
		}

		recsByType[rec.Type] = rec
		recs = append(recs, rec)
	}

	recommendationsByType = recsByType
	loadedRecommendations = recs
	return nil
}
