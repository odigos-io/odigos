package status

import (
	"embed"
)

//go:embed instrumentationconfig
var statusFS embed.FS

var loadedStatuses []Status

var statusesByType map[string]Status

func Load() error {
	manifests, err := LoadManifestsFromFS(statusFS, "instrumentationconfig")
	if err != nil {
		return err
	}

	statusesByTypeMap := make(map[string]Status, len(manifests))
	for _, s := range manifests {
		statusesByTypeMap[s.Spec.Type] = s
	}

	statusesByType = statusesByTypeMap
	loadedStatuses = manifests
	return nil
}

func Get() []Status {
	return loadedStatuses
}

func GetStatusByType(statusType string) (Status, bool) {
	s, ok := statusesByType[statusType]
	return s, ok
}
