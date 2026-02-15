package rollout

import (
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testLogger = logr.Discard()

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
	now := metav1.Now()
	ic.Status.RollbackOccurred = true
	ic.Spec.RecoveredFromRollbackAt = &now

	changed := recoverFromRollback(ic, testLogger)

	assert.True(t, changed, "expected recovery to be applied")
	assert.False(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to be cleared")
	assert.Equal(t, now.Time.Format(time.RFC3339), ic.Annotations[k8sconsts.RollbackRecoveryAtAnnotation])
}

func Test_recoverFromRollback_SecondRecovery_UpdatesAnnotation(t *testing.T) {
	ic := newRecoveryIC()
	oldTime := metav1.NewTime(time.Now().Add(-10 * time.Minute))
	newTime := metav1.Now()
	ic.Status.RollbackOccurred = true
	ic.Spec.RecoveredFromRollbackAt = &newTime
	ic.Annotations = map[string]string{
		k8sconsts.RollbackRecoveryAtAnnotation: oldTime.Time.Format(time.RFC3339),
	}

	changed := recoverFromRollback(ic, testLogger)

	assert.True(t, changed, "expected recovery to be applied")
	assert.False(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to be cleared")
	assert.Equal(t, newTime.Time.Format(time.RFC3339), ic.Annotations[k8sconsts.RollbackRecoveryAtAnnotation])
}

func Test_recoverFromRollback_AnnotationMatchesSpec_NoChange(t *testing.T) {
	ic := newRecoveryIC()
	now := metav1.Now()
	ic.Status.RollbackOccurred = true
	ic.Spec.RecoveredFromRollbackAt = &now
	ic.Annotations = map[string]string{
		k8sconsts.RollbackRecoveryAtAnnotation: now.Time.Format(time.RFC3339),
	}

	changed := recoverFromRollback(ic, testLogger)

	assert.False(t, changed, "expected no change — already acknowledged")
	assert.True(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to remain true")
}

func Test_recoverFromRollback_RollbackNotOccurred_NoChange(t *testing.T) {
	ic := newRecoveryIC()
	now := metav1.Now()
	ic.Status.RollbackOccurred = false
	ic.Spec.RecoveredFromRollbackAt = &now

	changed := recoverFromRollback(ic, testLogger)

	assert.False(t, changed, "expected no change — rollback did not occur")
}

func Test_recoverFromRollback_SpecNil_NoChange(t *testing.T) {
	ic := newRecoveryIC()
	ic.Status.RollbackOccurred = true
	ic.Spec.RecoveredFromRollbackAt = nil

	changed := recoverFromRollback(ic, testLogger)

	assert.False(t, changed, "expected no change — no recovery requested")
	assert.True(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to remain true")
}

func Test_recoverFromRollback_MalformedAnnotation_SkipsRecovery(t *testing.T) {
	ic := newRecoveryIC()
	now := metav1.Now()
	ic.Status.RollbackOccurred = true
	ic.Spec.RecoveredFromRollbackAt = &now
	ic.Annotations = map[string]string{
		k8sconsts.RollbackRecoveryAtAnnotation: "not-a-valid-timestamp",
	}

	changed := recoverFromRollback(ic, testLogger)

	assert.False(t, changed, "expected no change — malformed annotation should skip recovery")
	assert.True(t, ic.Status.RollbackOccurred, "expected RollbackOccurred to remain true")
	assert.Equal(t, "not-a-valid-timestamp", ic.Annotations[k8sconsts.RollbackRecoveryAtAnnotation],
		"expected malformed annotation to be left unchanged")
}

func Test_recoverFromRollback_NilAnnotationsMap_InitializesAndSets(t *testing.T) {
	ic := newRecoveryIC()
	ic.Annotations = nil
	now := metav1.Now()
	ic.Status.RollbackOccurred = true
	ic.Spec.RecoveredFromRollbackAt = &now

	changed := recoverFromRollback(ic, testLogger)

	assert.True(t, changed, "expected recovery to be applied")
	assert.NotNil(t, ic.Annotations, "expected Annotations map to be initialized")
	assert.Equal(t, now.Time.Format(time.RFC3339), ic.Annotations[k8sconsts.RollbackRecoveryAtAnnotation])
}
