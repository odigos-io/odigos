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

export const UPSERT_INSIGHTS_POLICY = gql`
  mutation UpsertInsightsPolicy($policy: InsightsPolicyInput!) {
    upsertInsightsPolicy(policy: $policy) {
      ${INSIGHTS_POLICY_FIELDS}
    }
  }
`;

export const DELETE_INSIGHTS_POLICY = gql`
  mutation DeleteInsightsPolicy($scope: InsightsPolicyScope!, $scopeKey: String!) {
    deleteInsightsPolicy(scope: $scope, scopeKey: $scopeKey)
  }
`;
