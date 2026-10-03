package podsmanifestinjectionstatus

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// The reconciler is keyed by the instrumentation config name, which encodes the workload it
// belongs to. Resolving it to the wrong workload would write the status onto another workload.
func TestInstrumentationConfigControllerSyncsTheNamedWorkload(t *testing.T) {
	ctx := newInjectionTestContext()
	c := newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionDeployment("web", map[string]string{"app": "web"}),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
		withPodLabel(newInjectionPod(injectionNamespace, "web-1", injectionStaleHash), "app", "web"),
	)
	r := &InstrumentationConfigController{Client: c, PodsTracker: NewPodsTracker()}

	result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
		Namespace: injectionNamespace,
		Name:      workload.CalculateWorkloadRuntimeObjectName("web", k8sconsts.WorkloadKindDeployment),
	}})
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	requireInjectionStatus(t, syncedInstrumentationConfig(t, ctx, c, "web",
		k8sconsts.WorkloadKindDeployment), false, true, false)
}

func TestInstrumentationConfigControllerRejectsAnUnparsableName(t *testing.T) {
	ctx := newInjectionTestContext()
	c := newInjectionTestClient(t, newEffectiveConfigMap(""))
	r := &InstrumentationConfigController{Client: c, PodsTracker: NewPodsTracker()}

	_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
		Namespace: injectionNamespace, Name: "bogus-workload",
	}})
	assert.ErrorIs(t, err, workload.ErrKindNotSupported)
}
