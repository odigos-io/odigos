import { gql } from '@apollo/client';

export const GET_METRICS = gql`
  query GetOverviewMetrics {
    getOverviewMetrics {
      sources {
        namespace
        kind
        name
        totalDataSent
        throughput
      }
      destinations {
        id
        totalDataSent
        throughput
      }
    }
  }
`;
