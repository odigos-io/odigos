package tools

import (
	"regexp"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	corev1 "k8s.io/api/core/v1"
)

func TestParsePromCountersSumsAcrossLabels(t *testing.T) {
	body := `# HELP otelcol_receiver_accepted_spans foo
# TYPE otelcol_receiver_accepted_spans counter
otelcol_receiver_accepted_spans{receiver="otlp",transport="grpc"} 100
otelcol_receiver_accepted_spans{receiver="otlp",transport="http"} 25
otelcol_exporter_sent_spans{exporter="otlp/destA"} 50
otelcol_exporter_sent_spans{exporter="otlp/destB"} 200
otelcol_uninteresting_metric{x="y"} 999
`
	got := ParsePromCountersForNames(body, []string{
		"otelcol_receiver_accepted_spans",
		"otelcol_exporter_sent_spans",
	})
	if got["otelcol_receiver_accepted_spans"] != 125 {
		t.Errorf("accepted_spans: got %v want 125", got["otelcol_receiver_accepted_spans"])
	}
	if got["otelcol_exporter_sent_spans"] != 250 {
		t.Errorf("exporter_sent_spans: got %v want 250", got["otelcol_exporter_sent_spans"])
	}
	if _, present := got["otelcol_uninteresting_metric"]; present {
		t.Error("uninteresting metric must not be returned")
	}
}

func TestParsePromCountersHandlesNoLabels(t *testing.T) {
	body := "otelcol_processor_dropped_spans 7\n"
	got := ParsePromCountersForNames(body, []string{"otelcol_processor_dropped_spans"})
	if got["otelcol_processor_dropped_spans"] != 7 {
		t.Errorf("dropped_spans: got %v want 7", got["otelcol_processor_dropped_spans"])
	}
}

func TestParsePromLineSkipsMalformed(t *testing.T) {
	cases := []string{
		"",
		"# comment",
		"only_a_name",
		"name{unbalanced 12",
		"name{a=\"b\"} not_a_number",
	}
	for _, line := range cases {
		got := ParsePromCountersForNames(line+"\n", []string{"name"})
		if len(got) != 0 {
			t.Errorf("input %q should yield no metrics, got %v", line, got)
		}
	}
}

func TestParseCollectorYAMLDecodesPipelines(t *testing.T) {
	yamlText := `
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
exporters:
  otlp/destA:
    endpoint: https://api.example.com:4317
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlp/destA]
`
	parsed, err := parseCollectorYAML(yamlText)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	exporters, ok := parsed["exporters"].(map[string]any)
	if !ok {
		t.Fatalf("expected exporters map, got %T", parsed["exporters"])
	}
	if _, has := exporters["otlp/destA"]; !has {
		t.Error("expected exporter otlp/destA")
	}
	service, ok := parsed["service"].(map[string]any)
	if !ok {
		t.Fatalf("expected service map, got %T", parsed["service"])
	}
	pipelines, ok := service["pipelines"].(map[string]any)
	if !ok {
		t.Fatalf("expected pipelines map, got %T", service["pipelines"])
	}
	if _, has := pipelines["traces"]; !has {
		t.Error("expected traces pipeline")
	}
}

func TestFilterLinesByRegex(t *testing.T) {
	text := "info: hello\nerror: bang\nwarning: meh\nerror: again\n"
	pattern := regexp.MustCompile(`(?i)error`)
	got := filterLinesByRegex(text, pattern)
	if !strings.Contains(got, "error: bang") || !strings.Contains(got, "error: again") {
		t.Errorf("missing error lines in %q", got)
	}
	if strings.Contains(got, "hello") || strings.Contains(got, "warning") {
		t.Errorf("non-matching lines leaked: %q", got)
	}
}

func TestIsPodReady(t *testing.T) {
	cases := []struct {
		name  string
		pod   corev1.Pod
		ready bool
	}{
		{"running and ready", corev1.Pod{
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionTrue},
				},
			},
		}, true},
		{"running but not ready", corev1.Pod{
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				Conditions: []corev1.PodCondition{
					{Type: corev1.PodReady, Status: corev1.ConditionFalse},
				},
			},
		}, false},
		{"pending", corev1.Pod{
			Status: corev1.PodStatus{Phase: corev1.PodPending},
		}, false},
		{"no ready condition", corev1.Pod{
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		}, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isPodReady(&testCase.pod); got != testCase.ready {
				t.Errorf("got %v want %v", got, testCase.ready)
			}
		})
	}
}

func TestCollectorResourceNamesAndConfig(t *testing.T) {
	groupName, workloadName := collectorResourceNames(string(k8sconsts.CollectorsRoleClusterGateway))
	if groupName != k8sconsts.OdigosClusterCollectorCollectorGroupName {
		t.Errorf("cluster gateway group: got %q", groupName)
	}
	if workloadName != k8sconsts.OdigosClusterCollectorDeploymentName {
		t.Errorf("cluster gateway workload: got %q", workloadName)
	}

	configMapName, configKey := collectorConfigCoordinates(string(k8sconsts.CollectorsRoleNodeCollector))
	if configMapName != k8sconsts.OdigosNodeCollectorConfigMapName {
		t.Errorf("node configmap: got %q", configMapName)
	}
	if configKey != k8sconsts.OdigosNodeCollectorConfigMapKey {
		t.Errorf("node configmap key: got %q", configKey)
	}
}

func TestCollectorRoleLabelSelector(t *testing.T) {
	want := "odigos.io/collector-role=CLUSTER_GATEWAY"
	if got := collectorRoleLabelSelector("CLUSTER_GATEWAY"); got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
