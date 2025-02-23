import { gql } from '@apollo/client';

export const GET_COMPUTE_PLATFORM = gql`
  query GetComputePlatform {
    computePlatform {
      computePlatformType
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
    }
  }
`;

export const GET_NAMESPACE = gql`
  query GetNamespace($namespaceName: String!) {
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

export const GET_ACTIONS = gql`
  query GetActions {
    computePlatform {
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
    }
  }
`;

export const GET_INSTRUMENTATION_RULES = gql`
  query GetInstrumentationRules {
    computePlatform {
      instrumentationRules {
        ruleId
        ruleName
        notes
        disabled
        mutable
        profileName
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
        codeAttributes {
          column
          filePath
          function
          lineNumber
          namespace
          stacktrace
        }
      }
    }
  }
`;
