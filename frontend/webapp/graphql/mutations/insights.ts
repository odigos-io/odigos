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
  findings {
    defaultWindowHours
    maxWindowHours
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
  mutation DeleteInsightsGuardrail($scopeKey: String!) {
    deleteInsightsGuardrail(scopeKey: $scopeKey)
  }
`;

export const SEED_INSIGHTS_GUARDRAIL = gql`
  mutation SeedInsightsGuardrail($seed: InsightsGuardrailSeedInput!) {
    seedInsightsGuardrail(seed: $seed)
  }
`;

export const ALLOW_INSIGHTS_GUARDRAIL_VIOLATION = gql`
  mutation AllowInsightsGuardrailViolation($action: InsightsViolationActionInput!) {
    allowInsightsGuardrailViolation(action: $action)
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
