import { gql } from '@apollo/client';

const INSIGHTS_POLICY_FIELDS = `
  id
  name
  enabled
  fireAtScore
  signalWeights
  enricherLists
  scope
  scopeKey
`;

export const GET_INSIGHTS_POLICIES = gql`
  query GetInsightsPolicies {
    insights {
      policies {
        ${INSIGHTS_POLICY_FIELDS}
      }
    }
  }
`;
