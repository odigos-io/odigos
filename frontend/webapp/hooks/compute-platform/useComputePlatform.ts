import { useCallback } from 'react';
import { useQuery } from '@apollo/client';
import { useBooleanStore } from '@/store';
import type { ComputePlatform } from '@/types';
import { GET_COMPUTE_PLATFORM } from '@/graphql';

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

  const startPolling = useCallback(async () => {
    togglePolling(true);

    const maxRetries = 5;
    const retryInterval = 1000; // Poll every second
    let retries = 0;

    while (retries < maxRetries) {
      await new Promise((resolve) => setTimeout(resolve, retryInterval));
      refetch();
      retries++;
    }

    togglePolling(false);
  }, [refetch, togglePolling]);

  return { data, loading, error, refetch, startPolling };
};
