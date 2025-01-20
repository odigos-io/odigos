import { ACTION } from '@/utils';
import { useMutation, useQuery } from '@apollo/client';
import { useComputePlatform } from './useComputePlatform';
import { useAppStore, useNotificationStore } from '@/store';
import { GET_NAMESPACE, PERSIST_NAMESPACE } from '@/graphql';
import { type ComputePlatform, NOTIFICATION_TYPE, type PersistNamespaceItemInput } from '@/types';

export const useNamespace = (namespaceName?: string) => {
  const { setConfiguredFutureApps } = useAppStore();
  const { addNotification } = useNotificationStore();
  const { data: cp, loading: cpLoading } = useComputePlatform();

  const handleError = (title: string, message: string) => addNotification({ type: NOTIFICATION_TYPE.ERROR, title, message });
  const handleComplete = () => setConfiguredFutureApps({});

  const { data, loading } = useQuery<ComputePlatform>(GET_NAMESPACE, {
    skip: !namespaceName,
    variables: { namespaceName },
    onError: (error) => handleError(error.name || ACTION.FETCH, error.cause?.message || error.message),
  });

  const [persistNamespaceMutation] = useMutation<{ persistK8sNamespace: boolean }, { namespace: PersistNamespaceItemInput }>(PERSIST_NAMESPACE, {
    onError: (error) => handleError(error.name || ACTION.UPDATE, error.cause?.message || error.message),
    onCompleted: () => handleComplete(),
  });

  return {
    loading: loading || cpLoading,
    data: data?.computePlatform?.k8sActualNamespace,
    allNamespaces: cp?.computePlatform?.k8sActualNamespaces || [],

    persistNamespace: async (payload: PersistNamespaceItemInput) => {
      await persistNamespaceMutation({ variables: { namespace: payload } });
    },

    persistNamespaces: async (payload: { [namespace: string]: boolean }) => {
      const entries = Object.entries(payload);

      for (const [name, futureSelected] of entries) {
        await persistNamespaceMutation({ variables: { namespace: { name, futureSelected } } });
      }

      // this is to handle "on success" callback if there are no namespaces to persist
      if (!entries.length) handleComplete();
    },
  };
};
