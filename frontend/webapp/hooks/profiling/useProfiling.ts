import { useCallback } from 'react';
import { useLazyQuery, useMutation } from '@apollo/client';
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

interface UseProfiling {
  fetchProfilingSlots: () => Promise<ProfilingSlots | undefined>;
  enableProfiling: (source: SourceIdentifier) => Promise<EnableProfilingResult | undefined>;
  releaseProfiling: (source: SourceIdentifier) => Promise<ReleaseProfilingResult | undefined>;
  fetchSourceProfiling: (source: SourceIdentifier) => Promise<SourceProfilingResult | undefined>;
}

export const useProfiling = (): UseProfiling => {
  const [querySlots] = useLazyQuery<{ profilingSlots: ProfilingSlots }>(GET_PROFILING_SLOTS, {
    fetchPolicy: 'network-only',
  });

  const [querySourceProfiling] = useLazyQuery<{ sourceProfiling: SourceProfilingResult }, SourceIdentifier>(GET_SOURCE_PROFILING, {
    fetchPolicy: 'network-only',
  });

  const [mutateEnable] = useMutation<{ enableSourceProfiling: EnableProfilingResult }, SourceIdentifier>(ENABLE_SOURCE_PROFILING);
  const [mutateRelease] = useMutation<{ releaseSourceProfiling: ReleaseProfilingResult }, SourceIdentifier>(RELEASE_SOURCE_PROFILING);

  // Returns buffer/slot diagnostics: which workloads have active slots, which have buffered data, and memory usage.
  // Example response: { activeKeys: ["default/Deployment/inventory", ...], keysWithData: [...], totalBytesUsed: 4897024, ... }
  const fetchProfilingSlots: UseProfiling['fetchProfilingSlots'] = useCallback(async () => {
    const { data } = await querySlots();
    return data?.profilingSlots;
  }, [querySlots]);

  // Activates (or refreshes) a profiling slot for a workload. Must be called before fetchSourceProfiling will return data.
  // Example: await enableProfiling({ namespace: "default", kind: "Deployment", name: "inventory" })
  //       => { status: "ok", sourceKey: "default/Deployment/inventory", maxSlots: 24, activeSlots: 6 }
  const enableProfiling: UseProfiling['enableProfiling'] = useCallback(
    async (source) => {
      const { data } = await mutateEnable({ variables: source });
      return data?.enableSourceProfiling;
    },
    [mutateEnable],
  );

  // Drops the profiling slot and frees buffered OTLP data for a workload (e.g. user closed the profiling panel).
  // Example: await releaseProfiling({ namespace: "default", kind: "Deployment", name: "inventory" })
  //       => { status: "ok", sourceKey: "default/Deployment/inventory", activeSlots: 5 }
  const releaseProfiling: UseProfiling['releaseProfiling'] = useCallback(
    async (source) => {
      const { data } = await mutateRelease({ variables: source });
      return data?.releaseSourceProfiling;
    },
    [mutateRelease],
  );

  // Fetches the aggregated Pyroscope-shaped flame graph for a workload. Returns a JSON-encoded FlamebearerProfile.
  // Example: await fetchSourceProfiling({ namespace: "default", kind: "Deployment", name: "inventory" })
  //       => { profileJson: '{"version":1,"flamebearer":{"names":[...],"levels":[...],...},...}' }
  const fetchSourceProfiling: UseProfiling['fetchSourceProfiling'] = useCallback(
    async (source) => {
      const { data } = await querySourceProfiling({ variables: source });
      return data?.sourceProfiling;
    },
    [querySourceProfiling],
  );

  return {
    fetchProfilingSlots,
    enableProfiling,
    releaseProfiling,
    fetchSourceProfiling,
  };
};
