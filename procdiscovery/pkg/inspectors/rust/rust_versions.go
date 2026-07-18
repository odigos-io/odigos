package rust

import (
	_ "embed"
	"log"

	"gopkg.in/yaml.v3"
)

// The generator lives outside this package (procdiscovery is meant to be
// consumed, not to host tooling) at scripts/rust-versions-gen. It appends
// newly discovered hash/version pairs to the end of rust_versions.yaml
// rather than rewriting the whole file, so re-running it produces a minimal,
// reviewable diff.
//go:generate sh -c "cd ../../../../scripts/rust-versions-gen && go run main.go"

//go:embed rust_versions.yaml
var rustVersionsYAML []byte

// rustcHashToVersion maps rustc commit hashes to their semantic release versions.
// This allows efficiently resolving the runtime version without network calls.
var rustcHashToVersion map[string]string

func init() {
	if err := yaml.Unmarshal(rustVersionsYAML, &rustcHashToVersion); err != nil {
		log.Fatalf("Failed to parse embedded rust_versions.yaml: %v", err)
	}
}
