import { useEffect } from 'react';
import { useConfig } from '../config';
import type { NamespaceInstrumentInput } from '@/types';
import { useLazyQuery, useMutation } from '@apollo/client';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { GET_NAMESPACE, GET_NAMESPACES, PERSIST_NAMESPACES } from '@/graphql';
import { Crud, EntityTypes, Namespace, StatusType } from '@odigos/ui-kit/types';
import { useDataStreamStore, useEntityStore, useNotificationStore } from '@odigos/ui-kit/store';

interface FetchSingleNamespacePayload {
  namespaceName: string;
}

export const useNamespace = () => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();
  const { selectedStreamName } = useDataStreamStore();
  const { namespacesLoading, setEntitiesLoading, namespaces, setEntities } = useEntityStore();

  const notifyUser = (type: StatusType, title: string, message: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, hideFromHistory });
  };

  const [queryAllNs] = useLazyQuery<{ computePlatform?: { k8sActualNamespaces?: Namespace[] } }>(GET_NAMESPACES, {
    onError: (error) => addNotification({ type: StatusType.Error, title: error.name || Crud.Read, message: error.cause?.message || error.message }),
  });
  const [querySingleNs] = useLazyQuery<{ computePlatform?: { k8sActualNamespace?: Namespace } }, FetchSingleNamespacePayload>(GET_NAMESPACE, {
    onError: (error) => addNotification({ type: StatusType.Error, title: error.name || Crud.Read, message: error.cause?.message || error.message }),
  });

  const [mutatePersist] = useMutation<{ persistK8sNamespaces: boolean }, NamespaceInstrumentInput>(PERSIST_NAMESPACES, {
    onError: (error) => {
      // TODO: after estimating the number of instrumentationConfigs to create for future apps in "useSourceCRUD" hook, then uncomment the below
      // setInstrumentCount('sourcesToCreate', 0);
      // setInstrumentCount('sourcesCreated', 0);
      // setInstrumentAwait(false);
      addNotification({ type: StatusType.Error, title: error.name || Crud.Update, message: error.cause?.message || error.message });
    },
  });

  const fetchNamespaces = async () => {
    setEntitiesLoading(EntityTypes.Namespace, true);
    const { error, data } = await queryAllNs();

    if (error) {
      notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message);
    } else if (data?.computePlatform?.k8sActualNamespaces) {
      const { k8sActualNamespaces: items } = data.computePlatform;
      setEntities(EntityTypes.Namespace, items);
      setEntitiesLoading(EntityTypes.Namespace, false);
    }
  };

  const persistNamespaces = async (payload: NamespaceInstrumentInput) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, true);
    } else {
      await mutatePersist({ variables: payload });
    }
  };

  useEffect(() => {
    if (selectedStreamName && !namespaces.length && !namespacesLoading) fetchNamespaces();
  }, [selectedStreamName]);

  return {
    namespacesLoading,
    namespaces,
    fetchNamespaces,
    fetchNamespace: (payload: FetchSingleNamespacePayload) => querySingleNs({ variables: payload }),
    persistNamespaces,
  };
};
