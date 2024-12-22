import { gql } from '@apollo/client';

export const GET_DESTINATION_TYPE = gql`
  query GetDestinationType {
    destinationTypes {
      categories {
        name
        items {
          type
          testConnectionSupported
          displayName
          imageUrl
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
      }
    }
  }
`;

export const GET_DESTINATION_TYPE_DETAILS = gql`
  query GetDestinationTypeDetails($type: String!) {
    destinationTypeDetails(type: $type) {
      fields {
        name
        displayName
        componentType
        componentProperties
        initialValue
        renderCondition
      }
    }
  }
`;

export const GET_POTENTIAL_DESTINATIONS = gql`
  query GetPotentialDestinations {
    potentialDestinations {
      type
      fields
    }
  }
`;
