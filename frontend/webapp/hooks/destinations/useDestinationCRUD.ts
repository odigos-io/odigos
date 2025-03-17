import { useEffect } from 'react';
import { useConfig } from '../config';
import { GET_DESTINATIONS } from '@/graphql';
import { mapFetchedDestinations } from '@/utils';
import { useLazyQuery, useMutation } from '@apollo/client';
import { getSseTargetFromId } from '@odigos/ui-kit/functions';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { useEntityStore, useNotificationStore, usePendingStore } from '@odigos/ui-kit/store';
import { CREATE_DESTINATION, DELETE_DESTINATION, UPDATE_DESTINATION } from '@/graphql/mutations';
import { CRUD, ENTITY_TYPES, NOTIFICATION_TYPE, type Destination, type DestinationFormData } from '@odigos/ui-kit/types';

interface UseDestinationCrud {
  destinations: Destination[];
  destinationsLoading: boolean;
  fetchDestinations: () => void;
  createDestination: (destination: DestinationFormData) => void;
  updateDestination: (id: string, destination: DestinationFormData) => void;
  deleteDestination: (id: string) => void;
}

export const useDestinationCRUD = (): UseDestinationCrud => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();
  const { addPendingItems, removePendingItems } = usePendingStore();
  const { destinationsLoading, setEntitiesLoading, destinations, addEntities, removeEntities } = useEntityStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: ENTITY_TYPES.DESTINATION, target: id ? getSseTargetFromId(id, ENTITY_TYPES.DESTINATION) : undefined, hideFromHistory });
  };

  const [fetchAll] = useLazyQuery<{ computePlatform?: { destinations?: Destination[] } }>(GET_DESTINATIONS, {
    fetchPolicy: 'cache-and-network',
  });

  const fetchDestinations = async () => {
    setEntitiesLoading(ENTITY_TYPES.DESTINATION, true);
    const { error, data } = await fetchAll();

    if (error) {
      notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message);
    } else if (data?.computePlatform?.destinations) {
      const { destinations: items } = data.computePlatform;
      addEntities(ENTITY_TYPES.DESTINATION, mapFetchedDestinations(items));
      setEntitiesLoading(ENTITY_TYPES.DESTINATION, false);
    }
  };

  const [mutateCreate] = useMutation<{ createNewDestination: Destination }, { destination: DestinationFormData }>(CREATE_DESTINATION, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.CREATE, error.cause?.message || error.message),
    onCompleted: (res) => {
      const destination = res.createNewDestination;
      addEntities(ENTITY_TYPES.DESTINATION, mapFetchedDestinations([destination]));
      notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.CREATE, `Successfully created "${destination.destinationType.type}" destination`, destination.id);
    },
  });

  const [mutateUpdate] = useMutation<{ updateDestination: { id: string } }, { id: string; destination: DestinationFormData }>(UPDATE_DESTINATION, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      setTimeout(() => {
        const id = req?.variables?.id as string;
        const destination = destinations.find((r) => r.id === id);
        removePendingItems([{ entityType: ENTITY_TYPES.DESTINATION, entityId: id }]);
        notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.CREATE, `Successfully updated "${destination?.destinationType?.type || id}" destination`, id);
        // We wait for SSE
      }, 3000);
    },
  });

  const [mutateDelete] = useMutation<{ deleteDestination: boolean }, { id: string }>(DELETE_DESTINATION, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.DELETE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id as string;
      const destination = destinations.find((r) => r.id === id);
      removeEntities(ENTITY_TYPES.DESTINATION, [id]);
      notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.DELETE, `Successfully deleted "${destination?.destinationType.type || id}" destination`, id);
    },
  });

  const createDestination: UseDestinationCrud['createDestination'] = (destination) => {
    if (isReadonly) {
      notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateCreate({ variables: { destination: { ...destination, fields: destination.fields.filter(({ value }) => value !== undefined) } } });
    }
  };

  const updateDestination: UseDestinationCrud['updateDestination'] = (id, destination) => {
    if (isReadonly) {
      notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      notifyUser(NOTIFICATION_TYPE.DEFAULT, 'Pending', 'Updating destination...', undefined, true);
      addPendingItems([{ entityType: ENTITY_TYPES.DESTINATION, entityId: id }]);
      mutateUpdate({ variables: { id, destination: { ...destination, fields: destination.fields.filter(({ value }) => value !== undefined) } } });
    }
  };

  const deleteDestination: UseDestinationCrud['deleteDestination'] = (id) => {
    if (isReadonly) {
      notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateDelete({ variables: { id } });
    }
  };

  useEffect(() => {
    if (!destinations.length && !destinationsLoading) fetchDestinations();
  }, []);

  return {
    destinations,
    destinationsLoading,
    fetchDestinations,
    createDestination,
    updateDestination,
    deleteDestination,
  };
};
