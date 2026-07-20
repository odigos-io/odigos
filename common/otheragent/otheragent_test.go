package otheragent

import (
	"io"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/common"
)

// fakeProcess is a test Process implementation with no /proc dependency.
type fakeProcess struct {
	cmdline string
	envs    map[string]string
	maps    string
}

func (f fakeProcess) Cmdline() string { return f.cmdline }

func (f fakeProcess) LookupEnv(key string) (string, bool) {
	v, ok := f.envs[key]
	return v, ok
}

func (f fakeProcess) MapsReader() (io.Reader, error) {
	return strings.NewReader(f.maps), nil
}

// ctx merges the detailed and overwrite env sets into one lookup (LookupEnv does
// not distinguish them), matching how the real Process implementations behave.
func ctx(cmdline string, detailed, overwrite map[string]string) fakeProcess {
	envs := map[string]string{}
	for k, v := range detailed {
		envs[k] = v
	}
	for k, v := range overwrite {
		envs[k] = v
	}
	return fakeProcess{cmdline: cmdline, envs: envs}
}

func TestDetect(t *testing.T) {
	const unknown = common.UnknownProgrammingLanguage
	tests := []struct {
		name      string
		pcx       Process
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

func TestDetectAll_MultiAgent(t *testing.T) {
	// A process carrying BOTH Datadog and New Relic (language-agnostic) signals.
	pcx := ctx("", map[string]string{
		"DD_TRACE_AGENT_URL":    "http://localhost:8126",
		"NEW_RELIC_CONFIG_FILE": "/nr.yml",
	}, nil)

	got := DetectAll(pcx, common.UnknownProgrammingLanguage)
	names := map[string]bool{}
	for _, a := range got {
		names[a.Name] = true
	}
	if len(got) != 2 || !names[DatadogAgentName] || !names[NewRelicAgentName] {
		t.Fatalf("DetectAll = %v, want both Datadog + New Relic", got)
	}

	if !Blocks(got, false) {
		t.Error("Blocks(2 agents, allow=false) = false, want true")
	}
	if Blocks(got, true) {
		t.Error("Blocks(2 agents, allow=true) = true, want false")
	}
	if Blocks(nil, false) {
		t.Error("Blocks(nil, allow=false) = true, want false")
	}
}

func TestDetectAll_DedupSameAgent(t *testing.T) {
	// Datadog matched via TWO signals must be reported once.
	pcx := ctx("", map[string]string{
		"DD_TRACE_AGENT_URL":   "http://localhost:8126",
		"DD_INJECTION_ENABLED": "true",
	}, nil)
	got := DetectAll(pcx, common.UnknownProgrammingLanguage)
	if len(got) != 1 || got[0].Name != DatadogAgentName {
		t.Fatalf("DetectAll dedup = %v, want single Datadog", got)
	}
}

func TestLibLoadedDetection(t *testing.T) {
	// Dynatrace liboneagent present in the process maps.
	p := fakeProcess{
		maps: "7f0000000000-7f0000001000 r-xp 00000000 00:00 0 /opt/dynatrace/oneagent/agent/lib64/liboneagentproc.so\n",
	}
	got := Detect(p, common.UnknownProgrammingLanguage)
	if got == nil || got.Name != DynatraceAgentName {
		t.Fatalf("expected %q from maps, got %v", DynatraceAgentName, got)
	}
}
