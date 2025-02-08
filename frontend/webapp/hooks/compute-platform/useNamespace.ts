import { useConfig } from '../config';
import { useMutation, useQuery } from '@apollo/client';
import { useComputePlatform } from './useComputePlatform';
import { GET_NAMESPACE, PERSIST_NAMESPACE } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-containers';
import { CRUD, DISPLAY_TITLES, FORM_ALERTS, NOTIFICATION_TYPE } from '@odigos/ui-utils';
import type { FetchedNamespace, NamespaceInstrumentInput, ComputePlatform } from '@/types';

interface UseNameSpaceResponse {
  loading: boolean;
  data?: FetchedNamespace;
  allNamespaces: FetchedNamespace[];

  persistNamespace: (namespace: NamespaceInstrumentInput) => Promise<void>;
}

export const useNamespace = (namespaceName?: string): UseNameSpaceResponse => {
  const { data: config } = useConfig();
  const { addNotification } = useNotificationStore();
  const { data: cp, loading: cpLoading } = useComputePlatform();

  const { data, loading } = useQuery<ComputePlatform>(GET_NAMESPACE, {
    skip: !namespaceName,
    variables: { namespaceName },
    onError: (error) => addNotification({ type: NOTIFICATION_TYPE.ERROR, title: error.name || CRUD.READ, message: error.cause?.message || error.message }),
  });

  const [persistNamespaceMutation] = useMutation(PERSIST_NAMESPACE, {
    onError: (error) => addNotification({ type: NOTIFICATION_TYPE.ERROR, title: error.name || CRUD.UPDATE, message: error.cause?.message || error.message }),
  });

  return {
    loading: loading || cpLoading,
    data: data?.computePlatform?.k8sActualNamespace,
    allNamespaces: cp?.computePlatform?.k8sActualNamespaces || [],

    persistNamespace: async (namespace) => {
      if (config?.readonly) {
        addNotification({ type: NOTIFICATION_TYPE.WARNING, title: DISPLAY_TITLES.READONLY, message: FORM_ALERTS.READONLY_WARNING, hideFromHistory: true });
      } else {
        await persistNamespaceMutation({ variables: { namespace } });
      }
    },
  };
};
