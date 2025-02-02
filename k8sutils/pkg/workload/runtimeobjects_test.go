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
	workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(runtimeObjectName)
	assert.Nil(t, err)
	assert.Equal(t, "my-app", workloadName)
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, workloadKind)
}

func TestExtractDaemonSetWorkloadInfoFromRuntimeObjectName(t *testing.T) {
	runtimeObjectName := workload.CalculateWorkloadRuntimeObjectName("my-app", "DaemonSet")
	workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(runtimeObjectName)
	assert.Nil(t, err)
	assert.Equal(t, "my-app", workloadName)
	assert.Equal(t, k8sconsts.WorkloadKindDaemonSet, workloadKind)
}

func TestExtractStatefulSetWorkloadInfoFromRuntimeObjectName(t *testing.T) {
	runtimeObjectName := workload.CalculateWorkloadRuntimeObjectName("my-app", "StatefulSet")
	workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(runtimeObjectName)
	assert.Nil(t, err)
	assert.Equal(t, "my-app", workloadName)
	assert.Equal(t, k8sconsts.WorkloadKindStatefulSet, workloadKind)
}

func TestExtractInvalidWorkloadInfoFromRuntimeObjectName(t *testing.T) {
	_, _, err := workload.ExtractWorkloadInfoFromRuntimeObjectName("nohyphen")
	assert.NotNil(t, err)
}

func TestExtractUnknownWorkloadInfoFromRuntimeObjectName(t *testing.T) {
	_, _, err := workload.ExtractWorkloadInfoFromRuntimeObjectName("unknownkind-my-app")
	assert.NotNil(t, err)
}
