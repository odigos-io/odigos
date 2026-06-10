package odigosconfigk8sextension

import (
	"testing"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestWorkloadContainerKeyFromResourceAttributesPrefersOdigosWorkloadKind(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "apps")
	attrs.PutStr(string(semconv.K8SDeploymentNameKey), "payments")
	attrs.PutStr(consts.OdigosWorkloadKindAttribute, "DeploymentConfig")
	attrs.PutStr(consts.OdigosWorkloadNameAttribute, "payments")

	key, err := workloadContainerKeyFromResourceAttributes(attrs)

	require.NoError(t, err)
	require.Equal(t, "apps/DeploymentConfig/payments/", key)
}
