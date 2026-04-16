package loaders

import (
	"context"
	"sync"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func TestSetFiltersConcurrentWithNamespaceReads(t *testing.T) {
	ctx := context.Background()
	marked := true

	l := &Loaders{
		odigosConfiguration: &common.OdigosConfiguration{},
		workloadFilter: &WorkloadFilter{
			ClusterWide:       &WorkloadFilterClusterWide{},
			IgnoredNamespaces: map[string]struct{}{},
		},
		instrumentationConfigsFetched: true,
		instrumentationConfigs: map[model.K8sWorkloadID]*odigosv1.InstrumentationConfig{
			{Namespace: "default", Kind: model.K8sResourceKindDeployment, Name: "svc-a"}: &odigosv1.InstrumentationConfig{},
			{Namespace: "default", Kind: model.K8sResourceKindDeployment, Name: "svc-b"}: &odigosv1.InstrumentationConfig{},
			{Namespace: "prod", Kind: model.K8sResourceKindDaemonSet, Name: "agent"}:     &odigosv1.InstrumentationConfig{},
		},
		workloadIdsMap: make(map[k8sconsts.PodWorkload]struct{}),
		nsToWorkloadIds: map[string][]model.K8sWorkloadID{
			"default": []model.K8sWorkloadID{},
		},
	}

	filter := &model.WorkloadFilter{MarkedForInstrumentation: &marked}

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 400 {
				if err := l.SetFilters(ctx, filter); err != nil {
					t.Errorf("SetFilters failed: %v", err)
					return
				}
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 400 {
				_ = l.GetWorkloadIdsInNamespace("default")
			}
		}()
	}

	wg.Wait()
}
