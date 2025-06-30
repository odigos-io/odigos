package workload_test

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/tj/assert"
)

func TestRuntimeObjectName(t *testing.T) {
	runtimeObjectName := workload.CalculateWorkloadRuntimeObjectName("my-app", "Deployment")
	assert.Equal(t, "deployment-my-app", runtimeObjectName)
}

func TestExtractDeploymentWorkloadInfoFromRuntimeObjectName(t *testing.T) {
	runtimeObjectName := workload.CalculateWorkloadRuntimeObjectName("my-app", "Deployment")
	pw, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(runtimeObjectName, "test")
	assert.Nil(t, err)
	assert.Equal(t, "my-app", pw.Name)
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, pw.Kind)
}

func TestExtractDaemonSetWorkloadInfoFromRuntimeObjectName(t *testing.T) {
	runtimeObjectName := workload.CalculateWorkloadRuntimeObjectName("my-app", "DaemonSet")
	pw, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(runtimeObjectName, "test")
	assert.Nil(t, err)
	assert.Equal(t, "my-app", pw.Name)
	assert.Equal(t, k8sconsts.WorkloadKindDaemonSet, pw.Kind)
}

func TestExtractStatefulSetWorkloadInfoFromRuntimeObjectName(t *testing.T) {
	runtimeObjectName := workload.CalculateWorkloadRuntimeObjectName("my-app", "StatefulSet")
	pw, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(runtimeObjectName, "test")
	assert.Nil(t, err)
	assert.Equal(t, "my-app", pw.Name)
	assert.Equal(t, k8sconsts.WorkloadKindStatefulSet, pw.Kind)
}

func TestExtractInvalidWorkloadInfoFromRuntimeObjectName(t *testing.T) {
	_, err := workload.ExtractWorkloadInfoFromRuntimeObjectName("nohyphen", "test")
	assert.NotNil(t, err)
}

func TestExtractUnknownWorkloadInfoFromRuntimeObjectName(t *testing.T) {
	_, err := workload.ExtractWorkloadInfoFromRuntimeObjectName("unknownkind-my-app", "test")
	assert.NotNil(t, err)
}
