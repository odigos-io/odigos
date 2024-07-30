import { gql } from '@apollo/client';

export const GET_DESTINATION_TYPE = gql`
  query GetDestinationType {
    destinationTypes {
      categories {
        name
        items {
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
