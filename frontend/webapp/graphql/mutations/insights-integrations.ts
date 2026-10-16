import { gql } from '@apollo/client';
import { INSIGHTS_INTEGRATION_FIELDS } from '../integrations';

export const CREATE_INSIGHTS_INTEGRATION = gql`mutation CreateInsightsIntegration($integration: InsightsIntegrationInput!) { createInsightsIntegration(integration: $integration) { ${INSIGHTS_INTEGRATION_FIELDS} } }`;

export const UPDATE_INSIGHTS_INTEGRATION = gql`mutation UpdateInsightsIntegration($id: ID!, $integration: InsightsIntegrationInput!) { updateInsightsIntegration(id: $id, integration: $integration) { ${INSIGHTS_INTEGRATION_FIELDS} } }`;

export const DELETE_INSIGHTS_INTEGRATION = gql`
  mutation DeleteInsightsIntegration($id: ID!) {
    deleteInsightsIntegration(id: $id)
  }
`;

export const TEST_INSIGHTS_INTEGRATION = gql`
  mutation TestInsightsIntegration($id: ID!) {
    testInsightsIntegration(id: $id)
  }
`;

export const RETRY_INSIGHTS_INTEGRATION_DELIVERIES = gql`
  mutation RetryInsightsIntegrationDeliveries($id: ID!, $findingKeys: [String!]) {
    retryInsightsIntegrationDeliveries(id: $id, findingKeys: $findingKeys)
  }
`;
