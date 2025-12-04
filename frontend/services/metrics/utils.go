package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// rateSumByPod builds a PromQL query that returns per-pod rates over a window.
// Uses VictoriaMetrics/OTel k8s labels and filters by pod only to be resilient
// to environments where k8s.namespace.name label is not present on the series.
// Example:
//
//	sum by (k8s.pod.name) (rate(<metric>{k8s.pod.name=~"<regex>"}[<win>]))
func rateSumByPod(metric, namespace, podRegex, window string) string {
	return fmt.Sprintf(
		`sum by (k8s.pod.name) (rate(%s{k8s.pod.name=~"%s"}[%s]))`,
		metric, podRegex, window,
	)
}

// buildPodRegex builds a regular expression that matches the pod names
// e.g. for a single pod name ["odiglet-8ncl7"], the function returns: "^(odiglet-8ncl7)$"
func buildPodRegex(podNames []string) string {
	escaped := make([]string, 0, len(podNames))
	for _, n := range podNames {
		escaped = append(escaped, regexpEscape(n))
	}
	return fmt.Sprintf("^(%s)$", strings.Join(escaped, "|"))
}

func queryVector(ctx context.Context, api v1.API, query string, ts time.Time) (map[string]float64, time.Time, error) {
	val, _, err := api.Query(ctx, query, ts)
	if err != nil {
		return nil, time.Time{}, err
	}
	vec, ok := val.(model.Vector)
	if !ok {
		return map[string]float64{}, ts, nil
	}
	res := make(map[string]float64, len(vec))
	for _, s := range vec {
		// Prefer the VM/OTel k8s label; fallback to common alternatives for robustness.
		pod := string(s.Metric["k8s.pod.name"])
		if pod == "" {
			pod = string(s.Metric["k8s_pod_name"])
		}
		if pod == "" {
			pod = string(s.Metric["pod"])
		}
		if pod == "" {
			pod = string(s.Metric["pod_name"])
		}
		res[pod] = float64(s.Value)
	}
	return res, ts, nil
}

// regexpEscape escapes regex metacharacters in s for safe use in PromQL regex matchers.
func regexpEscape(s string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		`.`, `\\.`,
		`+`, `\\+`,
		`*`, `\\*`,
		`?`, `\\?`,
		`|`, `\\|`,
		`{`, `\\{`,
		`}`, `\\}`,
		`(`, `\\(`,
		`)`, `\\)`,
		`^`, `\\^`,
		`$`, `\\$`,
		`[`, `\\[`,
		`]`, `\\]`,
	)
	return replacer.Replace(s)
}

func maxTime(times ...time.Time) time.Time {
	var z time.Time
	for _, t := range times {
		if t.After(z) {
			z = t
		}
	}
	return z
}
