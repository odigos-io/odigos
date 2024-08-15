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
  query GetK8sActualNamespace($namespaceName: String!) {
    computePlatform {
      k8sActualNamespace(name: $namespaceName) {
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
