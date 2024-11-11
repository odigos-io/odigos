import { useCallback, useMemo } from 'react';
import { useQuery } from '@apollo/client';
import { useBooleanStore } from '@/store';
import type { ComputePlatform } from '@/types';
import { GET_COMPUTE_PLATFORM } from '@/graphql';
import { useFilterStore } from '@/store/useFilterStore';

type UseComputePlatformHook = {
  data?: ComputePlatform;
  loading: boolean;
  error?: Error;
  refetch: () => void;
  startPolling: () => Promise<void>;
};

export const useComputePlatform = (): UseComputePlatformHook => {
  const { data, loading, error, refetch } = useQuery<ComputePlatform>(GET_COMPUTE_PLATFORM);
  const { togglePolling } = useBooleanStore();
  const { namespace } = useFilterStore();

  const startPolling = useCallback(async () => {
    togglePolling(true);

    let retries = 0;
    const maxRetries = 5;
    const retryInterval = 1 * 1000; // time in milliseconds

    while (retries < maxRetries) {
      await new Promise((resolve) => setTimeout(resolve, retryInterval));
      refetch();
      retries++;
    }

    togglePolling(false);
  }, [refetch, togglePolling]);

  const filteredData = useMemo(() => {
    if (!data) return undefined;

    const k8sActualSources = !!namespace ? data.computePlatform.k8sActualSources.filter((source) => source.namespace === namespace.id) : data.computePlatform.k8sActualSources;

    return {
      computePlatform: {
        ...data.computePlatform,
        k8sActualSources,
        // destinations,
        // actions,
        // instrumentationRules,
      },
    };
  }, [data, namespace]);

  return { data: filteredData, loading, error, refetch, startPolling };
};
