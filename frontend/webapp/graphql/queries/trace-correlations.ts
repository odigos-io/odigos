import { gql } from '@apollo/client';

export const GET_TRACE_CORRELATIONS = gql`
  query GetTraceCorrelations($filter: WorkloadFilter, $timeRange: TraceCorrelationsTimeRangeInput) {
    traceCorrelations(filter: $filter, timeRange: $timeRange) {
      workloads {
        namespace
        kind
        name
        containerName
        telemetrySdkLanguage
        processRuntimeName
        processRuntimeVersion
        inputs {
          attributes {
            key
            value
          }
          outputs {
            attributes {
              key
              value
            }
            connectionCount
            firstDetectedAt
          }
        }
      }
    }
  }
`;
