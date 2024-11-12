<<<<<<< HEAD
import { useCallback } from 'react';
=======
import { useCallback, useMemo } from 'react';
import { safeJsonParse } from '@/utils';
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
import { useQuery } from '@apollo/client';
import { useBooleanStore } from '@/store';
import type { ComputePlatform } from '@/types';
import { GET_COMPUTE_PLATFORM } from '@/graphql';
<<<<<<< HEAD
=======
import { useFilterStore } from '@/store/useFilterStore';
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866

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
<<<<<<< HEAD
=======
  const filters = useFilterStore();
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866

  const startPolling = useCallback(async () => {
    togglePolling(true);

<<<<<<< HEAD
    const maxRetries = 5;
    const retryInterval = 1000; // Poll every second
    let retries = 0;
=======
    let retries = 0;
    const maxRetries = 5;
    const retryInterval = 1 * 1000; // time in milliseconds
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866

    while (retries < maxRetries) {
      await new Promise((resolve) => setTimeout(resolve, retryInterval));
      refetch();
      retries++;
    }

    togglePolling(false);
  }, [refetch, togglePolling]);

<<<<<<< HEAD
  return { data, loading, error, refetch, startPolling };
=======
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
    if (!!filters.errors.length) {
      k8sActualSources = k8sActualSources.filter((source) => !!filters.errors.find((error) => !!source.instrumentedApplicationDetails?.conditions?.find((cond) => cond.type === error.id)));
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
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
};
