package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
)

type fakeProcessor struct {
	id        string
	pType     string
	signals   []common.ObservabilitySignal
	config    GenericMap
	orderHint int
}

func (f fakeProcessor) GetSignals() []common.ObservabilitySignal { return f.signals }
func (f fakeProcessor) GetType() string                          { return f.pType }
func (f fakeProcessor) GetID() string                            { return f.id }
func (f fakeProcessor) GetConfig() (GenericMap, error)           { return f.config, nil }
func (f fakeProcessor) GetOrderHint() int                        { return f.orderHint }

func TestCrdProcessorToConfig_ProfilesBucketing(t *testing.T) {
	profiles := []common.ObservabilitySignal{common.ProfilesObservabilitySignal}
	traces := []common.ObservabilitySignal{common.TracesObservabilitySignal}

	processors := []ProcessorConfigurer{
		fakeProcessor{id: "addclusterinfo", pType: "resource", signals: profiles, config: GenericMap{}},
		fakeProcessor{id: "rename", pType: "transform", signals: profiles, config: GenericMap{}},
		// Not selecting PROFILES: must not be bucketed into profiles.
		fakeProcessor{id: "traces-only", pType: "resource", signals: traces, config: GenericMap{}},
	}

	results := CrdProcessorToConfig(processors)

	// Profiles is bucketed like the other signals: any processor selecting PROFILES is included.
	assert.ElementsMatch(t,
		[]string{"resource/addclusterinfo", "transform/rename"},
		results.ProfilesProcessors,
		"every processor selecting PROFILES should be wired into profiles pipelines",
	)
	assert.NotContains(t, results.ProfilesProcessors, "resource/traces-only")
}

func TestCrdProcessorToConfig_ProfilesNotSelected(t *testing.T) {
	// A profiles-capable type that did NOT select the PROFILES signal must not be bucketed.
	processors := []ProcessorConfigurer{
		fakeProcessor{
			id:      "rename-traces-only",
			pType:   "transform",
			signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
			config:  GenericMap{},
		},
	}

	results := CrdProcessorToConfig(processors)

	assert.Empty(t, results.ProfilesProcessors)
	assert.Equal(t, []string{"transform/rename-traces-only"}, results.TracesProcessors)
}
