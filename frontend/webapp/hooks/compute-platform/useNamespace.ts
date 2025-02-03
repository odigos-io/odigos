import { ACTION, DISPLAY_TITLES, FORM_ALERTS } from '@/utils';
import { useConfig } from '../config';
import { useNotificationStore } from '@/store';
import { useMutation, useQuery } from '@apollo/client';
import { NOTIFICATION_TYPE } from '@odigos/ui-components';
import { useComputePlatform } from './useComputePlatform';
import { GET_NAMESPACE, PERSIST_NAMESPACE } from '@/graphql';
import { type ComputePlatform, type PersistNamespaceItemInput } from '@/types';

export const useNamespace = (namespaceName?: string) => {
  const { data: config } = useConfig();
  const { addNotification } = useNotificationStore();
  const { data: cp, loading: cpLoading } = useComputePlatform();

  const { data, loading } = useQuery<ComputePlatform>(GET_NAMESPACE, {
    skip: !namespaceName,
    variables: { namespaceName },
    onError: (error) => addNotification({ type: NOTIFICATION_TYPE.ERROR, title: error.name || ACTION.FETCH, message: error.cause?.message || error.message }),
  });

  const [persistNamespaceMutation] = useMutation(PERSIST_NAMESPACE, {
    onError: (error) => addNotification({ type: NOTIFICATION_TYPE.ERROR, title: error.name || ACTION.UPDATE, message: error.cause?.message || error.message }),
  });

  return {
    loading: loading || cpLoading,
    data: data?.computePlatform?.k8sActualNamespace,
    allNamespaces: cp?.computePlatform?.k8sActualNamespaces || [],

    persistNamespace: async (namespace: PersistNamespaceItemInput) => {
      if (config?.readonly) {
        addNotification({ type: NOTIFICATION_TYPE.WARNING, title: DISPLAY_TITLES.READONLY, message: FORM_ALERTS.READONLY_WARNING, hideFromHistory: true });
      } else {
        await persistNamespaceMutation({ variables: { namespace } });
      }
    },
  };
};
