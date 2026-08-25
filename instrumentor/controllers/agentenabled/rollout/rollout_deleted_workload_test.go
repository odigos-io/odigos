package rollout_test

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A workload that is deleted while its rollout is still in flight can never report the rollout as
// finished, so the slot it holds has to be released when the workload is gone. Otherwise every
// deletion permanently shrinks the pool and, once maxConcurrentRollouts deletions have happened,
// no workload is ever instrumented again until the instrumentor restarts.
func Test_Rollout_WorkloadDeletedMidRollout_ReleasesConcurrencySlot(t *testing.T) {
	for _, tc := range []struct {
		name string
		// the InstrumentationConfig is owned by the workload, so it is usually garbage collected
		// with it, but the reconcile can also still see it from the cache
		icGarbageCollected bool
	}{
		{name: "instrumentation config already garbage collected", icGarbageCollected: true},
		{name: "instrumentation config still cached", icGarbageCollected: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestSetup()
			setConfigConcurrentRolloutLimit(s.conf, 1)
			deployment := testutil.NewMockTestDeployment(s.ns, "deleted-deployment")
			ic := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(deployment))
			pw := k8sconsts.PodWorkload{Name: deployment.Name, Namespace: deployment.Namespace, Kind: k8sconsts.WorkloadKindDeployment}

			limiter := newRolloutConcurrencyLimiterActive()

			// The workload takes the only slot and is restarted.
			_, err := rollout.Do(s.ctx, s.newFakeClient(deployment), ic, pw, s.conf, s.distroProvider, limiter)
			require.NoError(t, err)
			require.Equal(t, 1, limiter.InFlightCount(), "expected the rollout to take the slot")

			// The workload is deleted before the rollout completes. reconcileWorkload passes a nil
			// InstrumentationConfig once it is gone too.
			reconciledIC := ic
			if tc.icGarbageCollected {
				reconciledIC = nil
			}
			_, err = rollout.Do(s.ctx, s.newFakeClient(), reconciledIC, pw, s.conf, s.distroProvider, limiter)
			require.NoError(t, err)

			assert.Equal(t, 0, limiter.InFlightCount(), "the deleted workload leaked its rollout slot")

			// The freed slot is usable by the next workload.
			next := testutil.NewMockTestDeployment(s.ns, "next-deployment")
			nextIC := mockICRolloutRequiredDistro(testutil.NewMockInstrumentationConfig(next))
			nextPW := k8sconsts.PodWorkload{Name: next.Name, Namespace: next.Namespace, Kind: k8sconsts.WorkloadKindDeployment}
			nextClient := s.newFakeClient(next)

			_, err = rollout.Do(s.ctx, nextClient, nextIC, nextPW, s.conf, s.distroProvider, limiter)
			require.NoError(t, err)
			assertWorkloadRestarted(t, s.ctx, nextClient, nextPW)
		})
	}
}
