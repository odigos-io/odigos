import { gql } from '@apollo/client';

export const CREATE_DESTINATION = gql`
  mutation CreateNewDestination($destination: DestinationInput!) {
    createNewDestination(destination: $destination) {
      id
      name
      disabled
      dataStreamNames
      fields
      exportedSignals {
        logs
        metrics
        traces
      }
      destinationType {
        type
        imageUrl
        displayName
        supportedSignals {
          logs {
            supported
          }
          metrics {
            supported
          }
          traces {
            supported
          }
        }
      }
      conditions {
        status
        type
        reason
        message
        lastTransitionTime
      }
    }
  }
`;

export const TEST_CONNECTION_MUTATION = gql`
  mutation TestConnection($destination: DestinationInput!) {
    testConnectionForDestination(destination: $destination) {
      succeeded
      statusCode
      destinationType
      message
      reason
    }
  }
`;

export const UPDATE_DESTINATION = gql`
  mutation UpdateDestination($id: ID!, $destination: DestinationInput!) {
    updateDestination(id: $id, destination: $destination) {
      id
    }
  }
`;

export const DELETE_DESTINATION = gql`
  mutation DeleteDestination($id: ID!, $currentStreamName: String!) {
    deleteDestination(id: $id, currentStreamName: $currentStreamName)
  }
`;
