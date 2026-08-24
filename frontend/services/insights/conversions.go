package insights

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/odigos-io/odigos/frontend/graph/model"
)

// LearningCondition camelCase wire form.
//
// The insights REST API is otherwise snake_case, but LearningCondition fields
// are camelCase on the wire (see openapi LearningCondition). Custom JSON
// methods keep that quirk here so the HTTP client and GraphQL resolvers stay
// unaware of it.

type learningConditionWire struct {
	Type               LearningConditionType `json:"type"`
	MinObservations    *int64                `json:"minObservations,omitempty"`
	MinDurationMinutes *int64                `json:"minDurationMinutes,omitempty"`
	StableObservations *int64                `json:"stableObservations,omitempty"`
	StableMinutes      *int64                `json:"stableMinutes,omitempty"`
}

func (c LearningCondition) MarshalJSON() ([]byte, error) {
	return json.Marshal(learningConditionWire{
		Type:               c.Type,
		MinObservations:    c.MinObservations,
		MinDurationMinutes: c.MinDurationMinutes,
		StableObservations: c.StableObservations,
		StableMinutes:      c.StableMinutes,
	})
}

func (c *LearningCondition) UnmarshalJSON(data []byte) error {
	var wire learningConditionWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	*c = LearningCondition{
		Type:               wire.Type,
		MinObservations:    wire.MinObservations,
		MinDurationMinutes: wire.MinDurationMinutes,
		StableObservations: wire.StableObservations,
		StableMinutes:      wire.StableMinutes,
	}
	return nil
}

// ID helpers — GraphQL exposes transaction / policy ids as ID (string); REST uses int64.
//
// These and the input converters further down report failures as ErrBadRequest:
// they only ever fail on caller-supplied GraphQL arguments, so resolvers surface
// them as invalid requests rather than as service errors.

func FormatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func ParseID(id string) (int64, error) {
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: id %q is not a number", ErrBadRequest, id)
	}
	return parsed, nil
}

func formatOptionalID(id *int64) *string {
	if id == nil {
		return nil
	}
	formatted := FormatID(*id)
	return &formatted
}

func parseOptionalID(id *string) (*int64, error) {
	if id == nil {
		return nil, nil
	}
	parsed, err := ParseID(*id)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// Free-form JSON fields are GraphQL strings.

func encodeJSONString(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode insights json field: %w", err)
	}
	return string(encoded), nil
}

func encodeOptionalJSONString(value any) (*string, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case json.RawMessage:
		if len(typed) == 0 {
			return nil, nil
		}
		encoded := string(typed)
		return &encoded, nil
	case map[string]int:
		if typed == nil {
			return nil, nil
		}
	case map[string][]string:
		if typed == nil {
			return nil, nil
		}
	}
	encoded, err := encodeJSONString(value)
	if err != nil {
		return nil, err
	}
	return &encoded, nil
}

func decodeJSONString[T any](encoded string) (T, error) {
	var value T
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return value, fmt.Errorf("%w: field is not valid JSON: %v", ErrBadRequest, err)
	}
	return value, nil
}

func decodeOptionalJSONString[T any](encoded *string) (T, error) {
	var zero T
	if encoded == nil {
		return zero, nil
	}
	return decodeJSONString[T](*encoded)
}

func int64ToInt(value int64) int {
	return int(value)
}

func int64PtrToIntPtr(value *int64) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

func intPtrToInt64Ptr(value *int) *int64 {
	if value == nil {
		return nil
	}
	converted := int64(*value)
	return &converted
}

func mapSlice[T any, U any](items []T, convert func(T) U) []U {
	if items == nil {
		return nil
	}
	out := make([]U, len(items))
	for i, item := range items {
		out[i] = convert(item)
	}
	return out
}

func mapSliceErr[T any, U any](items []T, convert func(T) (U, error)) ([]U, error) {
	if items == nil {
		return nil, nil
	}
	out := make([]U, len(items))
	for i, item := range items {
		converted, err := convert(item)
		if err != nil {
			return nil, err
		}
		out[i] = converted
	}
	return out, nil
}

// Enum conversions.

func TransactionKindToModel(kind TransactionKind) model.InsightsTransactionKind {
	return model.InsightsTransactionKind(kind)
}

func TransactionKindFromModel(kind model.InsightsTransactionKind) TransactionKind {
	return TransactionKind(kind)
}

func TransactionKindPtrFromModel(kind *model.InsightsTransactionKind) *TransactionKind {
	if kind == nil {
		return nil
	}
	converted := TransactionKindFromModel(*kind)
	return &converted
}

func TransactionKindPtrToModel(kind *TransactionKind) *model.InsightsTransactionKind {
	if kind == nil {
		return nil
	}
	converted := TransactionKindToModel(*kind)
	return &converted
}

func DeviationClassToModel(class DeviationClass) model.InsightsDeviationClass {
	return model.InsightsDeviationClass(class)
}

func DeviationClassFromModel(class model.InsightsDeviationClass) DeviationClass {
	return DeviationClass(class)
}

func SampleReasonToModel(reason SampleReason) model.InsightsSampleReason {
	s := string(reason)
	switch {
	case s == string(model.InsightsSampleReasonExample):
		return model.InsightsSampleReasonExample
	case s == string(model.InsightsSampleReasonAnomalyEvidence) || strings.HasPrefix(s, "anomaly:"):
		// Storage tags evidence as anomaly:<signature>; GraphQL keeps the legacy enum token.
		return model.InsightsSampleReasonAnomalyEvidence
	case s == string(model.InsightsSampleReasonGuardrailEvidence) || strings.HasPrefix(s, "guardrail:"):
		// Storage tags evidence as guardrail:<rule>:<offending>.
		return model.InsightsSampleReasonGuardrailEvidence
	default:
		return model.InsightsSampleReason(reason)
	}
}

func SampleReasonFromModel(reason model.InsightsSampleReason) SampleReason {
	return SampleReason(reason)
}

func SampleReasonPtrFromModel(reason *model.InsightsSampleReason) *SampleReason {
	if reason == nil {
		return nil
	}
	converted := SampleReasonFromModel(*reason)
	return &converted
}

func PolicyScopeToModel(scope PolicyScope) model.InsightsPolicyScope {
	return model.InsightsPolicyScope(scope)
}

func PolicyScopeFromModel(scope model.InsightsPolicyScope) PolicyScope {
	return PolicyScope(scope)
}

func SeverityToModel(severity Severity) model.InsightsSeverity {
	return model.InsightsSeverity(severity)
}

func FindingKindToModel(kind FindingKind) model.InsightsFindingKind {
	return model.InsightsFindingKind(kind)
}

func FindingKindPtrFromModel(kind *model.InsightsFindingKind) *FindingKind {
	if kind == nil {
		return nil
	}
	converted := FindingKind(*kind)
	return &converted
}

func AnomalyStatusToModel(status AnomalyStatus) model.InsightsAnomalyStatus {
	return model.InsightsAnomalyStatus(status)
}

func ViolationStatusToModel(status ViolationStatus) model.InsightsViolationStatus {
	return model.InsightsViolationStatus(status)
}

func RuleModeToModel(mode RuleMode) model.InsightsRuleMode {
	return model.InsightsRuleMode(mode)
}

func RuleModeFromModel(mode model.InsightsRuleMode) RuleMode {
	return RuleMode(mode)
}

func RuleModePtrFromModel(mode *model.InsightsRuleMode) *RuleMode {
	if mode == nil {
		return nil
	}
	converted := RuleModeFromModel(*mode)
	return &converted
}

func AnomalyResolutionFromModel(resolution model.InsightsAnomalyResolution) AnomalyResolution {
	return AnomalyResolution(resolution)
}

func BulkResolutionFromModel(resolution model.InsightsBulkResolution) BulkResolution {
	return BulkResolution(resolution)
}

func LearningModeToModel(mode LearningMode) model.InsightsLearningMode {
	return model.InsightsLearningMode(mode)
}

func LearningModeFromModel(mode model.InsightsLearningMode) LearningMode {
	return LearningMode(mode)
}

func LearningConditionTypeToModel(conditionType LearningConditionType) model.InsightsLearningConditionType {
	return model.InsightsLearningConditionType(conditionType)
}

func LearningConditionTypeFromModel(conditionType model.InsightsLearningConditionType) LearningConditionType {
	return LearningConditionType(conditionType)
}

func StorageHealthStatusToModel(status StorageHealthStatus) model.InsightsStorageHealthStatus {
	return model.InsightsStorageHealthStatus(status)
}

func StorageDiskStatusToModel(status StorageDiskStatus) model.InsightsStorageDiskStatus {
	return model.InsightsStorageDiskStatus(status)
}

// Response converters.

func ServiceStatToModel(stat ServiceStat) *model.InsightsServiceStat {
	return &model.InsightsServiceStat{
		Namespace:        stat.Namespace,
		Service:          stat.Service,
		TransactionCount: int64ToInt(stat.TransactionCount),
		Volume:           int64ToInt(stat.Volume),
		LastSeen:         stat.LastSeen,
	}
}

func ServiceStatsToModel(stats []ServiceStat) []*model.InsightsServiceStat {
	return mapSlice(stats, ServiceStatToModel)
}

func ServiceProfileToModel(profile ServiceProfile) *model.InsightsServiceProfile {
	return &model.InsightsServiceProfile{
		Namespace:    profile.Namespace,
		Service:      profile.Service,
		Callers:      stringSliceOrEmpty(profile.Callers),
		Callees:      stringSliceOrEmpty(profile.Callees),
		Egress:       stringSliceOrEmpty(profile.Egress),
		Transactions: stringSliceOrEmpty(profile.Transactions),
	}
}

func BlastRadiusNodeToModel(node BlastRadiusNode) *model.InsightsBlastRadiusNode {
	return &model.InsightsBlastRadiusNode{
		Namespace: node.Namespace,
		Service:   node.Service,
		IsVirtual: node.IsVirtual,
	}
}

func BlastRadiusEdgeToModel(edge BlastRadiusEdge) *model.InsightsBlastRadiusEdge {
	return &model.InsightsBlastRadiusEdge{
		ClientNamespace: edge.ClientNamespace,
		ClientService:   edge.ClientService,
		ServerNamespace: edge.ServerNamespace,
		ServerService:   edge.ServerService,
		ConnectionType:  edge.ConnectionType,
		RequestCount:    int64ToInt(edge.RequestCount),
		FailedCount:     int64ToInt(edge.FailedCount),
		LastSeen:        edge.LastSeen,
	}
}

func BlastRadiusSubgraphToModel(subgraph BlastRadiusSubgraph) *model.InsightsBlastRadiusSubgraph {
	return &model.InsightsBlastRadiusSubgraph{
		Root:  BlastRadiusNodeToModel(subgraph.Root),
		Nodes: mapSlice(subgraph.Nodes, BlastRadiusNodeToModel),
		Edges: mapSlice(subgraph.Edges, BlastRadiusEdgeToModel),
	}
}

func stringSliceOrEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func TransactionIdentityValueToModel(dim TransactionIdentityValue) *model.InsightsTransactionIdentityValue {
	return &model.InsightsTransactionIdentityValue{
		Key:   dim.Key,
		Value: dim.Value,
	}
}

func TransactionIdentityValuesToModel(dims []TransactionIdentityValue) []*model.InsightsTransactionIdentityValue {
	out := mapSlice(dims, TransactionIdentityValueToModel)
	if out == nil {
		return []*model.InsightsTransactionIdentityValue{}
	}
	return out
}

func TransactionStatToModel(stat TransactionStat) *model.InsightsTransactionStat {
	return &model.InsightsTransactionStat{
		ID:                 FormatID(stat.ID),
		Service:            stat.Service,
		Namespace:          stat.Namespace,
		Operation:          stat.Operation,
		OperationName:      stat.OperationName,
		IdentityDimensions: TransactionIdentityValuesToModel(stat.IdentityDimensions),
		Kind:               TransactionKindToModel(stat.Kind),
		Volume:             int64ToInt(stat.Volume),
		LastSeen:           stat.LastSeen,
		HasBaseline:        stat.HasBaseline,
		Promoted:           stat.Promoted,
	}
}

func TransactionStatsToModel(stats []TransactionStat) []*model.InsightsTransactionStat {
	return mapSlice(stats, TransactionStatToModel)
}

func TransactionToModel(transaction Transaction) *model.InsightsTransaction {
	return &model.InsightsTransaction{
		ID:                 FormatID(transaction.ID),
		Service:            transaction.Service,
		Namespace:          transaction.Namespace,
		Operation:          transaction.Operation,
		OperationName:      transaction.OperationName,
		IdentityDimensions: TransactionIdentityValuesToModel(transaction.IdentityDimensions),
		Kind:               TransactionKindToModel(transaction.Kind),
	}
}

func BaselineClassToModel(baseline BaselineClass) (*model.InsightsBaselineClass, error) {
	data, err := encodeOptionalJSONString(baseline.Data)
	if err != nil {
		return nil, err
	}
	return &model.InsightsBaselineClass{
		TransactionID:                FormatID(baseline.TransactionID),
		Class:                        DeviationClassToModel(baseline.Class),
		ClassLabel:                   baseline.ClassLabel,
		ClassDescription:             baseline.ClassDescription,
		Data:                         data,
		DataSchemaVersion:            baseline.DataSchemaVersion,
		ObservationCount:             int64ToInt(baseline.ObservationCount),
		Promoted:                     baseline.Promoted,
		LearningStartedAt:            baseline.LearningStartedAt,
		LastChangedAt:                baseline.LastChangedAt,
		ObservationCountAtLastChange: int64PtrToIntPtr(baseline.ObservationCountAtLastChange),
		Learning:                     BaselineLearningToModel(baseline.Learning),
	}, nil
}

func BaselineLearningToModel(learning BaselineLearning) *model.InsightsBaselineLearning {
	return &model.InsightsBaselineLearning{
		Phase:                       model.InsightsBaselineLearningPhase(learning.Phase),
		ObservationsSinceLastChange: int64ToInt(learning.ObservationsSinceLastChange),
		QuietMinutes:                int64ToInt(learning.QuietMinutes),
		Mode:                        LearningModeToModel(learning.Mode),
		ReadyToPromote:              learning.ReadyToPromote,
		Stability: &model.InsightsBaselineLearningStability{
			Observations: BaselineStabilityProgressToModel(learning.Stability.Observations),
			Duration:     BaselineStabilityProgressToModel(learning.Stability.Duration),
		},
	}
}

func BaselineStabilityProgressToModel(progress BaselineStabilityProgress) *model.InsightsBaselineStabilityProgress {
	return &model.InsightsBaselineStabilityProgress{
		Current:         int64ToInt(progress.Current),
		Target:          int64ToInt(progress.Target),
		DrivesPromotion: progress.DrivesPromotion,
		Met:             progress.Met,
	}
}

func BaselineClassesToModel(baselines []BaselineClass) ([]*model.InsightsBaselineClass, error) {
	return mapSliceErr(baselines, BaselineClassToModel)
}

func PromoteResultToModel(result PromoteResult) *model.InsightsPromoteResult {
	return &model.InsightsPromoteResult{
		TransactionID: FormatID(result.TransactionID),
		Class:         DeviationClassToModel(result.Class),
		Promoted:      result.Promoted,
	}
}

func BulkPromoteResultToModel(result BulkPromoteResult) *model.InsightsBulkPromoteResult {
	return &model.InsightsBulkPromoteResult{
		Promoted: result.Promoted,
	}
}

func BulkDeleteResultToModel(result BulkDeleteResult) *model.InsightsBulkDeleteResult {
	return &model.InsightsBulkDeleteResult{
		Deleted: result.Deleted,
	}
}

func ObservationSummaryToModel(observation ObservationSummary) *model.InsightsObservationSummary {
	return &model.InsightsObservationSummary{
		TraceID:       observation.TraceID,
		TransactionID: FormatID(observation.TransactionID),
		ObservedAt:    observation.ObservedAt,
		DurationMs:    int64ToInt(observation.DurationMs),
		SampleReason:  SampleReasonToModel(observation.SampleReason),
	}
}

func ObservationSummariesToModel(observations []ObservationSummary) []*model.InsightsObservationSummary {
	return mapSlice(observations, ObservationSummaryToModel)
}

func ObservationToModel(observation Observation) *model.InsightsObservation {
	return &model.InsightsObservation{
		TraceID:       observation.TraceID,
		TransactionID: FormatID(observation.TransactionID),
		ObservedAt:    observation.ObservedAt,
		DurationMs:    int64ToInt(observation.DurationMs),
		SampleReason:  SampleReasonToModel(observation.SampleReason),
		RawOtlp:       observation.RawOTLP,
	}
}

func PolicyToModel(policy Policy) (*model.InsightsPolicy, error) {
	signalWeights, err := encodeOptionalJSONString(policy.SignalWeights)
	if err != nil {
		return nil, err
	}
	enricherLists, err := encodeOptionalJSONString(policy.EnricherLists)
	if err != nil {
		return nil, err
	}
	return &model.InsightsPolicy{
		ID:            FormatID(policy.ID),
		Name:          policy.Name,
		Enabled:       policy.Enabled,
		FireAtScore:   policy.FireAtScore,
		SignalWeights: signalWeights,
		EnricherLists: enricherLists,
		Scope:         PolicyScopeToModel(policy.Scope),
		ScopeKey:      policy.ScopeKey,
	}, nil
}

func PoliciesToModel(policies []Policy) ([]*model.InsightsPolicy, error) {
	return mapSliceErr(policies, PolicyToModel)
}

func LearningConditionToModel(condition LearningCondition) *model.InsightsLearningCondition {
	return &model.InsightsLearningCondition{
		Type:               LearningConditionTypeToModel(condition.Type),
		MinObservations:    int64PtrToIntPtr(condition.MinObservations),
		MinDurationMinutes: int64PtrToIntPtr(condition.MinDurationMinutes),
		StableObservations: int64PtrToIntPtr(condition.StableObservations),
		StableMinutes:      int64PtrToIntPtr(condition.StableMinutes),
	}
}

func LearningConditionFromInput(condition model.InsightsLearningConditionInput) LearningCondition {
	return LearningCondition{
		Type:               LearningConditionTypeFromModel(condition.Type),
		MinObservations:    intPtrToInt64Ptr(condition.MinObservations),
		MinDurationMinutes: intPtrToInt64Ptr(condition.MinDurationMinutes),
		StableObservations: intPtrToInt64Ptr(condition.StableObservations),
		StableMinutes:      intPtrToInt64Ptr(condition.StableMinutes),
	}
}

func LearningPolicyToModel(policy LearningPolicy) *model.InsightsLearningPolicy {
	return &model.InsightsLearningPolicy{
		Class:      DeviationClassToModel(policy.Class),
		Mode:       LearningModeToModel(policy.Mode),
		Conditions: mapSlice(policy.Conditions, LearningConditionToModel),
		MinMatches: policy.MinMatches,
		Scope:      PolicyScopeToModel(policy.Scope),
		ScopeKey:   policy.ScopeKey,
	}
}

func LearningPoliciesToModel(policies []LearningPolicy) []*model.InsightsLearningPolicy {
	return mapSlice(policies, LearningPolicyToModel)
}

func FindingToModel(finding Finding) *model.InsightsFinding {
	return &model.InsightsFinding{
		Kind:               FindingKindToModel(finding.Kind),
		Service:            finding.Service,
		Namespace:          finding.Namespace,
		Title:              finding.Title,
		Operation:          finding.Operation,
		OperationName:      finding.OperationName,
		IdentityDimensions: mapSlice(finding.IdentityDimensions, TransactionIdentityValueToModel),
		Offending:          finding.Offending,
		Score:              finding.Score,
		Severity:           SeverityToModel(finding.Severity),
		Occurrences:        int64ToInt(finding.Occurrences),
		LastSeen:           finding.LastSeen,
		Status:             finding.Status,
		TransactionID:      formatOptionalID(finding.TransactionID),
		Signature:          finding.Signature,
		TriggeredClasses:   mapSlice(finding.TriggeredClasses, DeviationClassToModel),
		ScopeKey:           finding.ScopeKey,
		RuleKey:            finding.RuleKey,
	}
}

func FindingsToModel(findings []Finding) []*model.InsightsFinding {
	return mapSlice(findings, FindingToModel)
}

func RiskSignalToModel(signal RiskSignal) *model.InsightsRiskSignal {
	return &model.InsightsRiskSignal{
		Source:       signal.Source,
		Category:     signal.Category,
		Weight:       signal.Weight,
		Confidence:   signal.Confidence,
		Contribution: signal.Contribution,
		Rationale:    signal.Rationale,
		Mitre:        signal.Mitre,
		Owasp:        signal.Owasp,
	}
}

func RiskAssessmentToModel(risk RiskAssessment) *model.InsightsRiskAssessment {
	return &model.InsightsRiskAssessment{
		Score:            risk.Score,
		Severity:         SeverityToModel(risk.Severity),
		Correlated:       risk.Correlated,
		CorrelationBonus: risk.CorrelationBonus,
		FireAtScore:      risk.FireAtScore,
		Categories:       risk.Categories,
		Signals:          mapSlice(risk.Signals, RiskSignalToModel),
	}
}

func AnomalySummaryToModel(anomaly AnomalySummary) *model.InsightsAnomalySummary {
	return &model.InsightsAnomalySummary{
		TransactionID:      FormatID(anomaly.TransactionID),
		Signature:          anomaly.Signature,
		Service:            anomaly.Service,
		Namespace:          anomaly.Namespace,
		Operation:          anomaly.Operation,
		OperationName:      anomaly.OperationName,
		IdentityDimensions: TransactionIdentityValuesToModel(anomaly.IdentityDimensions),
		Kind:               TransactionKindPtrToModel(anomaly.Kind),
		TriggeredClasses:   mapSlice(anomaly.TriggeredClasses, DeviationClassToModel),
		Offending:          anomaly.Offending,
		PolicyID:           formatOptionalID(anomaly.PolicyID),
		Occurrences:        int64ToInt(anomaly.Occurrences),
		MaxScore:           anomaly.MaxScore,
		Severity:           SeverityToModel(anomaly.Severity),
		FirstSeen:          anomaly.FirstSeen,
		LastSeen:           anomaly.LastSeen,
		LastTraceID:        anomaly.LastTraceID,
		Status:             AnomalyStatusToModel(anomaly.Status),
	}
}

func AnomalySummariesToModel(anomalies []AnomalySummary) []*model.InsightsAnomalySummary {
	return mapSlice(anomalies, AnomalySummaryToModel)
}

func AnomalySpanHighlightToModel(highlight AnomalySpanHighlight) *model.InsightsAnomalySpanHighlight {
	return &model.InsightsAnomalySpanHighlight{
		SpanID:   highlight.SpanID,
		Severity: highlight.Severity,
		Reason:   highlight.Reason,
	}
}

func AnomalyAttrHighlightToModel(highlight AnomalyAttrHighlight) *model.InsightsAnomalyAttrHighlight {
	return &model.InsightsAnomalyAttrHighlight{
		SpanID: highlight.SpanID,
		Key:    highlight.Key,
		Value:  highlight.Value,
	}
}

func AnomalyMetricComparisonToModel(metric AnomalyMetricComparison) *model.InsightsAnomalyMetricComparison {
	return &model.InsightsAnomalyMetricComparison{
		Observed: int64ToInt(metric.Observed),
		Baseline: int64ToInt(metric.Baseline),
		Unit:     metric.Unit,
	}
}

func AnomalyClassFindingToModel(finding AnomalyClassFinding) (*model.InsightsAnomalyClassFinding, error) {
	rawEvidence, err := encodeOptionalJSONString(finding.RawEvidence)
	if err != nil {
		return nil, err
	}
	var metric *model.InsightsAnomalyMetricComparison
	if finding.Metric != nil {
		metric = AnomalyMetricComparisonToModel(*finding.Metric)
	}
	return &model.InsightsAnomalyClassFinding{
		Class:          DeviationClassToModel(finding.Class),
		Kind:           model.InsightsAnomalyClassFindingKind(finding.Kind),
		Title:          finding.Title,
		Summary:        finding.Summary,
		SpanHighlights: mapSlice(finding.SpanHighlights, AnomalySpanHighlightToModel),
		AttrHighlights: mapSlice(finding.AttrHighlights, AnomalyAttrHighlightToModel),
		Metric:         metric,
		RawEvidence:    rawEvidence,
	}, nil
}

func AnomalyIssueToModel(anomaly AnomalyIssue) (*model.InsightsAnomalyIssue, error) {
	evidence := "{}"
	if len(anomaly.Evidence) > 0 {
		evidence = string(anomaly.Evidence)
	}
	var risk *model.InsightsRiskAssessment
	if anomaly.Risk != nil {
		risk = RiskAssessmentToModel(*anomaly.Risk)
	}
	classFindings, err := mapSliceErr(anomaly.ClassFindings, AnomalyClassFindingToModel)
	if err != nil {
		return nil, err
	}
	if classFindings == nil {
		classFindings = []*model.InsightsAnomalyClassFinding{}
	}
	var anomalyTrace *model.InsightsObservation
	if anomaly.AnomalyTrace != nil {
		anomalyTrace = ObservationToModel(*anomaly.AnomalyTrace)
	}
	var baselineTrace *model.InsightsObservation
	if anomaly.BaselineTrace != nil {
		baselineTrace = ObservationToModel(*anomaly.BaselineTrace)
	}
	return &model.InsightsAnomalyIssue{
		TransactionID:      FormatID(anomaly.TransactionID),
		Signature:          anomaly.Signature,
		Service:            anomaly.Service,
		Namespace:          anomaly.Namespace,
		Operation:          anomaly.Operation,
		OperationName:      anomaly.OperationName,
		IdentityDimensions: TransactionIdentityValuesToModel(anomaly.IdentityDimensions),
		Kind:               TransactionKindPtrToModel(anomaly.Kind),
		TriggeredClasses:   mapSlice(anomaly.TriggeredClasses, DeviationClassToModel),
		Offending:          anomaly.Offending,
		PolicyID:           formatOptionalID(anomaly.PolicyID),
		Occurrences:        int64ToInt(anomaly.Occurrences),
		MaxScore:           anomaly.MaxScore,
		Severity:           SeverityToModel(anomaly.Severity),
		FirstSeen:          anomaly.FirstSeen,
		LastSeen:           anomaly.LastSeen,
		LastTraceID:        anomaly.LastTraceID,
		Status:             AnomalyStatusToModel(anomaly.Status),
		Evidence:           evidence,
		Risk:               risk,
		ClassFindings:      classFindings,
		AnomalyTrace:       anomalyTrace,
		BaselineTrace:      baselineTrace,
	}, nil
}

func BulkResolveResultToModel(result BulkResolveResult) *model.InsightsBulkResolveResult {
	return &model.InsightsBulkResolveResult{
		Resolution: result.Resolution,
		Resolved:   result.Resolved,
	}
}

func GuardrailRuleToModel(rule GuardrailRule) *model.InsightsGuardrailRule {
	return &model.InsightsGuardrailRule{
		Key:       rule.Key,
		Label:     rule.Label,
		Mode:      RuleModeToModel(rule.Mode),
		Allowlist: rule.Allowlist,
	}
}

func GuardrailToModel(guardrail Guardrail) *model.InsightsGuardrail {
	return &model.InsightsGuardrail{
		Scope:    PolicyScopeToModel(guardrail.Scope),
		ScopeKey: guardrail.ScopeKey,
		Rules:    mapSlice(guardrail.Rules, GuardrailRuleToModel),
	}
}

func GuardrailsToModel(guardrails []Guardrail) []*model.InsightsGuardrail {
	return mapSlice(guardrails, GuardrailToModel)
}

func GuardrailViolationToModel(violation GuardrailViolation) *model.InsightsGuardrailViolation {
	return &model.InsightsGuardrailViolation{
		Service:     violation.Service,
		Namespace:   violation.Namespace,
		ScopeKey:    violation.ScopeKey,
		RuleKey:     violation.RuleKey,
		RuleLabel:   violation.RuleLabel,
		Offending:   violation.Offending,
		Occurrences: int64ToInt(violation.Occurrences),
		MaxScore:    violation.MaxScore,
		Severity:    SeverityToModel(violation.Severity),
		LastSeen:    violation.LastSeen,
		Status:      ViolationStatusToModel(violation.Status),
	}
}

func GuardrailViolationsToModel(violations []GuardrailViolation) []*model.InsightsGuardrailViolation {
	return mapSlice(violations, GuardrailViolationToModel)
}

func GuardrailViolationDetailToModel(detail GuardrailViolationDetail) *model.InsightsGuardrailViolationDetail {
	var evidenceTrace *model.InsightsObservation
	if detail.EvidenceTrace != nil {
		evidenceTrace = ObservationToModel(*detail.EvidenceTrace)
	}
	return &model.InsightsGuardrailViolationDetail{
		Service:        detail.Service,
		Namespace:      detail.Namespace,
		ScopeKey:       detail.ScopeKey,
		RuleKey:        detail.RuleKey,
		RuleLabel:      detail.RuleLabel,
		Offending:      detail.Offending,
		Occurrences:    int64ToInt(detail.Occurrences),
		MaxScore:       detail.MaxScore,
		Severity:       SeverityToModel(detail.Severity),
		LastSeen:       detail.LastSeen,
		Status:         ViolationStatusToModel(detail.Status),
		Summary:        detail.Summary,
		SpanHighlights: mapSlice(detail.SpanHighlights, AnomalySpanHighlightToModel),
		EvidenceTrace:  evidenceTrace,
	}
}

func CatalogClassToModel(class CatalogClass) *model.InsightsCatalogClass {
	return &model.InsightsCatalogClass{
		ID:                  class.ID,
		Label:               class.Label,
		Description:         class.Description,
		BaselineLabel:       class.BaselineLabel,
		BaselineDescription: class.BaselineDescription,
		Category:            class.Category,
		CategoryLabel:       class.CategoryLabel,
		Weight:              class.Weight,
		Mitre:               class.Mitre,
		Owasp:               class.Owasp,
	}
}

func EnricherListToModel(list EnricherList) *model.InsightsEnricherList {
	return &model.InsightsEnricherList{
		Key:     list.Key,
		Label:   list.Label,
		Hint:    list.Hint,
		Default: list.Default,
	}
}

func CatalogEnricherToModel(enricher CatalogEnricher) *model.InsightsCatalogEnricher {
	var list *model.InsightsEnricherList
	if enricher.List != nil {
		list = EnricherListToModel(*enricher.List)
	}
	return &model.InsightsCatalogEnricher{
		Source:      enricher.Source,
		ParentClass: enricher.ParentClass,
		Category:    enricher.Category,
		Weight:      enricher.Weight,
		Label:       enricher.Label,
		Description: enricher.Description,
		List:        list,
	}
}

func CatalogCategoryToModel(category CatalogCategory) *model.InsightsCatalogCategory {
	return &model.InsightsCatalogCategory{
		ID:    category.ID,
		Label: category.Label,
		Base:  category.Base,
		Mitre: category.Mitre,
		Owasp: category.Owasp,
	}
}

func CatalogGuardrailRuleToModel(rule CatalogGuardrailRule) *model.InsightsCatalogGuardrailRule {
	return &model.InsightsCatalogGuardrailRule{
		Key:         rule.Key,
		Label:       rule.Label,
		Description: rule.Description,
		Category:    rule.Category,
		Weight:      rule.Weight,
		Hint:        rule.Hint,
	}
}

func SeverityBandToModel(band SeverityBand) *model.InsightsSeverityBand {
	return &model.InsightsSeverityBand{
		Severity: SeverityToModel(band.Severity),
		MinScore: band.MinScore,
	}
}

func CatalogToModel(catalog Catalog) (*model.InsightsCatalog, error) {
	tagsMap := catalog.Tags
	if tagsMap == nil {
		tagsMap = map[string]string{}
	}
	tags, err := encodeJSONString(tagsMap)
	if err != nil {
		return nil, err
	}
	return &model.InsightsCatalog{
		DeviationClasses: mapSlice(catalog.DeviationClasses, CatalogClassToModel),
		Enrichers:        mapSlice(catalog.Enrichers, CatalogEnricherToModel),
		Categories:       mapSlice(catalog.Categories, CatalogCategoryToModel),
		GuardrailRules:   mapSlice(catalog.GuardrailRules, CatalogGuardrailRuleToModel),
		SeverityBands:    mapSlice(catalog.SeverityBands, SeverityBandToModel),
		CorrelationBonus: catalog.CorrelationBonus,
		Tags:             tags,
	}, nil
}

func SystemSettingsToModel(settings SystemSettings) *model.InsightsSystemSettings {
	return &model.InsightsSystemSettings{
		Sampling: &model.InsightsSystemSamplingSettings{
			ExamplesPerTransaction:       settings.Sampling.ExamplesPerTransaction,
			ExampleSampleIntervalSeconds: settings.Sampling.ExampleSampleIntervalSeconds,
		},
		Retention: &model.InsightsSystemRetentionSettings{
			ObservationRetentionDays: settings.Retention.ObservationRetentionDays,
		},
		Findings: &model.InsightsSystemFindingsSettings{
			DefaultWindowHours: settings.Findings.DefaultWindowHours,
			MaxWindowHours:     settings.Findings.MaxWindowHours,
		},
		Capacity: &model.InsightsSystemCapacitySettings{
			MaxResidentTransactions: settings.Capacity.MaxResidentTransactions,
			MaxBaselineSetMembers:   settings.Capacity.MaxBaselineSetMembers,
		},
		Writeback: &model.InsightsSystemWritebackSettings{
			FlushIntervalSeconds: settings.Writeback.FlushIntervalSeconds,
		},
		Detection: &model.InsightsSystemDetectionSettings{
			AutoTransactionGuardrail: settings.Detection.AutoTransactionGuardrail,
		},
		Identity: SystemIdentitySettingsToModel(settings.Identity),
	}
}

func SystemIdentitySettingsToModel(identity SystemIdentitySettings) *model.InsightsSystemIdentitySettings {
	dims := mapSlice(identity.TransactionIdentityDimensions, SystemTransactionIdentityDimensionToModel)
	if dims == nil {
		dims = []*model.InsightsSystemTransactionIdentityDimension{}
	}
	return &model.InsightsSystemIdentitySettings{
		TransactionIdentityDimensions: dims,
	}
}

func SystemTransactionIdentityDimensionToModel(dim SystemTransactionIdentityDimension) *model.InsightsSystemTransactionIdentityDimension {
	return &model.InsightsSystemTransactionIdentityDimension{
		Key:     dim.Key,
		Enabled: dim.Enabled,
	}
}

func StorageQueryToModel(query StorageQuery) *model.InsightsStorageQuery {
	return &model.InsightsStorageQuery{
		ElapsedSeconds: query.ElapsedSeconds,
		User:           query.User,
		MemoryBytes:    int64ToInt(query.MemoryBytes),
		Query:          query.Query,
		FinishedAt:     query.FinishedAt,
	}
}

func StorageTableToModel(table StorageTable) *model.InsightsStorageTable {
	return &model.InsightsStorageTable{
		Name:        table.Name,
		Rows:        int64ToInt(table.Rows),
		BytesOnDisk: int64ToInt(table.BytesOnDisk),
		ActiveParts: int64ToInt(table.ActiveParts),
	}
}

func StorageHealthToModel(health StorageHealth) *model.InsightsStorageHealth {
	return &model.InsightsStorageHealth{
		CheckedAt: health.CheckedAt,
		Status:    StorageHealthStatusToModel(health.Status),
		Connection: &model.InsightsStorageConnection{
			Reachable:     health.Connection.Reachable,
			PingLatencyMs: int64ToInt(health.Connection.PingLatencyMs),
			Version:       health.Connection.Version,
			Database:      health.Connection.Database,
			Error:         health.Connection.Error,
		},
		Disk: &model.InsightsStorageDisk{
			InsightsBytes:  int64ToInt(health.Disk.InsightsBytes),
			TotalBytes:     int64ToInt(health.Disk.TotalBytes),
			UsedBytes:      int64ToInt(health.Disk.UsedBytes),
			AvailableBytes: int64ToInt(health.Disk.AvailableBytes),
			UsedPercent:    health.Disk.UsedPercent,
			Status:         StorageDiskStatusToModel(health.Disk.Status),
		},
		Memory: &model.InsightsStorageMemory{
			ProcessRssBytes:  int64ToInt(health.Memory.ProcessRSSBytes),
			CgroupUsedBytes:  int64ToInt(health.Memory.CgroupUsedBytes),
			CgroupTotalBytes: int64ToInt(health.Memory.CgroupTotalBytes),
			UptimeSeconds:    health.Memory.UptimeSeconds,
		},
		Queries: &model.InsightsStorageQueries{
			Active:          health.Queries.Active,
			LongestSeconds:  health.Queries.LongestSeconds,
			PeakMemoryBytes: int64ToInt(health.Queries.PeakMemoryBytes),
			Items:           mapSlice(health.Queries.Items, StorageQueryToModel),
			RecentHeaviest:  mapSlice(health.Queries.RecentHeaviest, StorageQueryToModel),
			RecentLatest:    mapSlice(health.Queries.RecentLatest, StorageQueryToModel),
		},
		Tables: mapSlice(health.Tables, StorageTableToModel),
		WritePressure: &model.InsightsStorageWritePressure{
			UnfinishedMutations: int64ToInt(health.WritePressure.UnfinishedMutations),
			ActiveParts:         int64ToInt(health.WritePressure.ActiveParts),
		},
		Writeback: &model.InsightsStorageWriteback{
			FlushIntervalSeconds: health.Writeback.FlushIntervalSeconds,
			EverFlushed:          health.Writeback.EverFlushed,
			LastFlushOk:          health.Writeback.LastFlushOK,
			LastFlushAt:          health.Writeback.LastFlushAt,
			LastFlushError:       health.Writeback.LastFlushError,
		},
	}
}

// Input converters (GraphQL → REST).

func PolicyFromInput(input model.InsightsPolicyInput) (Policy, error) {
	id, err := parseOptionalID(input.ID)
	if err != nil {
		return Policy{}, err
	}
	signalWeights, err := decodeOptionalJSONString[map[string]int](input.SignalWeights)
	if err != nil {
		return Policy{}, err
	}
	enricherLists, err := decodeOptionalJSONString[map[string][]string](input.EnricherLists)
	if err != nil {
		return Policy{}, err
	}
	policy := Policy{
		Name:          input.Name,
		Enabled:       input.Enabled,
		FireAtScore:   input.FireAtScore,
		SignalWeights: signalWeights,
		EnricherLists: enricherLists,
		Scope:         PolicyScopeFromModel(input.Scope),
		ScopeKey:      input.ScopeKey,
	}
	if id != nil {
		policy.ID = *id
	}
	return policy, nil
}

func LearningPolicyFromInput(input model.InsightsLearningPolicyInput) LearningPolicy {
	var conditions []LearningCondition
	if input.Conditions != nil {
		conditions = make([]LearningCondition, 0, len(input.Conditions))
		for _, condition := range input.Conditions {
			if condition == nil {
				continue
			}
			conditions = append(conditions, LearningConditionFromInput(*condition))
		}
	}
	return LearningPolicy{
		Class:      DeviationClassFromModel(input.Class),
		Mode:       LearningModeFromModel(input.Mode),
		Conditions: conditions,
		MinMatches: input.MinMatches,
		Scope:      PolicyScopeFromModel(input.Scope),
		ScopeKey:   input.ScopeKey,
	}
}

func GuardrailRuleFromInput(input model.InsightsGuardrailRuleInput) GuardrailRule {
	return GuardrailRule{
		Key:       input.Key,
		Label:     input.Label,
		Mode:      RuleModeFromModel(input.Mode),
		Allowlist: input.Allowlist,
	}
}

func GuardrailFromInput(input model.InsightsGuardrailInput) Guardrail {
	rules := make([]GuardrailRule, 0, len(input.Rules))
	for _, rule := range input.Rules {
		if rule == nil {
			continue
		}
		rules = append(rules, GuardrailRuleFromInput(*rule))
	}
	return Guardrail{
		Scope:    PolicyScopeFromModel(input.Scope),
		ScopeKey: input.ScopeKey,
		Rules:    rules,
	}
}

func GuardrailSeedFromInput(input model.InsightsGuardrailSeedInput) (GuardrailSeedRequest, error) {
	items, err := decodeJSONString[map[string][]string](input.Items)
	if err != nil {
		return GuardrailSeedRequest{}, err
	}
	return GuardrailSeedRequest{
		ScopeKey: input.ScopeKey,
		Mode:     RuleModePtrFromModel(input.Mode),
		Items:    items,
	}, nil
}

func ViolationActionFromInput(input model.InsightsViolationActionInput) ViolationActionRequest {
	return ViolationActionRequest{
		ScopeKey:  input.ScopeKey,
		RuleKey:   input.RuleKey,
		Offending: input.Offending,
	}
}

func AnomalyRefFromInput(input model.InsightsAnomalyRefInput) (AnomalyRef, error) {
	transactionID, err := ParseID(input.TransactionID)
	if err != nil {
		return AnomalyRef{}, err
	}
	return AnomalyRef{
		TransactionID: transactionID,
		Signature:     input.Signature,
	}, nil
}

func BulkAnomalyRequestFromInput(resolution model.InsightsBulkResolution, items []*model.InsightsAnomalyRefInput) (BulkAnomalyRequest, error) {
	refs, err := mapSliceErr(items, func(item *model.InsightsAnomalyRefInput) (AnomalyRef, error) {
		if item == nil {
			return AnomalyRef{}, fmt.Errorf("%w: anomaly ref item is nil", ErrBadRequest)
		}
		return AnomalyRefFromInput(*item)
	})
	if err != nil {
		return BulkAnomalyRequest{}, err
	}
	return BulkAnomalyRequest{
		Resolution: BulkResolutionFromModel(resolution),
		Items:      refs,
	}, nil
}

func SystemSettingsFromInput(input model.InsightsSystemSettingsInput) (SystemSettings, error) {
	if input.Sampling == nil || input.Retention == nil || input.Findings == nil || input.Capacity == nil || input.Writeback == nil || input.Detection == nil || input.Identity == nil {
		return SystemSettings{}, fmt.Errorf("%w: system settings input is incomplete", ErrBadRequest)
	}
	return SystemSettings{
		Sampling: SystemSamplingSettings{
			ExamplesPerTransaction:       input.Sampling.ExamplesPerTransaction,
			ExampleSampleIntervalSeconds: input.Sampling.ExampleSampleIntervalSeconds,
		},
		Retention: SystemRetentionSettings{
			ObservationRetentionDays: input.Retention.ObservationRetentionDays,
		},
		Findings: SystemFindingsSettings{
			DefaultWindowHours: input.Findings.DefaultWindowHours,
			MaxWindowHours:     input.Findings.MaxWindowHours,
		},
		Capacity: SystemCapacitySettings{
			MaxResidentTransactions: input.Capacity.MaxResidentTransactions,
			MaxBaselineSetMembers:   input.Capacity.MaxBaselineSetMembers,
		},
		Writeback: SystemWritebackSettings{
			FlushIntervalSeconds: input.Writeback.FlushIntervalSeconds,
		},
		Detection: SystemDetectionSettings{
			AutoTransactionGuardrail: input.Detection.AutoTransactionGuardrail,
		},
		Identity:          SystemIdentitySettingsFromInput(input.Identity),
		ResetTransactions: input.ResetTransactions,
	}, nil
}

func SystemIdentitySettingsFromInput(identity *model.InsightsSystemIdentitySettingsInput) SystemIdentitySettings {
	if identity == nil {
		return SystemIdentitySettings{TransactionIdentityDimensions: []SystemTransactionIdentityDimension{}}
	}
	dims := make([]SystemTransactionIdentityDimension, 0, len(identity.TransactionIdentityDimensions))
	for _, dim := range identity.TransactionIdentityDimensions {
		if dim == nil {
			continue
		}
		dims = append(dims, SystemTransactionIdentityDimension{
			Key:     dim.Key,
			Enabled: dim.Enabled,
		})
	}
	return SystemIdentitySettings{TransactionIdentityDimensions: dims}
}
