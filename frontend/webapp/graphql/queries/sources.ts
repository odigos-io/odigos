import { gql } from '@apollo/client';
export const GET_SOURCES = gql`
  query GetSources {
    actualSources {
      name
      serviceName
      instrumentedApplicationDetails {
        containers {
          language
          containerName
        }
        conditions {
          type
          status
          message
        }
      }
    }
  }
`;
