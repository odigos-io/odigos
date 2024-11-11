import { useCallback, useMemo } from 'react';
import { safeJsonParse } from '@/utils';
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
  const filters = useFilterStore();

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
    let destinations = [...data.computePlatform.destinations];
    let actions = [...data.computePlatform.actions];

    if (!!filters.namespace) {
      k8sActualSources = k8sActualSources.filter((source) => filters.namespace?.id === source.namespace);
    }
    if (!!filters.types.length) {
      k8sActualSources = k8sActualSources.filter((source) => !!filters.types.find((type) => type.id === source.kind));
    }
    if (!!filters.monitors.length) {
      destinations = destinations.filter((destination) => !!filters.monitors.find((metric) => destination.exportedSignals[metric.id]));
      actions = actions.filter(
        (action) =>
          !!filters.monitors.find((metric) => {
            const { signals } = safeJsonParse(action.spec as string, { signals: [] as string[] });
            return signals.find((str) => str.toLowerCase() === metric.id);
          }),
      );
    }

    return {
      computePlatform: {
        ...data.computePlatform,
        k8sActualSources,
        destinations,
        actions,
      },
    };
  }, [data, filters]);

  return { data: filteredData, loading, error, refetch, startPolling };
};
