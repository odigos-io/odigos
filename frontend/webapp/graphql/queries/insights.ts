import { gql } from '@apollo/client';

const SERVICE_STAT_FIELDS = `
  namespace
  service
  transactionCount
  volume
  lastSeen
`;

const TRANSACTION_STAT_FIELDS = `
  id
  service
  namespace
  operation
  kind
  volume
  lastSeen
  hasBaseline
  promoted
`;

const TRANSACTION_FIELDS = `
  id
  service
  namespace
  operation
  kind
`;

const BASELINE_CLASS_FIELDS = `
  transactionId
  class
  data
  dataSchemaVersion
  observationCount
  promoted
  learningStartedAt
  lastChangedAt
  observationCountAtLastChange
  learning {
    phase
    observationsSinceLastChange
    quietMinutes
    mode
    readyToPromote
    stability {
      observations {
        current
        target
        drivesPromotion
        met
      }
      duration {
        current
        target
        drivesPromotion
        met
      }
    }
  }
`;

const OBSERVATION_SUMMARY_FIELDS = `
  traceId
  transactionId
  observedAt
  durationMs
  sampleReason
`;

const FINDING_FIELDS = `
  kind
  service
  namespace
  title
  offending
  score
  severity
  occurrences
  lastSeen
  status
  transactionId
  signature
  triggeredClasses
  scopeKey
  ruleKey
`;

const ANOMALY_SUMMARY_FIELDS = `
  transactionId
  signature
  service
  namespace
  operation
  kind
  triggeredClasses
  offending
  policyId
  occurrences
  maxScore
  severity
  firstSeen
  lastSeen
  lastTraceId
  status
`;

const RISK_ASSESSMENT_FIELDS = `
  score
  severity
  correlated
  correlationBonus
  fireAtScore
  categories
  signals {
    source
    category
    weight
    confidence
    contribution
    rationale
    mitre
    owasp
  }
`;

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

const GUARDRAIL_VIOLATION_FIELDS = `
  service
  namespace
  scopeKey
  ruleKey
  ruleLabel
  offending
  occurrences
  maxScore
  severity
  lastSeen
  status
`;

const CATALOG_FIELDS = `
  deviationClasses {
    id
    label
    description
    category
    categoryLabel
    weight
    mitre
    owasp
  }
  enrichers {
    source
    parentClass
    category
    weight
    label
    description
    list {
      key
      label
      hint
      default
    }
  }
  categories {
    id
    label
    base
    mitre
    owasp
  }
  guardrailRules {
    key
    label
    description
    category
    weight
    hint
  }
  severityBands {
    severity
    minScore
  }
  correlationBonus
  tags
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
`;

const STORAGE_HEALTH_FIELDS = `
  checkedAt
  status
  connection {
    reachable
    pingLatencyMs
    version
    database
    error
  }
  disk {
    insightsBytes
    totalBytes
    usedBytes
    availableBytes
    usedPercent
    status
  }
  memory {
    processRssBytes
    cgroupUsedBytes
    cgroupTotalBytes
    uptimeSeconds
  }
  queries {
    active
    longestSeconds
    peakMemoryBytes
    items {
      elapsedSeconds
      user
      memoryBytes
      query
      finishedAt
    }
    recentHeaviest {
      elapsedSeconds
      user
      memoryBytes
      query
      finishedAt
    }
    recentLatest {
      elapsedSeconds
      user
      memoryBytes
      query
      finishedAt
    }
  }
  tables {
    name
    rows
    bytesOnDisk
    activeParts
  }
  writePressure {
    unfinishedMutations
    activeParts
  }
  writeback {
    flushIntervalSeconds
    everFlushed
    lastFlushOk
    lastFlushAt
    lastFlushError
  }
`;

export const GET_INSIGHTS_SERVICES = gql`
  query GetInsightsServices {
    insights {
      services { ${SERVICE_STAT_FIELDS} }
    }
  }
`;

export const GET_INSIGHTS_SERVICE_PROFILE = gql`
  query GetInsightsServiceProfile($namespace: String, $service: String!) {
    insights {
      serviceProfile(namespace: $namespace, service: $service) {
        namespace
        service
        callers
        callees
        egress
      }
    }
  }
`;

const BLAST_RADIUS_NODE_FIELDS = `
  namespace
  service
  isVirtual
`;

const BLAST_RADIUS_EDGE_FIELDS = `
  clientNamespace
  clientService
  serverNamespace
  serverService
  connectionType
  requestCount
  failedCount
  lastSeen
`;

export const GET_INSIGHTS_BLAST_RADIUS = gql`
  query GetInsightsBlastRadius($namespace: String, $service: String!, $depth: Int) {
    insights {
      blastRadius(namespace: $namespace, service: $service, depth: $depth) {
        root { ${BLAST_RADIUS_NODE_FIELDS} }
        nodes { ${BLAST_RADIUS_NODE_FIELDS} }
        edges { ${BLAST_RADIUS_EDGE_FIELDS} }
      }
    }
  }
`;

export const GET_INSIGHTS_TRANSACTIONS = gql`
  query GetInsightsTransactions($namespace: String, $service: String, $kind: InsightsTransactionKind) {
    insights {
      transactions(namespace: $namespace, service: $service, kind: $kind) { ${TRANSACTION_STAT_FIELDS} }
    }
  }
`;

export const GET_INSIGHTS_TRANSACTION = gql`
  query GetInsightsTransaction($id: ID!) {
    insights {
      transaction(id: $id) { ${TRANSACTION_FIELDS} }
    }
  }
`;

export const GET_INSIGHTS_BASELINE = gql`
  query GetInsightsBaseline($transactionId: ID!) {
    insights {
      baseline(transactionId: $transactionId) { ${BASELINE_CLASS_FIELDS} }
    }
  }
`;

export const GET_INSIGHTS_OBSERVATIONS = gql`
  query GetInsightsObservations($transactionId: ID!, $sampleReason: InsightsSampleReason) {
    insights {
      observations(transactionId: $transactionId, sampleReason: $sampleReason) { ${OBSERVATION_SUMMARY_FIELDS} }
    }
  }
`;

export const GET_INSIGHTS_OBSERVATION = gql`
  query GetInsightsObservation($transactionId: ID!, $traceId: String!) {
    insights {
      observation(transactionId: $transactionId, traceId: $traceId) {
        ${OBSERVATION_SUMMARY_FIELDS}
        rawOtlp
      }
    }
  }
`;

export const GET_INSIGHTS_FINDINGS = gql`
  query GetInsightsFindings($windowHours: Int, $service: String, $namespace: String, $status: String, $kind: InsightsFindingKind) {
    insights {
      findings(windowHours: $windowHours, service: $service, namespace: $namespace, status: $status, kind: $kind) { ${FINDING_FIELDS} }
    }
  }
`;

export const GET_INSIGHTS_ANOMALIES = gql`
  query GetInsightsAnomalies($windowHours: Int, $service: String, $namespace: String, $status: String) {
    insights {
      anomalies(windowHours: $windowHours, service: $service, namespace: $namespace, status: $status) { ${ANOMALY_SUMMARY_FIELDS} }
    }
  }
`;

const OBSERVATION_FIELDS = `
  ${OBSERVATION_SUMMARY_FIELDS}
  rawOtlp
`;

const ANOMALY_CLASS_FINDING_FIELDS = `
  class
  kind
  title
  summary
  spanHighlights {
    spanId
    severity
    reason
  }
  attrHighlights {
    spanId
    key
    value
  }
  metric {
    observed
    baseline
    unit
  }
  rawEvidence
`;

export const GET_INSIGHTS_ANOMALY = gql`
  query GetInsightsAnomaly($transactionId: ID!, $signature: String!) {
    insights {
      anomaly(transactionId: $transactionId, signature: $signature) {
        ${ANOMALY_SUMMARY_FIELDS}
        evidence
        risk { ${RISK_ASSESSMENT_FIELDS} }
        classFindings { ${ANOMALY_CLASS_FINDING_FIELDS} }
        anomalyTrace { ${OBSERVATION_FIELDS} }
        baselineTrace { ${OBSERVATION_FIELDS} }
      }
    }
  }
`;

export const GET_INSIGHTS_POLICIES = gql`
  query GetInsightsPolicies {
    insights {
      policies { ${POLICY_FIELDS} }
    }
  }
`;

export const GET_INSIGHTS_LEARNING_POLICIES = gql`
  query GetInsightsLearningPolicies {
    insights {
      learningPolicies { ${LEARNING_POLICY_FIELDS} }
    }
  }
`;

export const GET_INSIGHTS_GUARDRAILS = gql`
  query GetInsightsGuardrails {
    insights {
      guardrails { ${GUARDRAIL_FIELDS} }
    }
  }
`;

export const GET_INSIGHTS_GUARDRAIL_VIOLATIONS = gql`
  query GetInsightsGuardrailViolations($windowHours: Int, $service: String, $namespace: String, $status: String) {
    insights {
      guardrailViolations(windowHours: $windowHours, service: $service, namespace: $namespace, status: $status) { ${GUARDRAIL_VIOLATION_FIELDS} }
    }
  }
`;

export const GET_INSIGHTS_GUARDRAIL_VIOLATION = gql`
  query GetInsightsGuardrailViolation($scopeKey: String!, $ruleKey: String!, $offending: String!) {
    insights {
      guardrailViolation(scopeKey: $scopeKey, ruleKey: $ruleKey, offending: $offending) {
        ${GUARDRAIL_VIOLATION_FIELDS}
        summary
        spanHighlights {
          spanId
          severity
          reason
        }
        evidenceTrace { ${OBSERVATION_FIELDS} }
      }
    }
  }
`;

export const GET_INSIGHTS_CATALOG = gql`
  query GetInsightsCatalog {
    insights {
      catalog { ${CATALOG_FIELDS} }
    }
  }
`;

export const GET_INSIGHTS_SYSTEM_SETTINGS = gql`
  query GetInsightsSystemSettings {
    insights {
      systemSettings { ${SYSTEM_SETTINGS_FIELDS} }
    }
  }
`;

export const GET_INSIGHTS_STORAGE_HEALTH = gql`
  query GetInsightsStorageHealth {
    insights {
      storageHealth { ${STORAGE_HEALTH_FIELDS} }
    }
  }
`;
