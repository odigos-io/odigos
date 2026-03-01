import { gql } from '@apollo/client';

export const GET_NAMESPACES = gql`
  query GetNamespaces {
    computePlatform {
      k8sActualNamespaces {
        name
        selected
        dataStreamNames
      }
    }
  }
`;
