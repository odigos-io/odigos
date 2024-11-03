import { useAppStore } from '@/store';
import { DestinationInput } from '@/types';
import { useActualSources } from '../sources';
import { useState, useCallback } from 'react';
import { useDestinationCRUD } from '../destinations';

type ConnectEnvResult = {
  success: boolean;
  destinationId?: string;
};

export const useConnectEnv = () => {
  const { createDestination } = useDestinationCRUD();
  const { createSourcesForNamespace, persistNamespaceItems } = useActualSources();

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
        // Persist namespaces based on namespaceFutureSelectAppsList
        const namespaceItems = Object.entries(namespaceFutureSelectAppsList).map(([namespaceName, futureSelected]) => ({
          name: namespaceName,
          futureSelected,
        }));

        await persistNamespaceItems(namespaceItems);

        // Create sources for each namespace in sourcesList
        for (const namespaceName in sourcesList) {
          const sources = sourcesList[namespaceName].map((source) => ({
            kind: source.kind,
            name: source.name,
            selected: true,
          }));
          await createSourcesForNamespace(namespaceName, sources);
        }

        resetSources();

        // Create destination
        const { data } = await createDestination(destination);
        const destinationId = data?.createNewDestination.id;

        if (!destinationId) {
          throw new Error('Error creating destination.');
        }
        callback && callback();
        setResult({
          success: true,
          destinationId,
        });
      } catch (err) {
        setError((err as Error).message);
        setResult({
          success: false,
        });
      } finally {
        setLoading(false);
      }
    },
    [sourcesList, createDestination, persistNamespaceItems, createSourcesForNamespace, namespaceFutureSelectAppsList]
  );

  return {
    connectEnv,
    result,
    loading,
    error,
  };
};
