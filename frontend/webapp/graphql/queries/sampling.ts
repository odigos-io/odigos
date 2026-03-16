import { gql } from '@apollo/client';

const SOURCES_SCOPE_FIELDS = `
  workloadName
  workloadKind
  workloadNamespace
  workloadLanguage
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
