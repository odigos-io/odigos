package source

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

const (
	// The workload Source CR and the namespace Source CR are deliberately given different
	// names and different namespaces so that a message built from the wrong one is visible.
	workloadSourceName       = "source-deployment-frontend"
	workloadSourceNamespace  = "workload-source-ns"
	namespaceSourceName      = "source-namespace-checkout"
	namespaceSourceNamespace = "namespace-source-ns"
)

func enabledSource(name, namespace string) *odigosv1.Source {
	return &odigosv1.Source{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       odigosv1.SourceSpec{DisableInstrumentation: false},
	}
}

func disabledSource(name, namespace string) *odigosv1.Source {
	src := enabledSource(name, namespace)
	src.Spec.DisableInstrumentation = true
	return src
}

func asTerminating(src *odigosv1.Source) *odigosv1.Source {
	deletedAt := metav1.NewTime(time.Date(2026, time.March, 4, 12, 0, 0, 0, time.UTC))
	src.DeletionTimestamp = &deletedAt
	return src
}

func labeled(src *odigosv1.Source, labels map[string]string) *odigosv1.Source {
	src.Labels = labels
	return src
}

func workloadEnabled() *odigosv1.Source {
	return enabledSource(workloadSourceName, workloadSourceNamespace)
}

func workloadDisabled() *odigosv1.Source {
	return disabledSource(workloadSourceName, workloadSourceNamespace)
}

func namespaceEnabled() *odigosv1.Source {
	return enabledSource(namespaceSourceName, namespaceSourceNamespace)
}

func namespaceDisabled() *odigosv1.Source {
	return disabledSource(namespaceSourceName, namespaceSourceNamespace)
}

// TestIsObjectInstrumentedBySource_Precedence walks the full workload-source x namespace-source
// matrix. This is the decision that determines whether a workload is instrumented at all, and the
// documented precedence is that an active workload Source wins over the namespace Source in both
// directions - it can opt a workload in when the namespace is not marked, and opt it out when the
// namespace is marked.
func TestIsObjectInstrumentedBySource_Precedence(t *testing.T) {
	tests := []struct {
		name                 string
		workload             *odigosv1.Source
		namespace            *odigosv1.Source
		expectedInstrumented bool
		expectedStatus       metav1.ConditionStatus
		expectedReason       odigosv1.MarkedForInstrumentationReason
	}{
		{
			name:                 "no source at all",
			expectedInstrumented: false,
			expectedStatus:       metav1.ConditionFalse,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonNoSource,
		},
		{
			name:                 "workload source only",
			workload:             workloadEnabled(),
			expectedInstrumented: true,
			expectedStatus:       metav1.ConditionTrue,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonWorkloadSource,
		},
		{
			name:                 "disabled workload source only",
			workload:             workloadDisabled(),
			expectedInstrumented: false,
			expectedStatus:       metav1.ConditionFalse,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonWorkloadSourceDisabled,
		},
		{
			name:                 "namespace source only",
			namespace:            namespaceEnabled(),
			expectedInstrumented: true,
			expectedStatus:       metav1.ConditionTrue,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonNamespaceSource,
		},
		{
			name:                 "disabled namespace source only",
			namespace:            namespaceDisabled(),
			expectedInstrumented: false,
			expectedStatus:       metav1.ConditionFalse,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonNoSource,
		},
		{
			name:                 "workload source opts in a namespace that is not marked",
			workload:             workloadEnabled(),
			namespace:            nil,
			expectedInstrumented: true,
			expectedStatus:       metav1.ConditionTrue,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonWorkloadSource,
		},
		{
			name:                 "workload source opts out of a marked namespace",
			workload:             workloadDisabled(),
			namespace:            namespaceEnabled(),
			expectedInstrumented: false,
			expectedStatus:       metav1.ConditionFalse,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonWorkloadSourceDisabled,
		},
		{
			name:                 "active workload source wins over a disabled namespace source",
			workload:             workloadEnabled(),
			namespace:            namespaceDisabled(),
			expectedInstrumented: true,
			expectedStatus:       metav1.ConditionTrue,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonWorkloadSource,
		},
		{
			name:                 "active workload source wins over an active namespace source",
			workload:             workloadEnabled(),
			namespace:            namespaceEnabled(),
			expectedInstrumented: true,
			expectedStatus:       metav1.ConditionTrue,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonWorkloadSource,
		},
		{
			name:                 "terminating workload source falls back to the namespace source",
			workload:             asTerminating(workloadEnabled()),
			namespace:            namespaceEnabled(),
			expectedInstrumented: true,
			expectedStatus:       metav1.ConditionTrue,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonNamespaceSource,
		},
		{
			name:                 "terminating disabled workload source no longer excludes the workload",
			workload:             asTerminating(workloadDisabled()),
			namespace:            namespaceEnabled(),
			expectedInstrumented: true,
			expectedStatus:       metav1.ConditionTrue,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonNamespaceSource,
		},
		{
			name:                 "terminating workload source with no namespace source",
			workload:             asTerminating(workloadEnabled()),
			expectedInstrumented: false,
			expectedStatus:       metav1.ConditionFalse,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonNoSource,
		},
		{
			name:                 "terminating namespace source does not mark the workload",
			namespace:            asTerminating(namespaceEnabled()),
			expectedInstrumented: false,
			expectedStatus:       metav1.ConditionFalse,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonNoSource,
		},
		{
			name:                 "both sources terminating",
			workload:             asTerminating(workloadEnabled()),
			namespace:            asTerminating(namespaceEnabled()),
			expectedInstrumented: false,
			expectedStatus:       metav1.ConditionFalse,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonNoSource,
		},
		{
			name:                 "disabled workload source and disabled namespace source",
			workload:             workloadDisabled(),
			namespace:            namespaceDisabled(),
			expectedInstrumented: false,
			expectedStatus:       metav1.ConditionFalse,
			expectedReason:       odigosv1.MarkedForInstrumentationReasonWorkloadSourceDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sources := &odigosv1.WorkloadSources{Workload: tt.workload, Namespace: tt.namespace}

			instrumented, condition, err := IsObjectInstrumentedBySource(context.Background(), sources, nil)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedInstrumented, instrumented)
			assert.Equal(t, tt.expectedStatus, condition.Status)
			assert.Equal(t, string(tt.expectedReason), condition.Reason)
			assert.Equal(t, odigosv1.MarkedForInstrumentationStatusConditionType, condition.Type)
			assert.NotEmpty(t, condition.Message)
		})
	}
}

// A terminating Source is one whose DeletionTimestamp is set. Only a non-zero timestamp counts,
// which is what distinguishes "being deleted" from "never deleted".
func TestIsObjectInstrumentedBySource_AZeroDeletionTimestampIsNotTerminating(t *testing.T) {
	src := workloadEnabled()
	src.DeletionTimestamp = &metav1.Time{}

	instrumented, condition, err := IsObjectInstrumentedBySource(context.Background(),
		&odigosv1.WorkloadSources{Workload: src}, nil)

	require.NoError(t, err)
	assert.True(t, instrumented)
	assert.Equal(t, string(odigosv1.MarkedForInstrumentationReasonWorkloadSource), condition.Reason)
}

// The caller passes the result of odigosv1.GetSources straight through, and that returns a nil
// WorkloadSources together with its error. Looking at the sources before the error would panic.
func TestIsObjectInstrumentedBySource_ReturnsTheLookupErrorWithoutTouchingNilSources(t *testing.T) {
	lookupErr := errors.New("etcd is unreachable")

	instrumented, condition, err := IsObjectInstrumentedBySource(context.Background(), nil, lookupErr)

	require.ErrorIs(t, err, lookupErr)
	assert.False(t, instrumented)
	assert.Equal(t, odigosv1.MarkedForInstrumentationStatusConditionType, condition.Type)
	assert.Equal(t, metav1.ConditionUnknown, condition.Status)
	assert.Equal(t, string(odigosv1.MarkedForInstrumentationReasonError), condition.Reason)
	assert.Contains(t, condition.Message, lookupErr.Error())
}

// An error must not be masked by a Source that would otherwise mark the workload: the answer is
// unknown, not "instrumented".
func TestIsObjectInstrumentedBySource_ErrorWinsOverAnActiveSource(t *testing.T) {
	sources := &odigosv1.WorkloadSources{Workload: workloadEnabled(), Namespace: namespaceEnabled()}

	instrumented, condition, err := IsObjectInstrumentedBySource(context.Background(), sources,
		errors.New("boom"))

	require.Error(t, err)
	assert.False(t, instrumented)
	assert.Equal(t, metav1.ConditionUnknown, condition.Status)
}

// The condition message is what a user reads to find out which CR made the decision, so the
// workload branch has to name the workload CR and the namespace branch the namespace CR.
func TestIsObjectInstrumentedBySource_MessageNamesTheDecidingSourceCR(t *testing.T) {
	tests := []struct {
		name            string
		sources         *odigosv1.WorkloadSources
		expectedMessage string
	}{
		{
			name:    "marked by the workload source",
			sources: &odigosv1.WorkloadSources{Workload: workloadEnabled(), Namespace: namespaceEnabled()},
			expectedMessage: "workload marked for automatic instrumentation by workload source CR " +
				"'source-deployment-frontend' in namespace 'workload-source-ns'",
		},
		{
			name:    "excluded by the workload source",
			sources: &odigosv1.WorkloadSources{Workload: workloadDisabled(), Namespace: namespaceEnabled()},
			expectedMessage: "workload marked to disable instrumentation by workload source CR " +
				"'source-deployment-frontend' in namespace 'workload-source-ns'",
		},
		{
			name:    "marked by the namespace source",
			sources: &odigosv1.WorkloadSources{Namespace: namespaceEnabled()},
			expectedMessage: "workload marked for automatic instrumentation by namespace source CR " +
				"'source-namespace-checkout' in namespace 'namespace-source-ns'",
		},
		{
			name:            "not marked by any source",
			sources:         &odigosv1.WorkloadSources{},
			expectedMessage: "workload not marked for automatic instrumentation by any source CR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, condition, err := IsObjectInstrumentedBySource(context.Background(), tt.sources, nil)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedMessage, condition.Message)
		})
	}
}

func TestIsDataStreamLabel(t *testing.T) {
	tests := []struct {
		labelKey string
		expected bool
	}{
		{labelKey: "odigos.io/data-stream-payments", expected: true},
		{labelKey: "odigos.io/data-stream-", expected: true},
		{labelKey: "odigos.io/data-stream", expected: false},
		{labelKey: "odigos.io/datastream-payments", expected: false},
		{labelKey: "app.kubernetes.io/odigos.io/data-stream-payments", expected: false},
		{labelKey: "odigos.io/source-name", expected: false},
		{labelKey: "", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.labelKey, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsDataStreamLabel(tt.labelKey))
		})
	}
}

// The prefix is the contract between whoever writes the label on a Source and every reader of the
// data stream membership, so it is pinned to its literal rather than asserted through the constant.
func TestDataStreamLabelPrefixIsStable(t *testing.T) {
	assert.Equal(t, "odigos.io/data-stream-", k8sconsts.SourceDataStreamLabelPrefix)
}

func TestGetSourceDataStreamsLabels(t *testing.T) {
	tests := []struct {
		name     string
		source   *odigosv1.Source
		expected map[string]string
	}{
		{
			name:     "nil source",
			source:   nil,
			expected: map[string]string{},
		},
		{
			name:     "source without labels",
			source:   workloadEnabled(),
			expected: map[string]string{},
		},
		{
			name: "only data stream labels are extracted",
			source: labeled(workloadEnabled(), map[string]string{
				"odigos.io/data-stream-payments": "true",
				"odigos.io/data-stream-checkout": "true",
				"odigos.io/workload-kind":        "Deployment",
				"app":                            "frontend",
			}),
			expected: map[string]string{
				"odigos.io/data-stream-payments": "true",
				"odigos.io/data-stream-checkout": "true",
			},
		},
		{
			name: "source with no data stream labels",
			source: labeled(workloadEnabled(), map[string]string{
				"odigos.io/workload-kind": "Deployment",
			}),
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, getSourceDataStreamsLabels(tt.source))
		})
	}
}

// Data stream membership decides which pipeline a workload's telemetry is routed to. The namespace
// Source provides the baseline and the workload Source may both add to it and override it, so the
// two sides carry different values for the shared key.
func TestCalculateDataStreamsLabels(t *testing.T) {
	tests := []struct {
		name     string
		workload *odigosv1.Source
		nsSource *odigosv1.Source
		expected map[string]string
	}{
		{
			name:     "no sources",
			expected: map[string]string{},
		},
		{
			name: "namespace labels only",
			nsSource: labeled(namespaceEnabled(), map[string]string{
				"odigos.io/data-stream-default": "namespace-value",
			}),
			expected: map[string]string{"odigos.io/data-stream-default": "namespace-value"},
		},
		{
			name: "workload labels only",
			workload: labeled(workloadEnabled(), map[string]string{
				"odigos.io/data-stream-payments": "workload-value",
			}),
			expected: map[string]string{"odigos.io/data-stream-payments": "workload-value"},
		},
		{
			name: "workload labels are merged on top of namespace labels",
			workload: labeled(workloadEnabled(), map[string]string{
				"odigos.io/data-stream-payments": "workload-value",
			}),
			nsSource: labeled(namespaceEnabled(), map[string]string{
				"odigos.io/data-stream-default": "namespace-value",
			}),
			expected: map[string]string{
				"odigos.io/data-stream-payments": "workload-value",
				"odigos.io/data-stream-default":  "namespace-value",
			},
		},
		{
			name: "the workload value wins for a shared data stream key",
			workload: labeled(workloadEnabled(), map[string]string{
				"odigos.io/data-stream-shared": "workload-value",
			}),
			nsSource: labeled(namespaceEnabled(), map[string]string{
				"odigos.io/data-stream-shared": "namespace-value",
			}),
			expected: map[string]string{"odigos.io/data-stream-shared": "workload-value"},
		},
		{
			name: "non data stream labels are dropped from both sides",
			workload: labeled(workloadEnabled(), map[string]string{
				"odigos.io/data-stream-payments": "workload-value",
				"odigos.io/workload-name":        "frontend",
			}),
			nsSource: labeled(namespaceEnabled(), map[string]string{
				"odigos.io/data-stream-default": "namespace-value",
				"kubernetes.io/metadata.name":   "app-ns",
			}),
			expected: map[string]string{
				"odigos.io/data-stream-payments": "workload-value",
				"odigos.io/data-stream-default":  "namespace-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sources := &odigosv1.WorkloadSources{Workload: tt.workload, Namespace: tt.nsSource}
			assert.Equal(t, tt.expected, CalculateDataStreamsLabels(sources))
		})
	}
}

// The merged map is handed to callers that set it on other objects, so it must not alias the
// labels of either Source.
func TestCalculateDataStreamsLabelsReturnsAFreshMap(t *testing.T) {
	workloadLabels := map[string]string{"odigos.io/data-stream-payments": "workload-value"}
	namespaceLabels := map[string]string{"odigos.io/data-stream-default": "namespace-value"}
	sources := &odigosv1.WorkloadSources{
		Workload:  labeled(workloadEnabled(), workloadLabels),
		Namespace: labeled(namespaceEnabled(), namespaceLabels),
	}

	merged := CalculateDataStreamsLabels(sources)
	merged["odigos.io/data-stream-payments"] = "mutated"
	merged["odigos.io/data-stream-extra"] = "added"

	assert.Equal(t, map[string]string{"odigos.io/data-stream-payments": "workload-value"}, workloadLabels)
	assert.Equal(t, map[string]string{"odigos.io/data-stream-default": "namespace-value"}, namespaceLabels)
}

// Whatever the filter is, the two functions have to agree: every key that survives the merge must
// be a data stream label, and every data stream label on either Source must survive it.
func TestCalculateDataStreamsLabelsKeepsExactlyTheDataStreamLabels(t *testing.T) {
	allLabels := map[string]string{
		"odigos.io/data-stream-payments": "true",
		"odigos.io/data-stream-checkout": "true",
		"odigos.io/data-stream-":         "true",
		"odigos.io/workload-kind":        "Deployment",
		"odigos.io/data-stream":          "true",
		"app":                            "frontend",
	}
	sources := &odigosv1.WorkloadSources{Workload: labeled(workloadEnabled(), allLabels)}

	merged := CalculateDataStreamsLabels(sources)

	for key := range merged {
		assert.True(t, IsDataStreamLabel(key), "merged label %q is not a data stream label", key)
	}
	for key := range allLabels {
		if IsDataStreamLabel(key) {
			assert.Contains(t, merged, key)
		} else {
			assert.NotContains(t, merged, key)
		}
	}
}
