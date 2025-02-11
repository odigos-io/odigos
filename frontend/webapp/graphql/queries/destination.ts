import { gql } from '@apollo/client';

export const GET_DESTINATION_CATEGORIES = gql`
  query GetDestinationCategories {
    destinationCategories {
      categories {
        name
        description
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
          fields {
            name
            displayName
            componentType
            componentProperties
            secret
            initialValue
            renderCondition
            hideFromReadData
            customReadDataLabels {
              condition
              title
              value
            }
          }
        }
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
