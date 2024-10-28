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
          }
          conditions {
            type
            status
            message
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
      }
      actions {
        id
        type
        spec
      }
      k8sActualNamespaces {
        name
      }
    }
  }
`;

export const GET_NAMESPACES = gql`
  query GetK8sActualNamespace(
    $namespaceName: String!
    $instrumentationLabeled: Boolean
  ) {
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
