import { useConfig } from '../config';
import { GET_DESTINATIONS } from '@/graphql';
import { useMutation, useQuery } from '@apollo/client';
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
  const { addPendingItems, removePendingItems } = usePendingStore();
  const { addNotification, removeNotifications } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: ENTITY_TYPES.DESTINATION, target: id ? getSseTargetFromId(id, ENTITY_TYPES.DESTINATION) : undefined, hideFromHistory });
  };

  const {
    data,
    loading: isFetching,
    refetch: fetchDestinations,
  } = useQuery<{ computePlatform: { destinations: FetchedDestination[] } }>(GET_DESTINATIONS, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message),
  });

  const [mutateCreate, cState] = useMutation<{ createNewDestination: { id: string } }, { destination: DestinationInput }>(CREATE_DESTINATION, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.CREATE, error.cause?.message || error.message),
  });

  const [mutateUpdate, uState] = useMutation<{ updateDestination: { id: string } }, { id: string; destination: DestinationInput }>(UPDATE_DESTINATION, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      // This is instead of toasting a k8s modified-event watcher...
      // If we do toast with a watcher, we can't guarantee an SSE will be sent for this update alone. It will definitely include SSE for all updates, even those unexpected.
      // Not that there's anything about a watcher that would break the UI, it's just that we would receive unexpected events with ridiculous amounts.
      setTimeout(() => {
        const { id, destination } = req?.variables || {};

        notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.UPDATE, `Successfully updated "${destination.type}" destination`, id);
        removePendingItems([{ entityType: ENTITY_TYPES.DESTINATION, entityId: id }]);
      }, 1000);
    },
  });

  const [mutateDelete, dState] = useMutation<{ deleteDestination: boolean }, { id: string }>(DELETE_DESTINATION, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.DELETE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id;
      removeNotifications(getSseTargetFromId(id, ENTITY_TYPES.DESTINATION));
    },
  });

  return {
    destinations: mapFetched(data?.computePlatform?.destinations || []),
    destinationsLoading: isFetching || cState.loading || uState.loading || dState.loading,
    fetchDestinations,

    createDestination: (destination) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Creating destination...', undefined, true);
        addPendingItems([{ entityType: ENTITY_TYPES.DESTINATION, entityId: undefined }]);
        mutateCreate({ variables: { destination: { ...destination, fields: destination.fields.filter(({ value }) => value !== undefined) } } });
      }
    },
    updateDestination: (id, destination) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Updating destination...', undefined, true);
        addPendingItems([{ entityType: ENTITY_TYPES.DESTINATION, entityId: id }]);
        mutateUpdate({ variables: { id, destination: { ...destination, fields: destination.fields.filter(({ value }) => value !== undefined) } } });
      }
    },
    deleteDestination: (id) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Deleting destination...', undefined, true);
        addPendingItems([{ entityType: ENTITY_TYPES.DESTINATION, entityId: id }]);
        mutateDelete({ variables: { id } });
      }
    },
  };
};
