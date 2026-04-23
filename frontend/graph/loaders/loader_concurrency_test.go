package loaders

import (
	"context"
	"sync"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

// TestSetFiltersConcurrentWithNamespaceReads exercises the real concurrent
// pattern: gqlgen fans out namespace field resolution, and each goroutine may
// call SetFilters(ctx, nil) while others read GetWorkloadIdsInNamespace.
// sync.Once inside SetFilters ensures only one goroutine performs the work;
// atomic.Pointer snapshot ensures readers are lock-free.
func TestSetFiltersConcurrentWithNamespaceReads(t *testing.T) {
	ctx := context.Background()

	l := &Loaders{
		odigosConfiguration: &common.OdigosConfiguration{},
		workloadFilter: &WorkloadFilter{
			ClusterWide:       &WorkloadFilterClusterWide{},
			IgnoredNamespaces: map[string]struct{}{},
		},
		sourcesFetched: true,
		workloadSources: map[model.K8sWorkloadID]*odigosv1.Source{
			{Namespace: "default", Kind: model.K8sResourceKindDeployment, Name: "svc-a"}: {Spec: odigosv1.SourceSpec{MatchWorkloadNameAsRegex: true}},
			{Namespace: "default", Kind: model.K8sResourceKindDeployment, Name: "svc-b"}: {Spec: odigosv1.SourceSpec{MatchWorkloadNameAsRegex: true}},
			{Namespace: "prod", Kind: model.K8sResourceKindDaemonSet, Name: "agent"}:     {Spec: odigosv1.SourceSpec{MatchWorkloadNameAsRegex: true}},
		},
		workloadManifestsFetched: true,
		workloadManifests:        map[model.K8sWorkloadID]*computed.CachedWorkloadManifest{},
	}
	l.configOnce.Do(func() {})

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 400 {
				if err := l.SetFilters(ctx, nil); err != nil {
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

	ids := l.GetWorkloadIdsInNamespace("default")
	if len(ids) != 2 {
		t.Errorf("expected 2 workloads in default namespace, got %d", len(ids))
	}
}

func TestGetWorkloadIdsLockFree(t *testing.T) {
	l := &Loaders{}

	if ids := l.GetWorkloadIds(); ids != nil {
		t.Fatalf("expected nil before any snapshot, got %v", ids)
	}

	l.publishSnapshot([]model.K8sWorkloadID{
		{Namespace: "ns1", Kind: model.K8sResourceKindDeployment, Name: "a"},
		{Namespace: "ns2", Kind: model.K8sResourceKindStatefulSet, Name: "b"},
	})

	ids := l.GetWorkloadIds()
	if len(ids) != 2 {
		t.Fatalf("expected 2 workload ids, got %d", len(ids))
	}
	ns1 := l.GetWorkloadIdsInNamespace("ns1")
	if len(ns1) != 1 || ns1[0].Name != "a" {
		t.Fatalf("unexpected ns1 workloads: %v", ns1)
	}

	snap := l.workloadSnap.Load()
	if _, ok := snap.workloadIdsMap[k8sconsts.PodWorkload{
		Namespace: "ns2", Kind: k8sconsts.WorkloadKindStatefulSet, Name: "b",
	}]; !ok {
		t.Fatal("expected workloadIdsMap to contain ns2/b")
	}
}
