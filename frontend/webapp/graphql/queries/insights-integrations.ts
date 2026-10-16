import { gql } from '@apollo/client';
import { INSIGHTS_INTEGRATION_FIELDS } from '../integrations';

export const GET_INSIGHTS_INTEGRATION_CATALOG = gql`
  query IntegrationCatalog {
    insights {
      integrationCatalog {
        type
        name
        description
        docsUrl
        fields {
          key
          label
          type
          required
          help
          placeholder
          default
          options
        }
        capabilities {
          needsHeartbeat
          supportsResolve
          returnsExternalId
          maxBatch
        }
      }
    }
  }
`;

export const GET_INSIGHTS_INTEGRATIONS = gql`query Integrations { insights { integrations { ${INSIGHTS_INTEGRATION_FIELDS} } } }`;

export const GET_INSIGHTS_INTEGRATION = gql`query Integration($id: ID!) { insights { integration(id: $id) { ${INSIGHTS_INTEGRATION_FIELDS} } } }`;

export const GET_INSIGHTS_INTEGRATION_DELIVERIES = gql`
  query IntegrationDeliveries($integrationId: ID!, $state: InsightsDeliveryState, $limit: Int) {
    insights {
      integrationDeliveries(integrationId: $integrationId, state: $state, limit: $limit) {
        integrationId
        findingKey
        lastAction
        state
        attempts
        nextAttemptAt
        lastError
        sentAt
        occurrencesAtSend
        severityAtSend
        statusAtSend
        externalId
        updatedAt
      }
    }
  }
`;

export const GET_INSIGHTS_INTEGRATION_ATTEMPTS = gql`
  query IntegrationAttempts($integrationId: ID!, $limit: Int) {
    insights {
      integrationAttempts(integrationId: $integrationId, limit: $limit) {
        integrationId
        findingKey
        action
        attempt
        outcome
        httpStatus
        error
        durationMs
        eventCount
        attemptedAt
      }
    }
  }
`;

export const GET_INSIGHTS_FINDING_DELIVERIES = gql`
  query FindingDeliveries($findingKey: String!) {
    insights {
      findingDeliveries(findingKey: $findingKey) {
        integrationId
        findingKey
        lastAction
        state
        attempts
        nextAttemptAt
        lastError
        sentAt
        occurrencesAtSend
        severityAtSend
        statusAtSend
        externalId
        updatedAt
      }
    }
  }
`;
