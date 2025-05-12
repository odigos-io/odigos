import { useEffect } from 'react';
import { useConfig } from '../config';
import { GET_DESTINATIONS } from '@/graphql';
import { mapFetchedDestinations } from '@/utils';
import { useLazyQuery, useMutation } from '@apollo/client';
import { getSseTargetFromId } from '@odigos/ui-kit/functions';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { useDataStreamStore, useEntityStore, useNotificationStore, usePendingStore } from '@odigos/ui-kit/store';
import { CREATE_DESTINATION, DELETE_DESTINATION, UPDATE_DESTINATION } from '@/graphql/mutations';
import { Crud, EntityTypes, StatusType, type Destination, type DestinationFormData } from '@odigos/ui-kit/types';

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
  const { selectedStreamName } = useDataStreamStore();
  const { addPendingItems, removePendingItems } = usePendingStore();
  const { destinationsLoading, setEntitiesLoading, destinations, addEntities, removeEntities } = useEntityStore();

  const notifyUser = (type: StatusType, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: EntityTypes.Destination, target: id ? getSseTargetFromId(id, EntityTypes.Destination) : undefined, hideFromHistory });
  };

  const [fetchAll] = useLazyQuery<{ computePlatform?: { destinations?: Destination[] } }, { streamName: string }>(GET_DESTINATIONS);

  const fetchDestinations = async () => {
    setEntitiesLoading(EntityTypes.Destination, true);
    const { error, data } = await fetchAll({ variables: { streamName: selectedStreamName } });

    if (error) {
      notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message);
    } else if (data?.computePlatform?.destinations) {
      const { destinations: items } = data.computePlatform;
      addEntities(EntityTypes.Destination, mapFetchedDestinations(items));
      setEntitiesLoading(EntityTypes.Destination, false);
    }
  };

  const [mutateCreate] = useMutation<{ createNewDestination: Destination }, { destination: DestinationFormData }>(CREATE_DESTINATION, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Create, error.cause?.message || error.message),
    onCompleted: (res) => {
      const destination = res.createNewDestination;
      addEntities(EntityTypes.Destination, mapFetchedDestinations([destination]));
      notifyUser(StatusType.Success, Crud.Create, `Successfully created "${destination.destinationType.type}" destination`, destination.id);
    },
  });

  const [mutateUpdate] = useMutation<{ updateDestination: { id: string } }, { id: string; destination: DestinationFormData }>(UPDATE_DESTINATION, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      setTimeout(() => {
        const id = req?.variables?.id as string;
        const destination = destinations.find((r) => r.id === id);
        removePendingItems([{ entityType: EntityTypes.Destination, entityId: id }]);
        notifyUser(StatusType.Success, Crud.Update, `Successfully updated "${destination?.destinationType?.type || id}" destination`, id);
        // We wait for SSE
      }, 3000);
    },
  });

  const [mutateDelete] = useMutation<{ deleteDestination: boolean }, { id: string }>(DELETE_DESTINATION, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Delete, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id as string;
      const destination = destinations.find((r) => r.id === id);
      removeEntities(EntityTypes.Destination, [id]);
      notifyUser(StatusType.Success, Crud.Delete, `Successfully deleted "${destination?.destinationType.type || id}" destination`, id);
    },
  });

  const createDestination: UseDestinationCrud['createDestination'] = (destination) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateCreate({ variables: { destination: { ...destination, fields: destination.fields.filter(({ value }) => value !== undefined) } } });
    }
  };

  const updateDestination: UseDestinationCrud['updateDestination'] = (id, destination) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      notifyUser(StatusType.Default, 'Pending', 'Updating destination...', undefined, true);
      addPendingItems([{ entityType: EntityTypes.Destination, entityId: id }]);
      mutateUpdate({ variables: { id, destination: { ...destination, fields: destination.fields.filter(({ value }) => value !== undefined) } } });
    }
  };

  const deleteDestination: UseDestinationCrud['deleteDestination'] = (id) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
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
