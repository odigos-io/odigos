import { useConfig } from '../config';
import { useMutation, useQuery } from '@apollo/client';
import { useComputePlatform } from './useComputePlatform';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { GET_NAMESPACE, PERSIST_NAMESPACE } from '@/graphql';
import { CRUD, NOTIFICATION_TYPE } from '@odigos/ui-kit/types';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import type { NamespaceInstrumentInput, ComputePlatform } from '@/@types';

export const useNamespace = (namespaceName?: string) => {
  const { data: config } = useConfig();
  const { addNotification } = useNotificationStore();
  const { data: cp, loading: cpLoading } = useComputePlatform();

  // TODO: change query, to lazy query
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
    namespaces: (cp?.computePlatform?.k8sActualNamespaces || []).map(({ name, selected, k8sActualSources }) => ({
      name,
      selected,
      sources: k8sActualSources,
    })),
    data: !!data?.computePlatform?.k8sActualNamespace
      ? {
          name: data.computePlatform.k8sActualNamespace.name,
          selected: data.computePlatform.k8sActualNamespace.selected,
          sources: data.computePlatform.k8sActualNamespace.k8sActualSources,
        }
      : undefined,

    persistNamespace: async (namespace: NamespaceInstrumentInput) => {
      if (config?.readonly) {
        addNotification({ type: NOTIFICATION_TYPE.WARNING, title: DISPLAY_TITLES.READONLY, message: FORM_ALERTS.READONLY_WARNING, hideFromHistory: true });
      } else {
        await persistNamespaceMutation({ variables: { namespace } });
      }
    },
  };
};
