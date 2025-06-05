package yamls

import (
	"embed"
)

//go:embed *.yaml
var embeddedFiles embed.FS

func GetFS() embed.FS {
	return embeddedFiles
}
