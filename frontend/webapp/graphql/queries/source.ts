import { gql } from '@apollo/client';

export const GET_SOURCE = gql`
  query GetSource($sourceId: K8sSourceId!) {
    computePlatform {
      source(sourceId: $sourceId) {
        namespace
        name
        kind
        dataStreamNames
        selected
        otelServiceName
        containers {
          containerName
          language
          runtimeVersion
          overriden
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
        manifestYAML
        instrumentationConfigYAML
      }
    }
  }
`;

export const GET_SOURCE_LIBRARIES = gql`
  query GetSourceLibraries($namespace: String!, $kind: String!, $name: String!) {
    instrumentationInstanceComponents(namespace: $namespace, kind: $kind, name: $name) {
      name
      type
      healthy
      message
      lastStatusTime
      nonIdentifyingAttributes {
        key
        value
      }
    }
  }
`;
