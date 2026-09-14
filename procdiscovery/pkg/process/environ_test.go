package process

import (
	"maps"
	"testing"
)

func TestParseRelevantEnvVars(t *testing.T) {
	runtimeDetectionEnvs := map[string]struct{}{
		"ODIGOS_SERVICE_NAME": {},
		"OTEL_SERVICE_NAME":   {},
	}

	t.Run("empty", func(t *testing.T) {
		envs := parseRelevantEnvVars(nil, runtimeDetectionEnvs)
		if len(envs.OverwriteEnvs) != 0 || len(envs.DetailedEnvs) != 0 {
			t.Fatalf("expected empty maps, got overwrite=%v detailed=%v", envs.OverwriteEnvs, envs.DetailedEnvs)
		}
	})

	t.Run("filters_to_watched_keys", func(t *testing.T) {
		payload := []byte(
			"PATH=/usr/bin\x00" +
				"NODE_VERSION=v20.11.0\x00" +
				"ODIGOS_SERVICE_NAME=checkout\x00" +
				"JAVA_TOOL_OPTIONS=-javaagent:/opt/dd.jar\x00" +
				"HOME=/home/app\x00" +
				"PYTHON_VERSION=3.11.6\x00",
		)
		envs := parseRelevantEnvVars(payload, runtimeDetectionEnvs)
		wantOverwrite := map[string]string{
			"ODIGOS_SERVICE_NAME": "checkout",
		}
		wantDetailed := map[string]string{
			"NODE_VERSION":      "v20.11.0",
			"JAVA_TOOL_OPTIONS": "-javaagent:/opt/dd.jar",
			"PYTHON_VERSION":    "3.11.6",
		}
		if !maps.Equal(envs.OverwriteEnvs, wantOverwrite) {
			t.Fatalf("OverwriteEnvs = %#v, want %#v", envs.OverwriteEnvs, wantOverwrite)
		}
		if !maps.Equal(envs.DetailedEnvs, wantDetailed) {
			t.Fatalf("DetailedEnvs = %#v, want %#v", envs.DetailedEnvs, wantDetailed)
		}
	})

	t.Run("handles_missing_trailing_nul_and_malformed", func(t *testing.T) {
		payload := []byte(
			"NODE_VERSION=v18.0.0\x00" +
				"NOT_A_PAIR\x00" +
				"=novalue\x00" +
				"JAVA_HOME=/usr/lib/jvm",
		)
		envs := parseRelevantEnvVars(payload, nil)
		if envs.OverwriteEnvs != nil {
			t.Fatalf("expected nil OverwriteEnvs, got %#v", envs.OverwriteEnvs)
		}
		wantDetailed := map[string]string{
			"NODE_VERSION": "v18.0.0",
			"JAVA_HOME":    "/usr/lib/jvm",
		}
		if !maps.Equal(envs.DetailedEnvs, wantDetailed) {
			t.Fatalf("DetailedEnvs = %#v, want %#v", envs.DetailedEnvs, wantDetailed)
		}
	})

	t.Run("value_may_contain_equals", func(t *testing.T) {
		payload := []byte("NODE_OPTIONS=--require=/a=b.js\x00")
		envs := parseRelevantEnvVars(payload, nil)
		wantDetailed := map[string]string{
			"NODE_OPTIONS": "--require=/a=b.js",
		}
		if !maps.Equal(envs.DetailedEnvs, wantDetailed) {
			t.Fatalf("DetailedEnvs = %#v, want %#v", envs.DetailedEnvs, wantDetailed)
		}
	})
}
