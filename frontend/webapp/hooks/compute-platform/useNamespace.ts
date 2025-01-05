import { ACTION } from '@/utils';
import { useNotificationStore } from '@/store';
import { useMutation, useQuery } from '@apollo/client';
import { useComputePlatform } from './useComputePlatform';
import { GET_NAMESPACES, PERSIST_NAMESPACE } from '@/graphql';
import { type ComputePlatform, NOTIFICATION_TYPE, type PersistNamespaceItemInput } from '@/types';

export const useNamespace = (namespaceName?: string, instrumentationLabeled = null as boolean | null) => {
  const { addNotification } = useNotificationStore();
  const { data: cpData, loading: cpLoading } = useComputePlatform();

  const { data, loading } = useQuery<ComputePlatform>(GET_NAMESPACES, {
    skip: !namespaceName,
    variables: { namespaceName, instrumentationLabeled },
    onError: (error) =>
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || ACTION.FETCH,
        message: error.cause?.message || error.message,
      }),
  });

  const [persistNamespaceMutation] = useMutation(PERSIST_NAMESPACE, {
    onError: (error) =>
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || ACTION.UPDATE,
        message: error.cause?.message || error.message,
      }),
  });

  return {
    loading: loading || cpLoading,
    data: data?.computePlatform.k8sActualNamespace,
    allNamespaces: cpData?.computePlatform.k8sActualNamespaces,
    persistNamespace: async (namespace: PersistNamespaceItemInput) => await persistNamespaceMutation({ variables: { namespace } }),
  };
};
