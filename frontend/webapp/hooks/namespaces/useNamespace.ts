import { useEffect } from 'react';
import { useConfig } from '../config';
import { useMutation, useQuery } from '@apollo/client';
import type { NamespaceInstrumentInput } from '@/types';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { useEntityStore, useNotificationStore } from '@odigos/ui-kit/store';
import { GET_NAMESPACE, GET_NAMESPACES, PERSIST_NAMESPACE } from '@/graphql';
import { Crud, EntityTypes, Namespace, StatusType } from '@odigos/ui-kit/types';

export const useNamespace = (namespaceName?: string) => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();
  const { namespacesLoading, setEntitiesLoading, namespaces, addEntities } = useEntityStore();

  const notifyUser = (type: StatusType, title: string, message: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, hideFromHistory });
  };

  // TODO: change query, to lazy query (needs to be handled in the UI-Kit first)
  const { refetch: fetchAll } = useQuery<{ computePlatform?: { k8sActualNamespaces?: Namespace[] } }>(GET_NAMESPACES, {
    onError: (error) => addNotification({ type: StatusType.Error, title: error.name || Crud.Read, message: error.cause?.message || error.message }),
  });

  const fetchNamespaces = async () => {
    setEntitiesLoading(EntityTypes.Namespace, true);
    const { error, data } = await fetchAll();

    if (error) {
      notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message);
    } else if (data?.computePlatform?.k8sActualNamespaces) {
      const { k8sActualNamespaces: items } = data.computePlatform;
      addEntities(EntityTypes.Namespace, items);
      setEntitiesLoading(EntityTypes.Namespace, false);
    }
  };

  // TODO: change query, to lazy query (needs to be handled in the UI-Kit first)
  const {
    refetch: fetchSingle,
    data: singleNamespace,
    loading: singleLoading,
  } = useQuery<{ computePlatform?: { k8sActualNamespace?: Namespace } }>(GET_NAMESPACE, {
    skip: !namespaceName,
    variables: { namespaceName },
    onError: (error) => addNotification({ type: StatusType.Error, title: error.name || Crud.Read, message: error.cause?.message || error.message }),
  });

  const [mutatePersist] = useMutation<{ persistK8sNamespace: boolean }>(PERSIST_NAMESPACE, {
    onError: (error) => {
      // TODO: after estimating the number of instrumentationConfigs to create for future apps in "useSourceCRUD" hook, then uncomment the below
      // setInstrumentCount('sourcesToCreate', 0);
      // setInstrumentCount('sourcesCreated', 0);
      // setInstrumentAwait(false);
      addNotification({ type: StatusType.Error, title: error.name || Crud.Update, message: error.cause?.message || error.message });
    },
  });

  const persistNamespace = async (payload: NamespaceInstrumentInput) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, true);
    } else {
      await mutatePersist({ variables: { namespace: payload } });
    }
  };

  useEffect(() => {
    if (!namespaces.length) fetchNamespaces();
  }, []);

  useEffect(() => {
    if (namespaceName && !singleLoading) fetchSingle({ namespaceName });
  }, [namespaceName]);

  return {
    loading: namespacesLoading || singleLoading,
    namespaces,
    namespace: singleNamespace?.computePlatform?.k8sActualNamespace,
    persistNamespace,
  };
};
