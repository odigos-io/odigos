import { useEffect, useMemo } from 'react';
import { useConfig } from '../config';
import { useMutation, useQuery } from '@apollo/client';
import type { NamespaceInstrumentInput } from '@/types';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { CRUD, Namespace, STATUS_TYPE } from '@odigos/ui-kit/types';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { GET_NAMESPACE, GET_NAMESPACES, PERSIST_NAMESPACE } from '@/graphql';

export const useNamespace = (namespaceName?: string) => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: STATUS_TYPE, title: string, message: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, hideFromHistory });
  };

  const {
    refetch: queryAll,
    data: allNamespaces,
    loading: allLoading,
  } = useQuery<{ computePlatform?: { k8sActualNamespaces?: Namespace[] } }>(GET_NAMESPACES, {
    onError: (error) => addNotification({ type: STATUS_TYPE.ERROR, title: error.name || CRUD.READ, message: error.cause?.message || error.message }),
  });

  // TODO: change query, to lazy query
  const {
    refetch: querySingle,
    data: singleNamespace,
    loading: singleLoading,
  } = useQuery<{ computePlatform?: { k8sActualNamespace?: Namespace } }>(GET_NAMESPACE, {
    skip: !namespaceName,
    variables: { namespaceName },
    onError: (error) => addNotification({ type: STATUS_TYPE.ERROR, title: error.name || CRUD.READ, message: error.cause?.message || error.message }),
  });

  const [mutatePersist] = useMutation<{ persistK8sNamespace: boolean }>(PERSIST_NAMESPACE, {
    onError: (error) => {
      // TODO: after estimating the number of instrumentationConfigs to create for future apps in "useSourceCRUD" hook, then uncomment the below
      // setInstrumentCount('sourcesToCreate', 0);
      // setInstrumentCount('sourcesCreated', 0);
      // setInstrumentAwait(false);
      addNotification({ type: STATUS_TYPE.ERROR, title: error.name || CRUD.UPDATE, message: error.cause?.message || error.message });
    },
  });

  const persistNamespace = async (payload: NamespaceInstrumentInput) => {
    if (isReadonly) {
      notifyUser(STATUS_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, true);
    } else {
      await mutatePersist({ variables: { namespace: payload } });
    }
  };

  useEffect(() => {
    if (!allNamespaces?.computePlatform?.k8sActualNamespaces?.length) queryAll();
  }, []);

  useEffect(() => {
    if (namespaceName && !singleLoading) querySingle({ namespaceName });
  }, [namespaceName]);

  const namespaces = allNamespaces?.computePlatform?.k8sActualNamespaces || [];
  const namespace = singleNamespace?.computePlatform?.k8sActualNamespace;

  return {
    loading: allLoading || singleLoading,
    namespaces,
    namespace,
    persistNamespace,
  };
};
