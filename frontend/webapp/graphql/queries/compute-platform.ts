import { gql } from '@apollo/client';

export const GET_COMPUTE_PLATFORM = gql`
  query GetComputePlatform($cpId: ID!) {
    computePlatform(cpId: $cpId) {
      id
      name
      computePlatformType
      k8sActualNamespaces {
        name
        k8sActualSources {
          name
          kind
          numberOfInstances
        }
      }
    }
  }
`;
