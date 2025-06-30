import { gql } from '@apollo/client';

export const PERSIST_NAMESPACES = gql`
  mutation PersistNamespaces($namespaces: [PersistNamespaceItemInput!]!) {
    persistK8sNamespaces(namespaces: $namespaces)
  }
`;
