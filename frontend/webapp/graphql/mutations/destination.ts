import { gql } from '@apollo/client';

export const CREATE_DESTINATION = gql`
  mutation CreateNewDestination($destination: DestinationInput!) {
    createNewDestination(destination: $destination) {
      id
    }
  }
`;
