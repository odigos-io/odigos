package manifests

import (
	"embed"
)

//go:embed *.yaml
var embeddedFiles embed.FS

func ReadProfileManifestFile(filename string) ([]byte, error) {
	// Read from inside the package where the embeddedFiles variable is defined
	return embeddedFiles.ReadFile(filename)
}
