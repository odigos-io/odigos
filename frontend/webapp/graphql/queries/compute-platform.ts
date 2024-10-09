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
        exportedSignals {
          logs
          metrics
          traces
        }
        destinationType {
          imageUrl
          displayName
        }
      }
      actions {
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
