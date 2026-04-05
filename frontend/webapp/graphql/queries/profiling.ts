import { gql } from '@apollo/client';

export const GET_PROFILING_SLOTS = gql`
  query GetProfilingSlots {
    profilingSlots {
      activeKeys
      keysWithData
      totalBytesUsed
      slotMaxBytes
      maxSlots
      maxTotalBytesBudget
      slotTtlSeconds
    }
  }
`;

export const GET_SOURCE_PROFILING = gql`
  query GetSourceProfiling($namespace: String!, $kind: String!, $name: String!) {
    sourceProfiling(namespace: $namespace, kind: $kind, name: $name) {
      profileJson
    }
  }
`;
