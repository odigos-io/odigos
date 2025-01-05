import { gql } from '@apollo/client';

export const GET_COMPUTE_PLATFORM = gql`
  query GetComputePlatform {
    computePlatform {
      k8sActualNamespaces {
        name
        selected
      }
      k8sActualSources {
        namespace
        name
        kind
        numberOfInstances
        selected
        reportedName
        containers {
          containerName
          language
          runtimeVersion
          otherAgent
        }
        conditions {
          status
          type
          reason
          message
          lastTransitionTime
        }
      }
      destinations {
        id
        name
        fields
        exportedSignals {
          logs
          metrics
          traces
        }
        destinationType {
          type
          imageUrl
          displayName
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
        conditions {
          type
          status
          message
        }
      }
      actions {
        id
        type
        spec
        conditions {
          status
          type
          reason
          message
          lastTransitionTime
        }
      }
      instrumentationRules {
        ruleId
        ruleName
        notes
        disabled
        # workloads {}
        # instrumentationLibraries {}
        payloadCollection {
          httpRequest {
            mimeTypes
            maxPayloadLength
            dropPartialPayloads
          }
          httpResponse {
            mimeTypes
            maxPayloadLength
            dropPartialPayloads
          }
          dbQuery {
            maxPayloadLength
            dropPartialPayloads
          }
          messaging {
            maxPayloadLength
            dropPartialPayloads
          }
        }
      }
    }
  }
`;

export const GET_NAMESPACES = gql`
  query GetK8sActualNamespace($namespaceName: String!) {
    computePlatform {
      k8sActualNamespace(name: $namespaceName) {
        name
        selected
        k8sActualSources {
          kind
          name
          numberOfInstances
          selected
        }
      }
    }
  }
`;
