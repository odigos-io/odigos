import { useCallback } from 'react';
import { useLazyQuery, useMutation, useQuery } from '@apollo/client';
import { GET_PROFILING_SLOTS, GET_SOURCE_PROFILING, ENABLE_SOURCE_PROFILING, RELEASE_SOURCE_PROFILING } from '@/graphql';

interface SourceIdentifier {
  namespace: string;
  kind: string;
  name: string;
}

interface ProfilingSlots {
  activeKeys: string[];
  keysWithData: string[];
  totalBytesUsed: number;
  slotMaxBytes: number;
  maxSlots: number;
  maxTotalBytesBudget: number;
  slotTtlSeconds: number;
}

interface EnableProfilingResult {
  status: string;
  sourceKey: string;
  maxSlots: number;
  activeSlots: number;
}

interface ReleaseProfilingResult {
  status: string;
  sourceKey: string;
  activeSlots: number;
}

interface SourceProfilingResult {
  profileJson: string;
}

interface UseProfiler {
  slots: ProfilingSlots | undefined;
  slotsLoading: boolean;
  refetchSlots: () => void;
  enableProfiling: (source: SourceIdentifier) => Promise<EnableProfilingResult | undefined>;
  releaseProfiling: (source: SourceIdentifier) => Promise<ReleaseProfilingResult | undefined>;
  fetchSourceProfiling: (source: SourceIdentifier) => Promise<SourceProfilingResult | undefined>;
}

export const useProfiling = (pollSlots?: number): UseProfiler => {
  const {
    data: slotsData,
    loading: slotsLoading,
    refetch: refetchSlots,
  } = useQuery<{ profilingSlots: ProfilingSlots }>(GET_PROFILING_SLOTS, {
    pollInterval: pollSlots,
  });

  const [querySourceProfiling] = useLazyQuery<{ sourceProfiling: SourceProfilingResult }, SourceIdentifier>(GET_SOURCE_PROFILING, {
    fetchPolicy: 'network-only',
  });

  const [mutateEnable] = useMutation<{ enableSourceProfiling: EnableProfilingResult }, SourceIdentifier>(ENABLE_SOURCE_PROFILING);
  const [mutateRelease] = useMutation<{ releaseSourceProfiling: ReleaseProfilingResult }, SourceIdentifier>(RELEASE_SOURCE_PROFILING);

  const enableProfiling: UseProfiler['enableProfiling'] = useCallback(
    async (source) => {
      const { data } = await mutateEnable({ variables: source });
      return data?.enableSourceProfiling;
    },
    [mutateEnable],
  );

  const releaseProfiling: UseProfiler['releaseProfiling'] = useCallback(
    async (source) => {
      const { data } = await mutateRelease({ variables: source });
      return data?.releaseSourceProfiling;
    },
    [mutateRelease],
  );

  const fetchSourceProfiling: UseProfiler['fetchSourceProfiling'] = useCallback(
    async (source) => {
      const { data } = await querySourceProfiling({ variables: source });
      return data?.sourceProfiling;
    },
    [querySourceProfiling],
  );

  return {
    slots: slotsData?.profilingSlots,
    slotsLoading,
    refetchSlots,
    enableProfiling,
    releaseProfiling,
    fetchSourceProfiling,
  };
};
