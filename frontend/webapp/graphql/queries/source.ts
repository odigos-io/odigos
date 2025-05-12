import { gql } from '@apollo/client';

export const GET_SOURCES = gql`
  query GetSources($nextPage: String!, $streamName: String!) {
    computePlatform {
      sources(nextPage: $nextPage, streamName: $streamName) {
        nextPage
        items {
          namespace
          name
          kind
          streamNames
          selected
          otelServiceName
          containers {
            containerName
            language
            runtimeVersion
            instrumented
            instrumentationMessage
            otelDistroName
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
    }
  }
`;

export const GET_SOURCE = gql`
  query GetSource($sourceId: K8sSourceId!, $streamName: String!) {
    computePlatform {
      source(sourceId: $sourceId, streamName: $streamName) {
        namespace
        name
        kind
        streamNames
        selected
        otelServiceName
        containers {
          containerName
          language
          runtimeVersion
          instrumented
          instrumentationMessage
          otelDistroName
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
  }
`;

export const GET_INSTANCES = gql`
  query GetInstrumentationInstancesHealth {
    instrumentationInstancesHealth {
      namespace
      name
      kind
      condition {
        status
        type
        reason
        message
        lastTransitionTime
      }
    }
  }
`;
