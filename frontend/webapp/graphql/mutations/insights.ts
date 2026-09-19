import { gql } from '@apollo/client';

const POLICY_FIELDS = `
  id
  name
  enabled
  fireAtScore
  signalWeights
  enricherLists
  scope
  scopeKey
`;

const LEARNING_POLICY_FIELDS = `
  class
  mode
  conditions {
    type
    minObservations
    minDurationMinutes
    stableObservations
    stableMinutes
  }
  minMatches
  scope
  scopeKey
`;

const GUARDRAIL_FIELDS = `
  scope
  scopeKey
  rules {
    key
    label
    mode
    allowlist
    correlations {
      name
      left {
        service
        span
        attr
        extract
      }
      right {
        service
        span
        attr
        extract
      }
      relation
      severity
      why
    }
    origin
  }
`;

const SYSTEM_SETTINGS_FIELDS = `
  sampling {
    examplesPerTransaction
    exampleSampleIntervalSeconds
  }
  retention {
    observationRetentionDays
  }
  capacity {
    maxResidentTransactions
    maxBaselineSetMembers
  }
  writeback {
    flushIntervalSeconds
  }
  detection {
    autoTransactionGuardrail
  }
  identity {
    transactionIdentityDimensions {
      key
      enabled
    }
  }
`;

const RECOMMENDATION_BULK_RESULT_FIELDS = `
  done
  failed
  errors {
    id
    error
  }
`;

const RECOMMENDATION_EXAMPLE_FIELDS = `
  traceId
  leftValue
  rightValue
  observedAt
`;

export const PROMOTE_INSIGHTS_BASELINE_CLASS = gql`
  mutation PromoteInsightsBaselineClass($transactionId: ID!, $class: InsightsDeviationClass!) {
    promoteInsightsBaselineClass(transactionId: $transactionId, class: $class) {
      transactionId
      class
      promoted
    }
  }
`;

export const RESET_INSIGHTS_BASELINE_CLASS = gql`
  mutation ResetInsightsBaselineClass($transactionId: ID!, $class: InsightsDeviationClass!) {
    resetInsightsBaselineClass(transactionId: $transactionId, class: $class)
  }
`;

export const RESET_INSIGHTS_TRANSACTION_BASELINES = gql`
  mutation ResetInsightsTransactionBaselines($transactionId: ID!) {
    resetInsightsTransactionBaselines(transactionId: $transactionId)
  }
`;

export const PROMOTE_INSIGHTS_TRANSACTION_BASELINES = gql`
  mutation PromoteInsightsTransactionBaselines($transactionId: ID!) {
    promoteInsightsTransactionBaselines(transactionId: $transactionId)
  }
`;

export const BULK_PROMOTE_INSIGHTS_TRANSACTIONS = gql`
  mutation BulkPromoteInsightsTransactions($transactionIds: [ID!]!) {
    bulkPromoteInsightsTransactions(transactionIds: $transactionIds) {
      promoted
    }
  }
`;

export const FORCE_PROMOTE_INSIGHTS_SERVICE = gql`
  mutation ForcePromoteInsightsService($namespace: String!, $service: String!) {
    forcePromoteInsightsService(namespace: $namespace, service: $service)
  }
`;

export const ENABLE_INSIGHTS_TRANSACTION_GUARDRAIL = gql`
  mutation EnableInsightsTransactionGuardrail($namespace: String!, $service: String!) {
    enableInsightsTransactionGuardrail(namespace: $namespace, service: $service)
  }
`;

export const DISABLE_INSIGHTS_TRANSACTION_GUARDRAIL = gql`
  mutation DisableInsightsTransactionGuardrail($namespace: String!, $service: String!) {
    disableInsightsTransactionGuardrail(namespace: $namespace, service: $service)
  }
`;

export const DELETE_INSIGHTS_TRANSACTION = gql`
  mutation DeleteInsightsTransaction($transactionId: ID!) {
    deleteInsightsTransaction(transactionId: $transactionId)
  }
`;

export const BULK_DELETE_INSIGHTS_TRANSACTIONS = gql`
  mutation BulkDeleteInsightsTransactions($transactionIds: [ID!]!) {
    bulkDeleteInsightsTransactions(transactionIds: $transactionIds) {
      deleted
    }
  }
`;

export const UPSERT_INSIGHTS_POLICY = gql`
  mutation UpsertInsightsPolicy($policy: InsightsPolicyInput!) {
    upsertInsightsPolicy(policy: $policy) { ${POLICY_FIELDS} }
  }
`;

export const DELETE_INSIGHTS_POLICY = gql`
  mutation DeleteInsightsPolicy($scope: InsightsPolicyScope!, $scopeKey: String!) {
    deleteInsightsPolicy(scope: $scope, scopeKey: $scopeKey)
  }
`;

export const UPSERT_INSIGHTS_LEARNING_POLICY = gql`
  mutation UpsertInsightsLearningPolicy($policy: InsightsLearningPolicyInput!) {
    upsertInsightsLearningPolicy(policy: $policy) { ${LEARNING_POLICY_FIELDS} }
  }
`;

export const DELETE_INSIGHTS_LEARNING_POLICY = gql`
  mutation DeleteInsightsLearningPolicy($class: InsightsDeviationClass!, $scope: InsightsPolicyScope!, $scopeKey: String!) {
    deleteInsightsLearningPolicy(class: $class, scope: $scope, scopeKey: $scopeKey)
  }
`;

export const RESOLVE_INSIGHTS_ANOMALY = gql`
  mutation ResolveInsightsAnomaly($transactionId: ID!, $signature: String!, $resolution: InsightsAnomalyResolution!) {
    resolveInsightsAnomaly(transactionId: $transactionId, signature: $signature, resolution: $resolution)
  }
`;

export const BULK_RESOLVE_INSIGHTS_ANOMALIES = gql`
  mutation BulkResolveInsightsAnomalies($resolution: InsightsBulkResolution!, $items: [InsightsAnomalyRefInput!]!) {
    bulkResolveInsightsAnomalies(resolution: $resolution, items: $items) {
      resolution
      resolved
    }
  }
`;

export const UPSERT_INSIGHTS_GUARDRAIL = gql`
  mutation UpsertInsightsGuardrail($guardrail: InsightsGuardrailInput!) {
    upsertInsightsGuardrail(guardrail: $guardrail) { ${GUARDRAIL_FIELDS} }
  }
`;

export const DELETE_INSIGHTS_GUARDRAIL = gql`
  mutation DeleteInsightsGuardrail($scopeKey: String!, $scope: InsightsPolicyScope) {
    deleteInsightsGuardrail(scopeKey: $scopeKey, scope: $scope)
  }
`;

export const SEED_INSIGHTS_GUARDRAIL = gql`
  mutation SeedInsightsGuardrail($seed: InsightsGuardrailSeedInput!) {
    seedInsightsGuardrail(seed: $seed)
  }
`;

export const ACCEPT_INSIGHTS_GUARDRAIL_VIOLATION = gql`
  mutation AcceptInsightsGuardrailViolation($action: InsightsViolationActionInput!) {
    acceptInsightsGuardrailViolation(action: $action)
  }
`;

export const DISMISS_INSIGHTS_GUARDRAIL_VIOLATION = gql`
  mutation DismissInsightsGuardrailViolation($action: InsightsViolationActionInput!) {
    dismissInsightsGuardrailViolation(action: $action)
  }
`;

export const REOPEN_INSIGHTS_GUARDRAIL_VIOLATION = gql`
  mutation ReopenInsightsGuardrailViolation($action: InsightsViolationActionInput!) {
    reopenInsightsGuardrailViolation(action: $action)
  }
`;

export const UPDATE_INSIGHTS_SYSTEM_SETTINGS = gql`
  mutation UpdateInsightsSystemSettings($settings: InsightsSystemSettingsInput!) {
    updateInsightsSystemSettings(settings: $settings) { ${SYSTEM_SETTINGS_FIELDS} }
  }
`;

export const APPLY_INSIGHTS_RECOMMENDATIONS = gql`
  mutation ApplyInsightsRecommendations($items: [InsightsRecommendationApplyItemInput!]!) {
    applyInsightsRecommendations(items: $items) { ${RECOMMENDATION_BULK_RESULT_FIELDS} }
  }
`;

export const DISMISS_INSIGHTS_RECOMMENDATIONS = gql`
  mutation DismissInsightsRecommendations($ids: [ID!]!) {
    dismissInsightsRecommendations(ids: $ids) { ${RECOMMENDATION_BULK_RESULT_FIELDS} }
  }
`;

export const RESTORE_INSIGHTS_RECOMMENDATIONS = gql`
  mutation RestoreInsightsRecommendations($ids: [ID!]!) {
    restoreInsightsRecommendations(ids: $ids) { ${RECOMMENDATION_BULK_RESULT_FIELDS} }
  }
`;

export const REVERT_INSIGHTS_RECOMMENDATIONS = gql`
  mutation RevertInsightsRecommendations($ids: [ID!]!) {
    revertInsightsRecommendations(ids: $ids) { ${RECOMMENDATION_BULK_RESULT_FIELDS} }
  }
`;

export const PREVIEW_INSIGHTS_RECOMMENDATION = gql`
  mutation PreviewInsightsRecommendation($transactionId: ID!, $spec: InsightsCorrelationSpecInput!) {
    previewInsightsRecommendation(transactionId: $transactionId, spec: $spec) {
      sampleCount
      observed
      held
      distinct
      holdRatio
      examples { ${RECOMMENDATION_EXAMPLE_FIELDS} }
      counterExamples { ${RECOMMENDATION_EXAMPLE_FIELDS} }
    }
  }
`;

export const RECOMPUTE_INSIGHTS_RECOMMENDATIONS = gql`
  mutation RecomputeInsightsRecommendations($transactionId: ID) {
    recomputeInsightsRecommendations(transactionId: $transactionId)
  }
`;
