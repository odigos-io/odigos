package odigosconfigk8sextension

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestWorkloadIdentityFromResourceAttributes(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "default")
	attrs.PutStr(string(semconv.K8SDeploymentNameKey), "checkout")
	attrs.PutStr(string(semconv.K8SContainerNameKey), "app")

	cacheKey, identityAttrs, err := workloadIdentityFromResourceAttributes(attrs)
	require.NoError(t, err)
	require.Equal(t, "default/Deployment/checkout/app", cacheKey)

	namespace, ok := identityAttrs.Get(string(semconv.K8SNamespaceNameKey))
	require.True(t, ok)
	require.Equal(t, "default", namespace.Str())

	deployment, ok := identityAttrs.Get(string(semconv.K8SDeploymentNameKey))
	require.True(t, ok)
	require.Equal(t, "checkout", deployment.Str())

	container, ok := identityAttrs.Get(string(semconv.K8SContainerNameKey))
	require.True(t, ok)
	require.Equal(t, "app", container.Str())
}

func TestWorkloadIdentityFromResourceAttributes_MissingAttributes(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "default")

	_, _, err := workloadIdentityFromResourceAttributes(attrs)
	require.Error(t, err)
}
