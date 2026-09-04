package insights

import (
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
)

// gqlEnum is the shape gqlgen generates for every GraphQL enum.
type gqlEnum interface {
	~string
	IsValid() bool
}

// The insights enum converters are unchecked casts between the REST vocabulary
// and the independently generated gqlgen enums, so the two vocabularies have to
// stay character-for-character identical. Nothing in codegen enforces that: a
// value that diverges is only noticed when gqlgen fails to marshal the response
// at request time. Driving every generated value through the bridge pins it.

func assertEnumBridge[M gqlEnum, R ~string](t *testing.T, values []M, toModel func(R) M, fromModel func(M) R) {
	t.Helper()

	for _, value := range values {
		assert.True(t, value.IsValid(), "%q is not a valid GraphQL enum value", value)
		rest := fromModel(value)
		assert.Equal(t, string(value), string(rest), "REST and GraphQL must use the same wire value")
		assert.Equal(t, value, toModel(rest))
	}
}

func assertEnumToModel[M gqlEnum, R ~string](t *testing.T, values []M, toModel func(R) M) {
	t.Helper()

	for _, value := range values {
		assert.True(t, value.IsValid(), "%q is not a valid GraphQL enum value", value)
		assert.Equal(t, value, toModel(R(value)))
	}
}

func TestEnumBridgesKeepBothVocabulariesInSync(t *testing.T) {
	t.Run("transaction kind", func(t *testing.T) {
		assertEnumBridge(t, model.AllInsightsTransactionKind, TransactionKindToModel, TransactionKindFromModel)
	})
	t.Run("deviation class", func(t *testing.T) {
		assertEnumBridge(t, model.AllInsightsDeviationClass, DeviationClassToModel, DeviationClassFromModel)
	})
	t.Run("policy scope", func(t *testing.T) {
		assertEnumBridge(t, model.AllInsightsPolicyScope, PolicyScopeToModel, PolicyScopeFromModel)
	})
	t.Run("rule mode", func(t *testing.T) {
		assertEnumBridge(t, model.AllInsightsRuleMode, RuleModeToModel, RuleModeFromModel)
	})
	t.Run("learning mode", func(t *testing.T) {
		assertEnumBridge(t, model.AllInsightsLearningMode, LearningModeToModel, LearningModeFromModel)
	})
	t.Run("learning condition type", func(t *testing.T) {
		assertEnumBridge(t, model.AllInsightsLearningConditionType, LearningConditionTypeToModel, LearningConditionTypeFromModel)
	})
	t.Run("sample reason", func(t *testing.T) {
		assertEnumBridge(t, model.AllInsightsSampleReason, SampleReasonToModel, SampleReasonFromModel)
	})

	t.Run("severity", func(t *testing.T) {
		assertEnumToModel(t, model.AllInsightsSeverity, SeverityToModel)
	})
	t.Run("finding kind", func(t *testing.T) {
		assertEnumToModel(t, model.AllInsightsFindingKind, FindingKindToModel)
	})
	t.Run("anomaly status", func(t *testing.T) {
		assertEnumToModel(t, model.AllInsightsAnomalyStatus, AnomalyStatusToModel)
	})
	t.Run("violation status", func(t *testing.T) {
		assertEnumToModel(t, model.AllInsightsViolationStatus, ViolationStatusToModel)
	})
	t.Run("storage health status", func(t *testing.T) {
		assertEnumToModel(t, model.AllInsightsStorageHealthStatus, StorageHealthStatusToModel)
	})
	t.Run("storage disk status", func(t *testing.T) {
		assertEnumToModel(t, model.AllInsightsStorageDiskStatus, StorageDiskStatusToModel)
	})
}

// Resolution enums only travel GraphQL -> REST.
func TestResolutionEnumsKeepTheirWireValues(t *testing.T) {
	for _, resolution := range model.AllInsightsAnomalyResolution {
		assert.Equal(t, string(resolution), string(AnomalyResolutionFromModel(resolution)))
	}
	for _, resolution := range model.AllInsightsBulkResolution {
		assert.Equal(t, string(resolution), string(BulkResolutionFromModel(resolution)))
	}
}

// Storage tags evidence samples as anomaly:<signature> / guardrail:<rule>:<offending>,
// while the GraphQL enum only has the two collapsed tokens. Leaving a raw tag in
// place would make gqlgen reject the whole observation.
func TestSampleReasonToModelCollapsesStorageTags(t *testing.T) {
	tests := []struct {
		name   string
		reason SampleReason
		want   model.InsightsSampleReason
	}{
		{
			name:   "example",
			reason: SampleReasonExample,
			want:   model.InsightsSampleReasonExample,
		},
		{
			name:   "anomaly evidence tag",
			reason: AnomalyEvidenceSampleReason("D2_egress|db:5432"),
			want:   model.InsightsSampleReasonAnomalyEvidence,
		},
		{
			name:   "anomaly evidence tag with an empty signature",
			reason: AnomalyEvidenceSampleReason(""),
			want:   model.InsightsSampleReasonAnomalyEvidence,
		},
		{
			name:   "anomaly evidence token",
			reason: SampleReason("anomaly_evidence"),
			want:   model.InsightsSampleReasonAnomalyEvidence,
		},
		{
			name:   "guardrail evidence tag",
			reason: SampleReason("guardrail:allowed_egress:db:5432"),
			want:   model.InsightsSampleReasonGuardrailEvidence,
		},
		{
			name:   "guardrail evidence token",
			reason: SampleReason("guardrail_evidence"),
			want:   model.InsightsSampleReasonGuardrailEvidence,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SampleReasonToModel(tt.reason)
			assert.Equal(t, tt.want, got)
			assert.True(t, got.IsValid())
		})
	}
}

// The prefixes are the only thing separating the two evidence families, so a
// reason that merely mentions one of them must not be collapsed.
func TestSampleReasonToModelPassesThroughUnknownReasons(t *testing.T) {
	for _, reason := range []SampleReason{"", "manual", "example_anomaly:sig", "not-anomaly:sig"} {
		assert.Equal(t, model.InsightsSampleReason(reason), SampleReasonToModel(reason))
	}
}

func TestOptionalEnumConvertersPreserveAbsentValues(t *testing.T) {
	assert.Nil(t, TransactionKindPtrToModel(nil))
	assert.Nil(t, TransactionKindPtrFromModel(nil))
	assert.Nil(t, SampleReasonPtrFromModel(nil))
	assert.Nil(t, FindingKindPtrFromModel(nil))
	assert.Nil(t, RuleModePtrFromModel(nil))

	kind := TransactionKind("CRON")
	modelKind := model.InsightsTransactionKindCron
	assert.Equal(t, &modelKind, TransactionKindPtrToModel(&kind))
	assert.Equal(t, &kind, TransactionKindPtrFromModel(&modelKind))

	reason := model.InsightsSampleReasonGuardrailEvidence
	assert.Equal(t, SampleReason("guardrail_evidence"), *SampleReasonPtrFromModel(&reason))

	findingKind := model.InsightsFindingKind(model.AllInsightsFindingKind[0])
	assert.Equal(t, FindingKind(findingKind), *FindingKindPtrFromModel(&findingKind))

	mode := model.InsightsRuleMode(model.AllInsightsRuleMode[0])
	assert.Equal(t, RuleMode(mode), *RuleModePtrFromModel(&mode))
}
