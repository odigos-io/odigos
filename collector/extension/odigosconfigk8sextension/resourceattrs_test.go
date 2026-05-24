package odigosconfigk8sextension

import (
	"testing"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestWorkloadContainerKeyFromResourceAttributesPrefersCronJobOverJob(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "ns")
	attrs.PutStr(string(semconv.K8SCronJobNameKey), "scheduled-job")
	attrs.PutStr(string(semconv.K8SJobNameKey), "scheduled-job-29292929")

	key, err := workloadContainerKeyFromResourceAttributes(attrs)

	require.NoError(t, err)
	require.Equal(t, "ns/CronJob/scheduled-job/", key)
}

func TestWorkloadContainerKeyFromResourceAttributesPrefersOdigosWorkloadIdentity(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "ns")
	attrs.PutStr(string(semconv.K8SDeploymentNameKey), "openshift-app")
	attrs.PutStr(consts.OdigosWorkloadKindAttribute, "DeploymentConfig")
	attrs.PutStr(consts.OdigosWorkloadNameAttribute, "openshift-app")

	key, err := workloadContainerKeyFromResourceAttributes(attrs)

	require.NoError(t, err)
	require.Equal(t, "ns/DeploymentConfig/openshift-app/", key)
}

func TestWorkloadContainerKeyFromResourceAttributesUsesKindSpecificFallbackName(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), "ns")
	attrs.PutStr(string(semconv.K8SDeploymentNameKey), "openshift-app")
	attrs.PutStr(consts.OdigosWorkloadKindAttribute, "DeploymentConfig")

	key, err := workloadContainerKeyFromResourceAttributes(attrs)

	require.NoError(t, err)
	require.Equal(t, "ns/DeploymentConfig/openshift-app/", key)
}
