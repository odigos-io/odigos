import { useEffect, useMemo } from 'react';
import { useConfig } from '../config';
import { useMutation, useQuery } from '@apollo/client';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { CRUD, NOTIFICATION_TYPE } from '@odigos/ui-kit/types';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import type { NamespaceInstrumentInput, FetchedNamespace } from '@/types';
import { GET_NAMESPACE, GET_NAMESPACES, PERSIST_NAMESPACE } from '@/graphql';

export const useNamespace = (namespaceName?: string) => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, hideFromHistory });
  };

  const {
    refetch: queryAll,
    data: allNamespaces,
    loading: allLoading,
  } = useQuery<{ computePlatform?: { k8sActualNamespaces?: FetchedNamespace[] } }>(GET_NAMESPACES, {
    onError: (error) => addNotification({ type: NOTIFICATION_TYPE.ERROR, title: error.name || CRUD.READ, message: error.cause?.message || error.message }),
  });

  // TODO: change query, to lazy query
  const {
    refetch: querySingle,
    data: singleNamespace,
    loading: singleLoading,
  } = useQuery<{ computePlatform?: { k8sActualNamespace?: FetchedNamespace } }>(GET_NAMESPACE, {
    skip: !namespaceName,
    variables: { namespaceName },
    onError: (error) => addNotification({ type: NOTIFICATION_TYPE.ERROR, title: error.name || CRUD.READ, message: error.cause?.message || error.message }),
  });

  const [mutatePersist] = useMutation<{ persistK8sNamespace: boolean }>(PERSIST_NAMESPACE, {
    onError: (error) => addNotification({ type: NOTIFICATION_TYPE.ERROR, title: error.name || CRUD.UPDATE, message: error.cause?.message || error.message }),
  });

  const persistNamespace = async (payload: NamespaceInstrumentInput) => {
    if (isReadonly) {
      notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, true);
    } else {
      await mutatePersist({ variables: { namespace: payload } });
    }
  };

  useEffect(() => {
    if (!allNamespaces?.computePlatform?.k8sActualNamespaces?.length) queryAll();
  }, []);

  useEffect(() => {
    if (!!namespaceName && !singleLoading) querySingle({ namespaceName });
  }, [namespaceName]);

  const namespaces = useMemo(
    () =>
      (allNamespaces?.computePlatform?.k8sActualNamespaces || []).map(({ name, selected, k8sActualSources }) => ({
        name,
        selected,
        sources: k8sActualSources,
      })),
    [allNamespaces],
  );

  const namespace = useMemo(
    () =>
      !!singleNamespace?.computePlatform?.k8sActualNamespace
        ? {
            name: singleNamespace.computePlatform.k8sActualNamespace.name,
            selected: singleNamespace.computePlatform.k8sActualNamespace.selected,
            sources: singleNamespace.computePlatform.k8sActualNamespace.k8sActualSources,
          }
        : undefined,
    [singleNamespace],
  );

  return {
    loading: allLoading || singleLoading,
    namespaces,
    namespace,
    persistNamespace,
  };
};
