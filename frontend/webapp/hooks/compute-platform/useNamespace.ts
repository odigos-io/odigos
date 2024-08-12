import { useMutation, useQuery } from '@apollo/client';
import { GET_NAMESPACES, PERSIST_NAMESPACE } from '@/graphql';
import {
  ComputePlatform,
  K8sActualNamespace,
  PersistNamespaceItemInput,
} from '@/types';

type UseNamespaceHook = {
  data?: K8sActualNamespace;
  loading: boolean;
  error?: Error;
  persistNamespace: (namespace: PersistNamespaceItemInput) => Promise<void>;
};

export const useNamespace = (
  namespaceName: string | undefined
): UseNamespaceHook => {
  const { data, loading, error } = useQuery<ComputePlatform>(GET_NAMESPACES, {
    skip: !namespaceName,
    variables: { namespaceName },
    fetchPolicy: 'cache-first',
  });

  const [persistNamespaceMutation] = useMutation(PERSIST_NAMESPACE);

  const persistNamespace = async (namespace: PersistNamespaceItemInput) => {
    try {
      await persistNamespaceMutation({
        variables: { namespace },
      });
    } catch (e) {
      console.error('Error persisting namespace:', e);
    }
  };

  return {
    persistNamespace,
    data: data?.computePlatform.k8sActualNamespace,
    loading,
    error,
  };
};
