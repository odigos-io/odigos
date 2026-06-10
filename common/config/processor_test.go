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
func (f fakeProcessor) GetType() string                         { return f.pType }
func (f fakeProcessor) GetID() string                           { return f.id }
func (f fakeProcessor) GetConfig() (GenericMap, error)          { return f.config, nil }
func (f fakeProcessor) GetOrderHint() int                       { return f.orderHint }

func TestCrdProcessorToConfig_ProfilesBucketing(t *testing.T) {
	profiles := []common.ObservabilitySignal{common.ProfilesObservabilitySignal}

	processors := []ProcessorConfigurer{
		// AddClusterInfo -> resource: profiles-capable, should be bucketed.
		fakeProcessor{id: "addclusterinfo", pType: "resource", signals: profiles, config: GenericMap{}},
		// RenameAttribute -> transform: profiles-capable, should be bucketed.
		fakeProcessor{id: "rename", pType: "transform", signals: profiles, config: GenericMap{}},
		// K8sAttributes -> k8sattributes: already applied unconditionally on the node profiles
		// pipeline, so it is intentionally excluded from the profiles bucket.
		fakeProcessor{id: "k8s", pType: "k8sattributes", signals: profiles, config: GenericMap{}},
		// SpanRenamer -> span-only, no profiles analog, excluded.
		fakeProcessor{id: "spanrename", pType: "odigosspanrenamer", signals: profiles, config: GenericMap{}},
	}

	results := CrdProcessorToConfig(processors)

	assert.ElementsMatch(t,
		[]string{"resource/addclusterinfo", "transform/rename"},
		results.ProfilesProcessors,
		"only resource and transform processors selecting PROFILES should be wired into profiles pipelines",
	)
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
