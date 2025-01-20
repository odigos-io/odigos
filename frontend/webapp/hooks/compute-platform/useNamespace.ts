import { ACTION } from '@/utils';
import { useMutation, useQuery } from '@apollo/client';
import { useComputePlatform } from './useComputePlatform';
import { useAppStore, useNotificationStore } from '@/store';
import { GET_NAMESPACE, PERSIST_NAMESPACE } from '@/graphql';
import { type ComputePlatform, NOTIFICATION_TYPE, OVERVIEW_ENTITY_TYPES, type PersistNamespaceItemInput } from '@/types';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useNamespace = (namespaceName?: string, params?: Params) => {
  const { setConfiguredFutureApps } = useAppStore();
  const { addNotification } = useNotificationStore();
  const { data: cp, loading: cpLoading } = useComputePlatform();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: OVERVIEW_ENTITY_TYPES.SOURCE, hideFromHistory });
  };

  const handleError = (actionType: string, message: string) => {
    notifyUser(NOTIFICATION_TYPE.ERROR, actionType, message);
    params?.onError?.(actionType);
  };

  const handleComplete = (actionType: string) => {
    setConfiguredFutureApps({});
    params?.onSuccess?.(actionType);
  };

  const { data, loading } = useQuery<ComputePlatform>(GET_NAMESPACE, {
    skip: !namespaceName,
    variables: { namespaceName },
    onError: (error) => handleError(error.name || ACTION.FETCH, error.cause?.message || error.message),
  });

  const [persistNamespaceMutation] = useMutation<{ persistK8sNamespace: boolean }, { namespace: PersistNamespaceItemInput }>(PERSIST_NAMESPACE, {
    onError: (error) => handleError(error.name || ACTION.UPDATE, error.cause?.message || error.message),
    onCompleted: () => handleComplete(''),
  });

  return {
    loading: loading || cpLoading,
    data: data?.computePlatform?.k8sActualNamespace,
    allNamespaces: cp?.computePlatform?.k8sActualNamespaces || [],

    persistNamespace: async (namespace: PersistNamespaceItemInput) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', `Persisting "${namespace.name}" namespace...`, true);
      await persistNamespaceMutation({ variables: { namespace } });
    },

    persistNamespaces: async (selectNamespaceList: { [namespace: string]: boolean }) => {
      const entries = Object.entries(selectNamespaceList);

      // this is to handle "on success" callback if there are no namespaces to persist,
      // and to notify if there are namespace to persist
      let hasNamespaces = !!entries.length;

      if (hasNamespaces) notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Persisting namespaces...', true);
      for (const [name, futureSelected] of entries) await persistNamespaceMutation({ variables: { namespace: { name, futureSelected } } });
      if (!hasNamespaces) handleComplete('');
    },
  };
};
