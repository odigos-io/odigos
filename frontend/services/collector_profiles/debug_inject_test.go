package collectorprofiles

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/services/common"
)

func TestInjectDebugSample(t *testing.T) {
	s := NewProfileStore(10, 60, 5*1024*1024, 0)
	err := s.InjectDebugSample("default", string(k8sconsts.WorkloadKindDeployment), "frontend")
	if err != nil {
		t.Fatal(err)
	}
	key := SourceKeyFromSourceID(common.SourceID{
		Namespace: "default",
		Kind:      k8sconsts.WorkloadKindDeployment,
		Name:      "frontend",
	})
	if data := s.GetProfileData(key); len(data) == 0 {
		t.Fatal("expected profile data after inject")
	}
}
