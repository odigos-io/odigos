import { gql } from '@apollo/client';

export const GET_COMPUTE_PLATFORM = gql`
  query GetComputePlatform {
    computePlatform {
      apiTokens {
        token
        name
        issuedAt
        expiresAt
      }
      k8sActualNamespaces {
        name
        selected
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
  query GetK8sActualNamespace($namespaceName: String!, $instrumentationLabeled: Boolean) {
    computePlatform {
      k8sActualNamespace(name: $namespaceName) {
        name
        selected
        k8sActualSources(instrumentationLabeled: $instrumentationLabeled) {
          kind
          name
          numberOfInstances
          selected
        }
      }
    }
  }
`;

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
      }
    }
  }
`;
