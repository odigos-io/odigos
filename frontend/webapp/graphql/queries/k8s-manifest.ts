import { gql } from '@apollo/client';

export const GET_K8S_MANIFEST = gql`
  query GetK8sManifest($namespace: String!, $kind: K8sResourceKind!, $name: String!) {
    k8sManifest(namespace: $namespace, kind: $kind, name: $name)
  }
`;
