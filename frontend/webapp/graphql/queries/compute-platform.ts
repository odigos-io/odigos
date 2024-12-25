import { gql } from '@apollo/client';

export const GET_COMPUTE_PLATFORM = gql`
  query GetComputePlatform {
    computePlatform {
      k8sActualSources {
        name
        kind
        namespace
        numberOfInstances
        reportedName
        instrumentedApplicationDetails {
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
        status {
          conditions {
            status
            type
            reason
            message
            lastTransitionTime
          }
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
      k8sActualNamespaces {
        name
      }
    }
  }
`;

export const GET_NAMESPACES = gql`
  query GetK8sActualNamespace($namespaceName: String!, $instrumentationLabeled: Boolean) {
    computePlatform {
      k8sActualNamespace(name: $namespaceName) {
        name
        instrumentationLabelEnabled
        k8sActualSources(instrumentationLabeled: $instrumentationLabeled) {
          kind
          name
          numberOfInstances
        }
      }
    }
  }
`;
