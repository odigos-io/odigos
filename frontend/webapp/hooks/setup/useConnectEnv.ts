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
  const { configuredSources, configuredFutureApps, resetSources } = useAppStore((state) => state);

  const [result, setResult] = useState<ConnectEnvResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const connectEnv = useCallback(
    async (destination: DestinationInput, callback?: () => void) => {
      setLoading(true);
      setError(null);
      setResult(null);

      try {
        await createSources(configuredSources, configuredFutureApps);
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
    [configuredSources, configuredFutureApps, createSources, resetSources, createDestination],
  );

  return {
    connectEnv,
    result,
    loading,
    error,
  };
};
