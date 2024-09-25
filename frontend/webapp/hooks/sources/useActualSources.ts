import { useCreateSource } from '../sources';
import { useNamespace } from '../compute-platform';
import { PersistNamespaceItemInput } from '@/types';
import { useComputePlatform } from '../compute-platform';

export function useActualSources() {
  const { data } = useComputePlatform();
  const {
    createSource,
    success: sourceSuccess,
    error: sourceError,
  } = useCreateSource();
  const { persistNamespace } = useNamespace(undefined);

  const createSourcesForNamespace = async (
    namespaceName: string,
    sources: any[]
  ) => {
    await createSource(namespaceName, sources);
    if (sourceError) {
      throw new Error(`Error creating sources for namespace: ${namespaceName}`);
    }
  };

  const persistNamespaceItems = async (
    namespaceItems: PersistNamespaceItemInput[]
  ) => {
    for (const namespace of namespaceItems) {
      await persistNamespace(namespace);
    }
  };

  return {
    sources: data?.computePlatform.k8sActualSources || [],
    createSourcesForNamespace,
    persistNamespaceItems,
  };
}
