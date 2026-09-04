package clustercollector

import (
	"testing"

	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

func TestAddInterrogationExporters(t *testing.T) {
	enabled := true
	cfg := &config.Config{
		Extensions: config.GenericMap{},
		Exporters:  config.GenericMap{},
		Service: config.Service{Pipelines: map[string]config.Pipeline{
			"profiles":  {},
			"traces/in": {},
		}},
	}
	addInterrogationExporters(cfg, "odigos-system", &common.InterrogationConfiguration{Enabled: &enabled})
	traceCfg, ok := cfg.Exporters[commonconf.InterrogationTracesExporter].(config.GenericMap)
	if !ok {
		t.Fatal("trace exporter missing")
	}
	if got := traceCfg["evidence_endpoint"]; got != "http://ui.odigos-system.svc:3000/api/interrogation/evidence" {
		t.Fatalf("endpoint = %v", got)
	}
	if _, ok := traceCfg["llm"]; ok {
		t.Fatal("collector-side LLM configuration was rendered")
	}
}

func TestAddInterrogationExportersDisabled(t *testing.T) {
	cfg := &config.Config{Service: config.Service{Pipelines: map[string]config.Pipeline{"traces/in": {}}}}
	addInterrogationExporters(cfg, "odigos-system", nil)
	if len(cfg.Exporters) != 0 {
		t.Fatalf("unexpected exporters: %#v", cfg.Exporters)
	}
}
