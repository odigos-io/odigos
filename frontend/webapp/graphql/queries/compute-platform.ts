import { gql } from '@apollo/client';

export const GET_COMPUTE_PLATFORM = gql`
  query GetComputePlatform {
    computePlatform {
      name
      k8sActualNamespaces {
        name
        k8sActualSources {
          kind
          name
          numberOfInstances
        }
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
      k8sActualSources {
        name
      }
    }
  }
`;
