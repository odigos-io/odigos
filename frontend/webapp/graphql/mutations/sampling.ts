import { gql } from '@apollo/client';

// ---- Noisy Operations ----

export const CREATE_NOISY_OPERATION_RULE = gql`
  mutation CreateNoisyOperationRule($rule: NoisyOperationRuleInput!) {
    createNoisyOperationRule(rule: $rule) {
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
  mutation UpdateNoisyOperationRule($ruleId: ID!, $rule: NoisyOperationRuleInput!) {
    updateNoisyOperationRule(ruleId: $ruleId, rule: $rule) {
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
  mutation DeleteNoisyOperationRule($ruleId: ID!) {
    deleteNoisyOperationRule(ruleId: $ruleId)
  }
`;

// ---- Highly Relevant Operations ----

export const CREATE_HIGHLY_RELEVANT_OPERATION_RULE = gql`
  mutation CreateHighlyRelevantOperationRule($rule: HighlyRelevantOperationRuleInput!) {
    createHighlyRelevantOperationRule(rule: $rule) {
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
  mutation UpdateHighlyRelevantOperationRule($ruleId: ID!, $rule: HighlyRelevantOperationRuleInput!) {
    updateHighlyRelevantOperationRule(ruleId: $ruleId, rule: $rule) {
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
  mutation DeleteHighlyRelevantOperationRule($ruleId: ID!) {
    deleteHighlyRelevantOperationRule(ruleId: $ruleId)
  }
`;

// ---- Cost Reduction Rules ----

export const CREATE_COST_REDUCTION_RULE = gql`
  mutation CreateCostReductionRule($rule: CostReductionRuleInput!) {
    createCostReductionRule(rule: $rule) {
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
  mutation UpdateCostReductionRule($ruleId: ID!, $rule: CostReductionRuleInput!) {
    updateCostReductionRule(ruleId: $ruleId, rule: $rule) {
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
  mutation DeleteCostReductionRule($ruleId: ID!) {
    deleteCostReductionRule(ruleId: $ruleId)
  }
`;
