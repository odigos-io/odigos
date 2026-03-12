import { gql } from '@apollo/client';

// ---- Noisy Operations ----

export const CREATE_NOISY_OPERATION_RULE = gql`
  mutation CreateNoisyOperationRule($samplingId: ID!, $rule: NoisyOperationRuleInput!) {
    createNoisyOperationRule(samplingId: $samplingId, rule: $rule) {
      ruleId
      name
      disabled
      sourceScopes { workloadName workloadKind workloadNamespace workloadLanguage }
      operation {
        httpServer { route routePrefix method }
        httpClient { serverAddress templatedPath templatedPathPrefix method }
      }
      percentageAtMost
      notes
    }
  }
`;

export const UPDATE_NOISY_OPERATION_RULE = gql`
  mutation UpdateNoisyOperationRule($samplingId: ID!, $ruleId: ID!, $rule: NoisyOperationRuleInput!) {
    updateNoisyOperationRule(samplingId: $samplingId, ruleId: $ruleId, rule: $rule) {
      ruleId
      name
      disabled
      sourceScopes { workloadName workloadKind workloadNamespace workloadLanguage }
      operation {
        httpServer { route routePrefix method }
        httpClient { serverAddress templatedPath templatedPathPrefix method }
      }
      percentageAtMost
      notes
    }
  }
`;

export const DELETE_NOISY_OPERATION_RULE = gql`
  mutation DeleteNoisyOperationRule($samplingId: ID!, $ruleId: ID!) {
    deleteNoisyOperationRule(samplingId: $samplingId, ruleId: $ruleId)
  }
`;

// ---- Highly Relevant Operations ----

export const CREATE_HIGHLY_RELEVANT_OPERATION_RULE = gql`
  mutation CreateHighlyRelevantOperationRule($samplingId: ID!, $rule: HighlyRelevantOperationRuleInput!) {
    createHighlyRelevantOperationRule(samplingId: $samplingId, rule: $rule) {
      ruleId
      name
      disabled
      sourceScopes { workloadName workloadKind workloadNamespace workloadLanguage }
      error
      durationAtLeastMs
      operation {
        httpServer { route routePrefix method }
        kafkaConsumer { kafkaTopic }
        kafkaProducer { kafkaTopic }
      }
      percentageAtLeast
      notes
    }
  }
`;

export const UPDATE_HIGHLY_RELEVANT_OPERATION_RULE = gql`
  mutation UpdateHighlyRelevantOperationRule($samplingId: ID!, $ruleId: ID!, $rule: HighlyRelevantOperationRuleInput!) {
    updateHighlyRelevantOperationRule(samplingId: $samplingId, ruleId: $ruleId, rule: $rule) {
      ruleId
      name
      disabled
      sourceScopes { workloadName workloadKind workloadNamespace workloadLanguage }
      error
      durationAtLeastMs
      operation {
        httpServer { route routePrefix method }
        kafkaConsumer { kafkaTopic }
        kafkaProducer { kafkaTopic }
      }
      percentageAtLeast
      notes
    }
  }
`;

export const DELETE_HIGHLY_RELEVANT_OPERATION_RULE = gql`
  mutation DeleteHighlyRelevantOperationRule($samplingId: ID!, $ruleId: ID!) {
    deleteHighlyRelevantOperationRule(samplingId: $samplingId, ruleId: $ruleId)
  }
`;

// ---- Cost Reduction Rules ----

export const CREATE_COST_REDUCTION_RULE = gql`
  mutation CreateCostReductionRule($samplingId: ID!, $rule: CostReductionRuleInput!) {
    createCostReductionRule(samplingId: $samplingId, rule: $rule) {
      ruleId
      name
      disabled
      sourceScopes { workloadName workloadKind workloadNamespace workloadLanguage }
      operation {
        httpServer { route routePrefix method }
        kafkaConsumer { kafkaTopic }
        kafkaProducer { kafkaTopic }
      }
      percentageAtMost
      notes
    }
  }
`;

export const UPDATE_COST_REDUCTION_RULE = gql`
  mutation UpdateCostReductionRule($samplingId: ID!, $ruleId: ID!, $rule: CostReductionRuleInput!) {
    updateCostReductionRule(samplingId: $samplingId, ruleId: $ruleId, rule: $rule) {
      ruleId
      name
      disabled
      sourceScopes { workloadName workloadKind workloadNamespace workloadLanguage }
      operation {
        httpServer { route routePrefix method }
        kafkaConsumer { kafkaTopic }
        kafkaProducer { kafkaTopic }
      }
      percentageAtMost
      notes
    }
  }
`;

export const DELETE_COST_REDUCTION_RULE = gql`
  mutation DeleteCostReductionRule($samplingId: ID!, $ruleId: ID!) {
    deleteCostReductionRule(samplingId: $samplingId, ruleId: $ruleId)
  }
`;
