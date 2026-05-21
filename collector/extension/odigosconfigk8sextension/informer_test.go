package odigosconfigk8sextension

import (
	"testing"

	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestHandleInstrumentationConfigMarksEnabledContainersAsManagedWithoutCollectorConfig(t *testing.T) {
	ext := newTestOdigosWorkloadConfig()

	ext.handleInstrumentationConfig(instrumentationConfig("ns", "deployment-app", map[string]interface{}{
		"containers": []interface{}{
			map[string]interface{}{
				"containerName": "app",
				"agentEnabled":  true,
			},
		},
	}))

	res := resourceWithWorkload("ns", string(semconv.K8SDeploymentNameKey), "app")
	require.True(t, ext.HasCachedWorkloadContainerConfig(res))

	_, found := ext.GetFromResource(res)
	require.False(t, found, "enabled agents without collector-side rules should not create dynamic collector config rows")

	ext.handleInstrumentationConfig(instrumentationConfig("ns", "deployment-app", map[string]interface{}{
		"containers": []interface{}{
			map[string]interface{}{
				"containerName": "app",
				"agentEnabled":  false,
			},
		},
	}))

	require.False(t, ext.HasCachedWorkloadContainerConfig(res))
}

func TestHandleInstrumentationConfigPreservesDeploymentConfigIdentityFromOdigosAttributes(t *testing.T) {
	ext := newTestOdigosWorkloadConfig()

	ext.handleInstrumentationConfig(instrumentationConfig("ns", "deploymentconfig-app", map[string]interface{}{
		"containers": []interface{}{
			map[string]interface{}{
				"containerName": "app",
				"agentEnabled":  true,
			},
		},
	}))

	res := resourceWithWorkload("ns", string(semconv.K8SDeploymentNameKey), "app")
	res.Attributes().PutStr(consts.OdigosWorkloadKindAttribute, "DeploymentConfig")
	res.Attributes().PutStr(consts.OdigosWorkloadNameAttribute, "app")

	require.True(t, ext.HasCachedWorkloadContainerConfig(res))
}

func TestHandleInstrumentationConfigKeepsCollectorConfigRowsSeparateFromManagedWorkloads(t *testing.T) {
	ext := newTestOdigosWorkloadConfig()

	ext.handleInstrumentationConfig(instrumentationConfig("ns", "deployment-app", map[string]interface{}{
		"containers": []interface{}{
			map[string]interface{}{
				"containerName": "app",
				"agentEnabled":  true,
			},
		},
		"workloadCollectorConfig": []interface{}{
			map[string]interface{}{
				"containerName": "app",
				"payloadCollection": map[string]interface{}{
					"httpRequest": map[string]interface{}{"maxPayloadLength": int64(100)},
				},
			},
		},
	}))

	res := resourceWithWorkload("ns", string(semconv.K8SDeploymentNameKey), "app")
	cfg, found := ext.GetFromResource(res)
	require.True(t, found)
	require.IsType(t, &commonapi.ContainerCollectorConfig{}, cfg)
	require.True(t, ext.HasCachedWorkloadContainerConfig(res))
}

func newTestOdigosWorkloadConfig() *OdigosWorkloadConfig {
	return &OdigosWorkloadConfig{
		cache:  newCache(),
		logger: zap.NewNop(),
	}
}

func instrumentationConfig(namespace, name string, spec map[string]interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
			},
			"spec": spec,
		},
	}
}

func resourceWithWorkload(namespace, workloadAttr, workloadName string) pcommon.Resource {
	res := pcommon.NewResource()
	attrs := res.Attributes()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), namespace)
	attrs.PutStr(workloadAttr, workloadName)
	attrs.PutStr(string(semconv.K8SContainerNameKey), "app")
	return res
}
