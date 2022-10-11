package backend

import (
	"github.com/keyval-dev/odigos/common"
	"github.com/spf13/cobra"
	"strings"
)

type ObservabilityArgs struct {
	Data   map[string]string
	Secret map[string]string
}

type ObservabilityBackend interface {
	Name() common.DestinationType
	ParseFlags(cmd *cobra.Command, selectedSignals []common.ObservabilitySignal) (*ObservabilityArgs, error)
	SupportedSignals() []common.ObservabilitySignal
}

var (
	availableBackends = []ObservabilityBackend{&Datadog{}, &Honeycomb{}, &Grafana{},
		&NewRelic{}, &LogzIO{}, &Prometheus{}, &Tempo{}, &Loki{}}
	backendsMap = calcBackendsMap()
)

func calcBackendsMap() map[string]ObservabilityBackend {
	backends := make(map[string]ObservabilityBackend, len(availableBackends))
	for i, b := range availableBackends {
		backends[strings.ToLower(string(b.Name()))] = availableBackends[i]
	}

	return backends
}

func GetAvailableBackends() []string {
	var names []string
	for n := range backendsMap {
		names = append(names, n)
	}

	return names
}

func Get(name string) ObservabilityBackend {
	b, ok := backendsMap[strings.ToLower(name)]
	if !ok {
		return nil
	}

	return b
}
