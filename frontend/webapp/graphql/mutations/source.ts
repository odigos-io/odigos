import { gql } from '@apollo/client';

export const CREATE_SOURCE = gql`
  mutation PersistSources(
    $namespace: String!
    $sources: [PersistNamespaceSourceInput!]!
  ) {
    persistK8sSources(namespace: $namespace, sources: $sources)
  }
`;
