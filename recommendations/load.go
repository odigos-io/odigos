package recommendations

import (
	"embed"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
	"gopkg.in/yaml.v3"
)

//go:embed manifests/*
var recommendationsFS embed.FS

var loadedRecommendations []Recommendation

var recommendationsByType map[common.RecommendationType]Recommendation

var recommendationsByK8sObjectName map[string]Recommendation

func Load() error {
	return load(recommendationsFS)
}

func Get() []Recommendation {
	return loadedRecommendations
}

func GetByType(recommendationType common.RecommendationType) (Recommendation, bool) {
	recommendation, ok := recommendationsByType[recommendationType]
	return recommendation, ok
}

func GetByK8sObjectName(k8sObjectName string) (Recommendation, bool) {
	recommendation, ok := recommendationsByK8sObjectName[k8sObjectName]
	return recommendation, ok
}

// RemediationByType returns the remediation with the given type within this recommendation.
func (r Recommendation) RemediationByType(remediationType string) (Remediation, bool) {
	for _, remediation := range r.Remediations {
		if remediation.Type == remediationType {
			return remediation, true
		}
	}
	return Remediation{}, false
}

func load(fs embed.FS) error {
	var recs []Recommendation
	recsByType := make(map[common.RecommendationType]Recommendation)
	recsByK8sObjectName := make(map[string]Recommendation)

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
		if rec.K8sObjectName == "" {
			return fmt.Errorf("recommendation %q in %s: k8sObjectName is required", rec.Type, fileName)
		}
		if _, exists := recsByK8sObjectName[rec.K8sObjectName]; exists {
			return fmt.Errorf("duplicate k8sObjectName %q in %s", rec.K8sObjectName, fileName)
		}

		recsByType[rec.Type] = rec
		recsByK8sObjectName[rec.K8sObjectName] = rec
		recs = append(recs, rec)
	}

	recommendationsByType = recsByType
	recommendationsByK8sObjectName = recsByK8sObjectName
	loadedRecommendations = recs
	return nil
}
