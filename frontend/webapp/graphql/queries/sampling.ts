import { gql } from '@apollo/client';

const SOURCES_SCOPE_FIELDS = `
  sources { namespace kind name }
  namespaces
  languages
`;

const NOISY_OPERATION_FIELDS = `
  ruleId
  name
  disabled
  sourceScopes { ${SOURCES_SCOPE_FIELDS} }
  operation {
    httpServer { route routePrefix method }
    httpClient { serverAddress templatedPath templatedPathPrefix method }
  }
  percentageAtMost
  notes
`;

const HIGHLY_RELEVANT_OPERATION_FIELDS = `
  ruleId
  name
  disabled
  sourceScopes { ${SOURCES_SCOPE_FIELDS} }
  error
  durationAtLeastMs
  operation {
    httpServer { route routePrefix method }
    kafkaConsumer { kafkaTopic }
    kafkaProducer { kafkaTopic }
  }
  percentageAtLeast
  notes
`;

const COST_REDUCTION_RULE_FIELDS = `
  ruleId
  name
  disabled
  sourceScopes { ${SOURCES_SCOPE_FIELDS} }
  operation {
    httpServer { route routePrefix method }
    kafkaConsumer { kafkaTopic }
    kafkaProducer { kafkaTopic }
  }
  percentageAtMost
  notes
`;

export const GET_SAMPLING_RULES = gql`
  query GetSamplingRules {
    sampling {
      configs {
        effective {
          k8sHealthProbesSampling {
            enabled
            keepPercentage
          }
        }
      }
      rules {
        id
        name
        noisyOperations { ${NOISY_OPERATION_FIELDS} }
        highlyRelevantOperations { ${HIGHLY_RELEVANT_OPERATION_FIELDS} }
        costReductionRules { ${COST_REDUCTION_RULE_FIELDS} }
      }
    }
  }
`;

// Returns the computed sampling view for a single source (workload), with the
// cluster-wide rules pre-filtered per container based on each rule's source
// scope (workload identity + container language). Used by the Source Drawer
// "Sampling" tab to show only the rules that actually apply to this source.
export const GET_SOURCE_SAMPLING = gql`
  query GetSourceSampling($workloadId: K8sWorkloadIdInput!) {
    sourceSampling(workloadId: $workloadId) {
      workloadId {
        namespace
        kind
        name
      }
      containers {
        containerName
        noisyOperations { ${NOISY_OPERATION_FIELDS} }
        highlyRelevantOperations { ${HIGHLY_RELEVANT_OPERATION_FIELDS} }
        costReductionRules { ${COST_REDUCTION_RULE_FIELDS} }
      }
    }
  }
`;
