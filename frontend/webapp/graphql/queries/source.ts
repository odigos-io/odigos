import { gql } from '@apollo/client';

export const GET_SOURCES = gql`
  query GetSources($nextPage: String!) {
    computePlatform {
      sources(nextPage: $nextPage) {
        nextPage
        items {
          namespace
          name
          kind
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
  query GetSource($sourceId: K8sSourceId!) {
    computePlatform {
      source(sourceId: $sourceId) {
        namespace
        name
        kind
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
  query GetInstrumentationInstancesHealth($sourceIds: [K8sSourceId!]!) {
    instrumentationInstancesHealth(sourceIds: $sourceIds) {
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
