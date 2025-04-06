import { gql } from '@apollo/client';

export const GET_SOURCES = gql`
  query GetSources($nextPage: String!, $groupName: String!) {
    computePlatform {
      sources(nextPage: $nextPage, groupName: $groupName) {
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
  query GetSource($sourceId: K8sSourceId!, $groupName: String!) {
    computePlatform {
      source(sourceId: $sourceId, groupName: $groupName) {
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
