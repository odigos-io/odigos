import { gql } from '@apollo/client';

export const CREATE_DESTINATION = gql`
  mutation CreateNewDestination($destination: DestinationInput!) {
    createNewDestination(destination: $destination) {
      id
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
