import { useState, useCallback } from 'react';
import { useCreateSource } from '../sources';
import { useNamespace } from '../compute-platform';
import { resetSources, useAppStore } from '@/store';
import { useCreateDestination } from '../destinations';
import { DestinationInput, PersistNamespaceItemInput } from '@/types';

type ConnectEnvResult = {
  success: boolean;
  destinationId?: string;
};

export const useConnectEnv = () => {
  const {
    createSource,
    success: sourceSuccess,
    loading: sourceLoading,
    error: sourceError,
  } = useCreateSource();
  const { createNewDestination } = useCreateDestination();
  const { persistNamespace } = useNamespace(undefined);

  const [result, setResult] = useState<ConnectEnvResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sourcesList = useAppStore((state) => state.sources);
  const resetSources = useAppStore((state) => state.resetSources);
  const namespaceFutureSelectAppsList = useAppStore(
    (state) => state.namespaceFutureSelectAppsList
  );

  const connectEnv = useCallback(
    async (destination: DestinationInput, callback?: () => void) => {
      setLoading(true);
      setError(null);
      setResult(null);

      try {
        // Persist namespaces based on namespaceFutureSelectAppsList
        for (const namespaceName in namespaceFutureSelectAppsList) {
          const futureSelected = namespaceFutureSelectAppsList[namespaceName];

          const namespace: PersistNamespaceItemInput = {
            name: namespaceName,
            futureSelected,
          };

          await persistNamespace(namespace);
        }

        // Create sources for each namespace in sourcesList
        for (const namespaceName in sourcesList) {
          const sources = sourcesList[namespaceName].map((source) => ({
            kind: source.kind,
            name: source.name,
            selected: true,
          }));
          await createSource(namespaceName, sources);

          if (sourceError) {
            throw new Error(
              `Error creating sources for namespace: ${namespaceName}`
            );
          }
        }
        resetSources();
        // Create destination
        const destinationId = await createNewDestination(destination);

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
    [
      createSource,
      createNewDestination,
      persistNamespace,
      sourceError,
      sourcesList,
      namespaceFutureSelectAppsList,
    ]
  );

  return {
    connectEnv,
    result,
    loading,
    error,
  };
};
