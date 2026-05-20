package odigosconfigk8sextension

import (
	"testing"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestWorkloadContainerKeyFromResourceAttributesPrefersOdigosWorkloadIdentity(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "prod")
	attrs.PutStr(string(semconv.K8SDeploymentNameKey), "api")
	attrs.PutStr(consts.OdigosWorkloadKindAttribute, "DeploymentConfig")
	attrs.PutStr(consts.OdigosWorkloadNameAttribute, "api")

	key, err := workloadContainerKeyFromResourceAttributes(attrs)
	require.NoError(t, err)
	require.Equal(t, "prod/DeploymentConfig/api/", key)
}

func TestWorkloadContainerKeyFromResourceAttributesFallsBackToK8sAttributes(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "prod")
	attrs.PutStr(string(semconv.K8SDeploymentNameKey), "api")

	key, err := workloadContainerKeyFromResourceAttributes(attrs)
	require.NoError(t, err)
	require.Equal(t, "prod/Deployment/api/", key)
}
