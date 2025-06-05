package odigosurltemplateprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	// assert that the config is of type *Config
	assert.IsType(t, &Config{}, cfg)
	assert.Empty(t, cfg.(*Config).TemplatizationRules)
}

func TestCreateProcessor(t *testing.T) {
	factory := NewFactory()
	set := processortest.NewNopSettings(factory.Type())
	cfg := factory.CreateDefaultConfig()
	// assert that the config is of type *Config
	assert.IsType(t, &Config{}, cfg)

	// create a new processor
	tp, err := factory.CreateTraces(context.Background(), set, cfg, nil)
	assert.NotNil(t, tp)
	assert.NoError(t, err, "cannot create tracer processor")
	assert.NoError(t, tp.Shutdown(context.Background()))
}

func TestInvalidRules(t *testing.T) {
	tests := []struct {
		name string
		rule string
	}{
		{
			name: "empty-regexp",
			rule: "{foo:}",
		},
		{
			name: "invalid-regexp",
			rule: "{foo:.*[}",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factory := NewFactory()
			set := processortest.NewNopSettings(factory.Type())
			_, err := factory.CreateTraces(context.Background(), set, &Config{
				TemplatizationConfig: TemplatizationConfig{
					TemplatizationRules: []string{test.rule},
				},
			}, nil)
			assert.Error(t, err)
		})
	}
}
