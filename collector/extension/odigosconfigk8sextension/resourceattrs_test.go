package odigosconfigk8sextension

import (
	"testing"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestGetKindAndNamePrefersOdigosWorkloadIdentity(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SDeploymentNameKey), "checkout")
	attrs.PutStr(consts.OdigosWorkloadKindAttribute, "DeploymentConfig")
	attrs.PutStr(consts.OdigosWorkloadNameAttribute, "checkout-dc")

	kind, name := getKindAndName(attrs)

	require.Equal(t, "DeploymentConfig", kind)
	require.Equal(t, "checkout-dc", name)
}

func TestGetKindAndNameFallsBackToKubernetesAttributes(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SStatefulSetNameKey), "postgres")

	kind, name := getKindAndName(attrs)

	require.Equal(t, "StatefulSet", kind)
	require.Equal(t, "postgres", name)
}
