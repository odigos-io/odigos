import { gql } from '@apollo/client';

export const GET_NAMESPACES = gql`
  query GetNamespaces {
    computePlatform {
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
        sources {
          kind
          name
          numberOfInstances
          selected
        }
      }
    }
  }
`;
