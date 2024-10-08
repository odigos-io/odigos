import { useState, useCallback } from 'react';
import { useCreateSource } from '../sources';
import { useNamespace } from '../compute-platform';
import { useUpdateSource } from './useUpdateSource';
import { useComputePlatform } from '../compute-platform';
import { PatchSourceRequestInput, WorkloadId } from '@/types';

export function useActualSources() {
  const { data, refetch } = useComputePlatform();
  const { createSource, error: sourceError } = useCreateSource();
  const { updateSource, error: updateError } = useUpdateSource();

  const { persistNamespace } = useNamespace(undefined);
  const [isPolling, setIsPolling] = useState(false);

  const startPolling = useCallback(async () => {
    setIsPolling(true);
    const maxRetries = 5;
    const retryInterval = 1000; // Poll every second
    let retries = 0;

    while (retries < maxRetries) {
      await new Promise((resolve) => setTimeout(resolve, retryInterval));
      refetch();
      retries++;
    }

    setIsPolling(false);
  }, [refetch]);

  const createSourcesForNamespace = async (namespaceName, sources) => {
    await createSource(namespaceName, sources);

    startPolling();
    if (sourceError) {
      throw new Error(`Error creating sources for namespace: ${namespaceName}`);
    }
  };

  const updateActualSource = async (
    sourceId: WorkloadId,
    patchRequest: PatchSourceRequestInput
  ) => {
    try {
      await updateSource(sourceId, patchRequest);
      refetch();
    } catch (error) {
      console.error('Error updating source:', error);
      throw error;
    }
  };

  const persistNamespaceItems = async (namespaceItems) => {
    for (const namespace of namespaceItems) {
      await persistNamespace(namespace);
    }
  };

  return {
    sources: data?.computePlatform.k8sActualSources || [],
    createSourcesForNamespace,
    persistNamespaceItems,
    updateActualSource,
    isPolling,
  };
}
