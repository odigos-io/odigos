import { gql } from '@apollo/client';

export const PERSIST_NAMESPACE = gql`
  mutation PersistNamespace($namespace: PersistNamespaceItemInput!) {
    persistK8sNamespace(namespace: $namespace)
  }
`;
