package otheragent

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

func ctx(cmdline string, detailed, overwrite map[string]string) *process.ProcessContext {
	if detailed == nil {
		detailed = map[string]string{}
	}
	if overwrite == nil {
		overwrite = map[string]string{}
	}
	return process.NewProcessContext(process.Details{
		ProcessID: -1, // no such pid; maps reads are expected to fail gracefully
		CmdLine:   cmdline,
		Environments: process.ProcessEnvs{
			DetailedEnvs:  detailed,
			OverwriteEnvs: overwrite,
		},
	})
}

func TestDetect(t *testing.T) {
	const unknown = common.UnknownProgrammingLanguage
	tests := []struct {
		name      string
		pcx       *process.ProcessContext
		lang      common.ProgrammingLanguage
		wantAgent string // "" means no detection
	}{
		// ---- parity with the original 3-agent detector (language-agnostic signals) ----
		{"newrelic env", ctx("", map[string]string{"NEW_RELIC_CONFIG_FILE": "/nr.yml"}, nil), unknown, NewRelicAgentName},
		{"newrelic jar cmdline", ctx("java -javaagent:/opt/newrelic.jar -jar app.jar", nil, nil),
			common.JavaProgrammingLanguage, NewRelicAgentName},
		{"datadog agent url", ctx("", map[string]string{"DD_TRACE_AGENT_URL": "http://localhost:8126"}, nil), unknown, DatadogAgentName},
		{"dynatrace dynamizer env", ctx("", map[string]string{"DT_DYNAMIZER_TARGET_EXE": "x"}, nil), unknown, DynatraceAgentName},
		{"dynatrace ld_preload overwrite",
			ctx("", nil, map[string]string{"LD_PRELOAD": "/opt/dynatrace/oneagent/lib64/liboneagentproc.so"}), unknown, DynatraceAgentName},

		// ---- widened Datadog per-language coverage (language-scoped) ----
		{"datadog dotnet guid", ctx("", map[string]string{"CORECLR_PROFILER": "{846F5F1C-F9AE-4B07-969E-05C26BC060D8}"}, nil),
			common.DotNetProgrammingLanguage, DatadogAgentName},
		{"datadog node options", ctx("", map[string]string{"NODE_OPTIONS": "--require dd-trace/init"}, nil),
			common.JavascriptProgrammingLanguage, DatadogAgentName},
		{"datadog python wrapper", ctx("/usr/bin/ddtrace-run python app.py", nil, nil),
			common.PythonProgrammingLanguage, DatadogAgentName},
		{"datadog ssi preload", ctx("", nil, map[string]string{"LD_PRELOAD": "/opt/datadog-packages/datadog-apm-inject/inject.so"}),
			unknown, DatadogAgentName},

		// ---- language scoping: a .NET GUID must NOT match a Java process ----
		{"dotnet guid ignored for java process", ctx("", map[string]string{"CORECLR_PROFILER": "{846F5F1C-F9AE-4B07-969E-05C26BC060D8}"}, nil),
			common.JavaProgrammingLanguage, ""},

		// ---- OpenTelemetry family ----
		{"otel javaagent", ctx("java -javaagent:/otel/opentelemetry-javaagent.jar -jar app.jar", nil, nil),
			common.JavaProgrammingLanguage, OpenTelemetryAgentName},
		{"grafana otel javaagent", ctx("java -javaagent:/opt/alloy/bin/grafana-opentelemetry-java-2-21.jar -jar app.jar", nil, nil),
			common.JavaProgrammingLanguage, GrafanaOtelAgentName},
		{"otel node",
			ctx("", map[string]string{"NODE_OPTIONS": "--require @opentelemetry/auto-instrumentations-node/register"}, nil),
			common.JavascriptProgrammingLanguage, OpenTelemetryAgentName},

		// ---- config-only envs are no longer signals (no false positives) ----
		{"datadog config env alone -> none", ctx("", map[string]string{"DD_SERVICE": "billing", "DD_ENV": "prod"}, nil), unknown, ""},

		// ---- clean process ----
		{"clean", ctx("java -jar app.jar", map[string]string{"PATH": "/usr/bin"}, nil), common.JavaProgrammingLanguage, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Detect(tt.pcx, tt.lang)
			if tt.wantAgent == "" {
				if got != nil {
					t.Fatalf("expected no detection, got %q", got.Name)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected %q, got nil", tt.wantAgent)
			}
			if got.Name != tt.wantAgent {
				t.Fatalf("expected %q, got %q", tt.wantAgent, got.Name)
			}
		})
	}
}

func TestEnvKeysOfInterest(t *testing.T) {
	keys := EnvKeysOfInterest()
	for _, want := range []string{
		"NEW_RELIC_CONFIG_FILE", "DD_TRACE_AGENT_URL", "DT_DYNAMIZER_TARGET_EXE",
		"NODE_OPTIONS", "JAVA_TOOL_OPTIONS", "CORECLR_PROFILER", "RUBYOPT", "LD_PRELOAD",
	} {
		if _, ok := keys[want]; !ok {
			t.Errorf("EnvKeysOfInterest missing %q", want)
		}
	}
}
