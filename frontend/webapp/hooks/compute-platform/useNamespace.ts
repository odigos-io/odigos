import { useNotificationStore } from '@/store';
import { useMutation, useQuery } from '@apollo/client';
import { useComputePlatform } from './useComputePlatform';
import { GET_NAMESPACES, PERSIST_NAMESPACE } from '@/graphql';
import { type ComputePlatform, NOTIFICATION_TYPE, type PersistNamespaceItemInput } from '@/types';

export const useNamespace = (namespaceName?: string, instrumentationLabeled = null as boolean | null) => {
  const { addNotification } = useNotificationStore();
  const cp = useComputePlatform();

  const handleError = (title: string, message: string) => {
    addNotification({ type: NOTIFICATION_TYPE.ERROR, title, message });
  };

  const handleComplete = (title: string, message: string) => {
    addNotification({ type: NOTIFICATION_TYPE.SUCCESS, title, message });
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
    allNamespaces: cp.data?.computePlatform.k8sActualNamespaces,
    persistNamespace: async (namespace: PersistNamespaceItemInput) => await persistNamespaceMutation({ variables: { namespace } }),
    data: data?.computePlatform.k8sActualNamespace,
    loading,
    error,
  };
};
