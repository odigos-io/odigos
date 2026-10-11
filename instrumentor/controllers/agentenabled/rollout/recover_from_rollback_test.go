package rollout

import (
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newRecoveryIC() *odigosv1alpha1.InstrumentationConfig {
	return &odigosv1alpha1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ic",
			Namespace: "default",
		},
	}
}

func Test_recoverFromRollback_FirstRecovery_ClearsRollbackAndSetsAnnotation(t *testing.T) {
	ic := newRecoveryIC()
	now := time.Now().Format(time.RFC3339)
	ic.Status.RollbackOccurred = true
	ic.Annotations = map[string]string{
		k8sconsts.RollbackRecoveryAtAnnotation: now,
	}

	changed := recoverFromRollback(ic)

	assert.True(t, changed, "expected recovery to be applied")
	assert.True(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to remain true — caller clears it")
	assert.Equal(t, now, ic.Annotations[k8sconsts.RollbackRecoveryProcessedAtAnnotation])
}

func Test_recoverFromRollback_SecondRecovery_UpdatesAnnotation(t *testing.T) {
	ic := newRecoveryIC()
	oldTime := time.Now().Add(-10 * time.Minute).Format(time.RFC3339)
	newTime := time.Now().Format(time.RFC3339)
	ic.Status.RollbackOccurred = true
	ic.Annotations = map[string]string{
		k8sconsts.RollbackRecoveryAtAnnotation:          newTime,
		k8sconsts.RollbackRecoveryProcessedAtAnnotation: oldTime,
	}

	changed := recoverFromRollback(ic)

	assert.True(t, changed, "expected recovery to be applied")
	assert.True(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to remain true — caller clears it")
	assert.Equal(t, newTime, ic.Annotations[k8sconsts.RollbackRecoveryProcessedAtAnnotation])
}

func Test_recoverFromRollback_ProcessedMatchesDesired_NoChange(t *testing.T) {
	ic := newRecoveryIC()
	now := time.Now().Format(time.RFC3339)
	ic.Status.RollbackOccurred = true
	ic.Annotations = map[string]string{
		k8sconsts.RollbackRecoveryAtAnnotation:          now,
		k8sconsts.RollbackRecoveryProcessedAtAnnotation: now,
	}

	changed := recoverFromRollback(ic)

	assert.False(t, changed, "expected no change — already acknowledged")
	assert.True(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to remain true")
}

func Test_recoverFromRollback_RollbackNotOccurred_NoChange(t *testing.T) {
	ic := newRecoveryIC()
	ic.Status.RollbackOccurred = false
	ic.Annotations = map[string]string{
		k8sconsts.RollbackRecoveryAtAnnotation: time.Now().Format(time.RFC3339),
	}

	changed := recoverFromRollback(ic)

	assert.False(t, changed, "expected no change — rollback did not occur")
}

func Test_recoverFromRollback_NoRecoveryAnnotation_NoChange(t *testing.T) {
	ic := newRecoveryIC()
	ic.Status.RollbackOccurred = true

	changed := recoverFromRollback(ic)

	assert.False(t, changed, "expected no change — no recovery requested")
	assert.True(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to remain true")
}

func Test_recoverFromRollback_MalformedProcessedAnnotation_OverwritesAndRecovers(t *testing.T) {
	ic := newRecoveryIC()
	now := time.Now().Format(time.RFC3339)
	ic.Status.RollbackOccurred = true
	ic.Annotations = map[string]string{
		k8sconsts.RollbackRecoveryAtAnnotation:          now,
		k8sconsts.RollbackRecoveryProcessedAtAnnotation: "not-a-valid-timestamp",
	}

	changed := recoverFromRollback(ic)

	assert.True(t, changed, "expected recovery — processed differs from desired")
	assert.True(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to remain true — caller clears it")
	assert.Equal(t, now, ic.Annotations[k8sconsts.RollbackRecoveryProcessedAtAnnotation],
		"expected processed annotation to be overwritten with the desired value")
}

func Test_recoverFromRollback_NilAnnotationsMap_InitializesAndSets(t *testing.T) {
	ic := newRecoveryIC()
	now := time.Now().Format(time.RFC3339)
	ic.Status.RollbackOccurred = true
	ic.Annotations = map[string]string{
		k8sconsts.RollbackRecoveryAtAnnotation: now,
	}

	changed := recoverFromRollback(ic)

	assert.True(t, changed, "expected recovery to be applied")
	assert.Equal(t, now, ic.Annotations[k8sconsts.RollbackRecoveryProcessedAtAnnotation])
}
