import { useMutation, useQuery } from '@apollo/client';
import { GET_NAMESPACES, PERSIST_NAMESPACE } from '@/graphql';
import { ComputePlatform, PersistNamespaceItemInput } from '@/types';
import { NOTIFICATION } from '@/utils';
import { useNotify } from '../notification';

export const useNamespace = (namespaceName?: string, instrumentationLabeled = null as boolean | null) => {
  const notify = useNotify();

  const handleError = (title: string, message: string) => {
    notify({ type: NOTIFICATION.ERROR, title, message });
  };

  const handleComplete = (title: string, message: string) => {
    notify({ type: NOTIFICATION.SUCCESS, title, message });
  };

  const { data, loading, error } = useQuery<ComputePlatform>(GET_NAMESPACES, {
    skip: !namespaceName,
    fetchPolicy: 'cache-first',
    variables: { namespaceName, instrumentationLabeled },
  });

  const [persistNamespaceMutation] = useMutation(PERSIST_NAMESPACE, {
    onError: (error) => handleError('', error.message),
    onCompleted: (res, req) => {},
  });

  return {
    persistNamespace: async (namespace: PersistNamespaceItemInput) => await persistNamespaceMutation({ variables: { namespace } }),
    data: data?.computePlatform.k8sActualNamespace,
    loading,
    error,
  };
};
