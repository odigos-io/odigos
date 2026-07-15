package rust

import (
	_ "embed"
	"encoding/json"
	"log"
)

//go:generate go run ./generate/main.go

//go:embed rust_versions.json
var rustVersionsJSON []byte

// rustcHashToVersion maps rustc commit hashes to their semantic release versions.
// This allows efficiently resolving the runtime version without network calls.
var rustcHashToVersion map[string]string

func init() {
	if err := json.Unmarshal(rustVersionsJSON, &rustcHashToVersion); err != nil {
		log.Fatalf("Failed to parse embedded rust_versions.json: %v", err)
	}
}
