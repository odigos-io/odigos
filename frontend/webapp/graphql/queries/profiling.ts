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
  query GetSourceProfiling($sourceId: K8sSourceId!, $profileType: String) {
    computePlatform {
      source(sourceId: $sourceId) {
        profiling(profileType: $profileType) {
          profileJson
        }
      }
    }
  }
`;
