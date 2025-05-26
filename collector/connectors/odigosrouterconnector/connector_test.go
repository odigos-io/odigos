package odigosrouterconnector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestDetermineRoutingPipelines(t *testing.T) {
	rt := SignalRoutingMap{
		"default/deployment/my-app": {
			"traces": {"traces/B"},
		},
		"default/daemonset/log-agent": {
			"logs": {"logs/A", "logs/B"},
		},
		"default/statefulset/metricsd": {
			"metrics": {"metrics/X"},
		},
	}

	t.Run("deployment with single trace route", func(t *testing.T) {
		attrs := pcommon.NewMap()
		attrs.PutStr("k8s.namespace.name", "default")
		attrs.PutStr("k8s.deployment.name", "my-app")

		pipelines, key := determineRoutingPipelines(attrs, rt, "traces")
		assert.Equal(t, "default/deployment/my-app", key)
		assert.ElementsMatch(t, []string{"traces/B"}, pipelines)
	})

	t.Run("daemonset with multiple log routes", func(t *testing.T) {
		attrs := pcommon.NewMap()
		attrs.PutStr("k8s.namespace.name", "default")
		attrs.PutStr("k8s.daemonset.name", "log-agent")

		pipelines, key := determineRoutingPipelines(attrs, rt, "logs")
		assert.Equal(t, "default/daemonset/log-agent", key)
		assert.ElementsMatch(t, []string{"logs/A", "logs/B"}, pipelines)
	})

	t.Run("statefulset with metrics", func(t *testing.T) {
		attrs := pcommon.NewMap()
		attrs.PutStr("k8s.namespace.name", "default")
		attrs.PutStr("k8s.statefulset.name", "metricsd")

		pipelines, key := determineRoutingPipelines(attrs, rt, "metrics")
		assert.Equal(t, "default/statefulset/metricsd", key)
		assert.ElementsMatch(t, []string{"metrics/X"}, pipelines)
	})

	t.Run("missing namespace returns nil", func(t *testing.T) {
		attrs := pcommon.NewMap()
		attrs.PutStr("k8s.deployment.name", "my-app")

		pipelines, key := determineRoutingPipelines(attrs, rt, "traces")
		assert.Equal(t, "", key)
		assert.Nil(t, pipelines)
	})

	t.Run("missing workload name returns nil", func(t *testing.T) {
		attrs := pcommon.NewMap()
		attrs.PutStr("k8s.namespace.name", "default")

		pipelines, key := determineRoutingPipelines(attrs, rt, "traces")
		assert.Equal(t, "", key)
		assert.Nil(t, pipelines)
	})

	t.Run("workload not in map", func(t *testing.T) {
		attrs := pcommon.NewMap()
		attrs.PutStr("k8s.namespace.name", "default")
		attrs.PutStr("k8s.deployment.name", "ghost")

		pipelines, key := determineRoutingPipelines(attrs, rt, "traces")
		assert.Equal(t, "", key)
		assert.Empty(t, pipelines)
	})
}
