import { gql } from '@apollo/client';

export const CREATE_INSTRUMENTATION_RULE = gql`
  mutation CreateInstrumentationRule($instrumentationRule: InstrumentationRuleInput!) {
    createInstrumentationRule(instrumentationRule: $instrumentationRule) {
      ruleId
    }
  }
`;

export const UPDATE_INSTRUMENTATION_RULE = gql`
  mutation UpdateInstrumentationRule($ruleId: ID!, $instrumentationRule: InstrumentationRuleInput!) {
    updateInstrumentationRule(ruleId: $ruleId, instrumentationRule: $instrumentationRule) {
      ruleId
    }
  }
`;

export const DELETE_INSTRUMENTATION_RULE = gql`
  mutation DeleteInstrumentationRule($ruleId: ID!) {
    deleteInstrumentationRule(ruleId: $ruleId)
  }
`;
