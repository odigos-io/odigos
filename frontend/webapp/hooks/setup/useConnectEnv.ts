import { useAppStore } from '@/store';
import { DestinationInput } from '@/types';
import { useSourceCRUD } from '../sources';
import { useState, useCallback } from 'react';
import { useDestinationCRUD } from '../destinations';

type ConnectEnvResult = {
  success: boolean;
  destinationId?: string;
};

export const useConnectEnv = () => {
  const { createSources } = useSourceCRUD();
  const { createDestination } = useDestinationCRUD();

  const [result, setResult] = useState<ConnectEnvResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sourcesList = useAppStore((state) => state.sources);
  const resetSources = useAppStore((state) => state.resetSources);
  const namespaceFutureSelectAppsList = useAppStore((state) => state.namespaceFutureSelectAppsList);

  const connectEnv = useCallback(
    async (destination: DestinationInput, callback?: () => void) => {
      setLoading(true);
      setError(null);
      setResult(null);

      try {
        await createSources(sourcesList, namespaceFutureSelectAppsList);
        resetSources();

        const { data } = await createDestination(destination);
        const destinationId = data?.createNewDestination.id;

        callback && callback();
        setResult({ success: true, destinationId });
      } catch (err) {
        setError((err as Error).message);
        setResult({ success: false });
      } finally {
        setLoading(false);
      }
    },
    [sourcesList, namespaceFutureSelectAppsList, createSources, resetSources, createDestination],
  );

  return {
    connectEnv,
    result,
    loading,
    error,
  };
};
