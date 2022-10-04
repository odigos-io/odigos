package backend

import "github.com/spf13/cobra"

type ObservabilityBackend interface {
	Name() string
	ValidateFlags(cmd *cobra.Command) error
}

var (
	availableBackends = []ObservabilityBackend{&Datadog{}}
	backendsMap       = calcBackendsMap()
)

func calcBackendsMap() map[string]ObservabilityBackend {
	backends := make(map[string]ObservabilityBackend, len(availableBackends))
	for i, b := range availableBackends {
		backends[b.Name()] = availableBackends[i]
	}

	return backends
}

func GetAvailableBackends() []string {
	names := make([]string, len(availableBackends))
	for n := range backendsMap {
		names = append(names, n)
	}

	return names
}

func Get(name string) ObservabilityBackend {
	b, ok := backendsMap[name]
	if !ok {
		return nil
	}

	return b
}
