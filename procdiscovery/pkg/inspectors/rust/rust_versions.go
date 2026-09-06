package rust

import (
	_ "embed"
	"log"

	"gopkg.in/yaml.v3"
)

//go:generate sh -c "cd ../../../../scripts/rust-versions-gen && go run main.go"

//go:embed rust_versions.yaml
var rustVersionsYAML []byte

var rustcHashToVersion map[string]string

func init() {
	if err := yaml.Unmarshal(rustVersionsYAML, &rustcHashToVersion); err != nil {
		log.Fatalf("Failed to parse embedded rust_versions.yaml: %v", err)
	}
}
