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

export const DISABLE_SOURCE_PROFILING = gql`
  mutation DisableSourceProfiling($namespace: String!, $kind: String!, $name: String!) {
    disableSourceProfiling(namespace: $namespace, kind: $kind, name: $name) {
      status
      sourceKey
      activeSlots
    }
  }
`;
