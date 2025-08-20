import { gql } from '@apollo/client';

export const GET_TRACES = gql`
  query GetTraces($serviceName: String!) {
    getTraces(serviceName: $serviceName) {
      traceID
      spans {
        traceID
        spanID
        operationName
        references {
          refType
          traceID
          spanID
        }
        startTime
        duration
        tags {
          key
          type
          value
        }
        logs {
          timestamp
          fields {
            key
            type
            value
          }
        }
        processID
        warnings
      }
      processes {
        serviceName
        tags {
          key
          type
          value
        }
      }
      warnings
    }
  }
`;
