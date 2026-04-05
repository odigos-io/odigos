import { gql } from '@apollo/client';

export const ENABLE_SOURCE_PROFILING = gql`
  mutation EnableSourceProfiling($namespace: String!, $kind: String!, $name: String!) {
    enableSourceProfiling(namespace: $namespace, kind: $kind, name: $name) {
      status
      sourceKey
      maxSlots
      activeSlots
    }
  }
`;

export const RELEASE_SOURCE_PROFILING = gql`
  mutation ReleaseSourceProfiling($namespace: String!, $kind: String!, $name: String!) {
    releaseSourceProfiling(namespace: $namespace, kind: $kind, name: $name) {
      status
      sourceKey
      activeSlots
    }
  }
`;
