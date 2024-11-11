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
  const { namespace, types } = useFilterStore();

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

    let k8sActualSources = [...data.computePlatform.k8sActualSources];

    if (!!namespace) k8sActualSources = k8sActualSources.filter((source) => source.namespace === namespace.id);
    if (!!types.length) k8sActualSources = k8sActualSources.filter((source) => !!types.find((type) => type.id === source.kind));

    return {
      computePlatform: {
        ...data.computePlatform,
        k8sActualSources,
      },
    };
  }, [data, namespace, types]);

  return { data: filteredData, loading, error, refetch, startPolling };
};
