import { useEffect } from 'react';
import { useConfig } from '../config';
import { usePaginatedStore } from '@/store';
import { GET_DESTINATIONS } from '@/graphql';
import { useLazyQuery, useMutation } from '@apollo/client';
import type { DestinationInput, FetchedDestination } from '@/@types';
import { CREATE_DESTINATION, DELETE_DESTINATION, UPDATE_DESTINATION } from '@/graphql/mutations';
import { type DestinationFormData, useNotificationStore, usePendingStore } from '@odigos/ui-containers';
import { CRUD, type Destination, DISPLAY_TITLES, ENTITY_TYPES, FORM_ALERTS, getSseTargetFromId, NOTIFICATION_TYPE } from '@odigos/ui-utils';

interface UseDestinationCrud {
  destinations: Destination[];
  destinationsLoading: boolean;
  fetchDestinations: () => void;
  createDestination: (destination: DestinationFormData) => void;
  updateDestination: (id: string, destination: DestinationFormData) => void;
  deleteDestination: (id: string) => void;
}

const mapFetched = (items: FetchedDestination[]): Destination[] => {
  return items.map((item) => {
    // Replace deprecated string values, with boolean values
    const fields =
      item.destinationType.type === 'clickhouse'
        ? item.fields.replace('"CLICKHOUSE_CREATE_SCHEME":"Create"', '"CLICKHOUSE_CREATE_SCHEME":"true"').replace('"CLICKHOUSE_CREATE_SCHEME":"Skip"', '"CLICKHOUSE_CREATE_SCHEME":"false"')
        : item.destinationType.type === 'qryn'
        ? item.fields
            .replace('"QRYN_ADD_EXPORTER_NAME":"Yes"', '"QRYN_ADD_EXPORTER_NAME":"true"')
            .replace('"QRYN_ADD_EXPORTER_NAME":"No"', '"QRYN_ADD_EXPORTER_NAME":"false"')
            .replace('"QRYN_RESOURCE_TO_TELEMETRY_CONVERSION":"Yes"', '"QRYN_RESOURCE_TO_TELEMETRY_CONVERSION":"true"')
            .replace('"QRYN_RESOURCE_TO_TELEMETRY_CONVERSION":"No"', '"QRYN_RESOURCE_TO_TELEMETRY_CONVERSION":"false"')
        : item.destinationType.type === 'qryn-oss'
        ? item.fields
            .replace('"QRYN_OSS_ADD_EXPORTER_NAME":"Yes"', '"QRYN_OSS_ADD_EXPORTER_NAME":"true"')
            .replace('"QRYN_OSS_ADD_EXPORTER_NAME":"No"', '"QRYN_OSS_ADD_EXPORTER_NAME":"false"')
            .replace('"QRYN_OSS_RESOURCE_TO_TELEMETRY_CONVERSION":"Yes"', '"QRYN_OSS_RESOURCE_TO_TELEMETRY_CONVERSION":"true"')
            .replace('"QRYN_OSS_RESOURCE_TO_TELEMETRY_CONVERSION":"No"', '"QRYN_OSS_RESOURCE_TO_TELEMETRY_CONVERSION":"false"')
        : item.fields;

    return { ...item, fields };
  });
};

export const useDestinationCRUD = (): UseDestinationCrud => {
  const { data: config } = useConfig();
  const { addNotification } = useNotificationStore();
  const { addPendingItems, removePendingItems } = usePendingStore();
  const { destinationsPaginating, setPaginating, destinations, addPaginated, removePaginated } = usePaginatedStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: ENTITY_TYPES.DESTINATION, target: id ? getSseTargetFromId(id, ENTITY_TYPES.DESTINATION) : undefined, hideFromHistory });
  };

  const [fetchAll, { loading: isFetching }] = useLazyQuery<{ computePlatform?: { destinations?: FetchedDestination[] } }>(GET_DESTINATIONS, {
    fetchPolicy: 'cache-and-network',
  });

  const fetchDestinations = async () => {
    setPaginating(ENTITY_TYPES.DESTINATION, true);
    const { error, data } = await fetchAll();

    if (!!error) {
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || CRUD.READ,
        message: error.cause?.message || error.message,
      });
    } else if (!!data?.computePlatform?.destinations) {
      const { destinations: items } = data.computePlatform;
      addPaginated(ENTITY_TYPES.DESTINATION, mapFetched(items));
      setPaginating(ENTITY_TYPES.DESTINATION, false);
    }
  };

  const [mutateCreate, cState] = useMutation<{ createNewDestination: FetchedDestination }, { destination: DestinationInput }>(CREATE_DESTINATION, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.CREATE, error.cause?.message || error.message),
    onCompleted: (res) => {
      const destination = res.createNewDestination;
      addPaginated(ENTITY_TYPES.DESTINATION, mapFetched([destination]));
      notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.CREATE, `Successfully created "${destination.destinationType.type}" destination`, destination.id);
    },
  });

  const [mutateUpdate, uState] = useMutation<{ updateDestination: { id: string } }, { id: string; destination: DestinationInput }>(UPDATE_DESTINATION, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id as string;
      removePendingItems([{ entityType: ENTITY_TYPES.DESTINATION, entityId: id }]);
      // We wait for SSE
    },
  });

  const [mutateDelete, dState] = useMutation<{ deleteDestination: boolean }, { id: string }>(DELETE_DESTINATION, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.DELETE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id as string;
      const destination = destinations.find((r) => r.id === id);
      removePaginated(ENTITY_TYPES.DESTINATION, [id]);
      notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.DELETE, `Successfully deleted "${destination?.destinationType.type || id}" destination`, id);
    },
  });

  useEffect(() => {
    if (!destinations.length && !destinationsPaginating) fetchDestinations();
  }, []);

  return {
    destinations,
    destinationsLoading: isFetching || destinationsPaginating || cState.loading || uState.loading || dState.loading,
    fetchDestinations,

    createDestination: (destination) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        mutateCreate({ variables: { destination: { ...destination, fields: destination.fields.filter(({ value }) => value !== undefined) } } });
      }
    },
    updateDestination: (id, destination) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        notifyUser(NOTIFICATION_TYPE.DEFAULT, 'Pending', 'Updating destination...', undefined, true);
        addPendingItems([{ entityType: ENTITY_TYPES.DESTINATION, entityId: id }]);
        mutateUpdate({ variables: { id, destination: { ...destination, fields: destination.fields.filter(({ value }) => value !== undefined) } } });
      }
    },
    deleteDestination: (id) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        mutateDelete({ variables: { id } });
      }
    },
  };
};
