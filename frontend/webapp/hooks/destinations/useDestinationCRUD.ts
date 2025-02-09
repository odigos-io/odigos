import { useMemo } from 'react';
import { useConfig } from '../config';
import { GET_DESTINATIONS } from '@/graphql';
import { useMutation, useQuery } from '@apollo/client';
import { type DestinationInput, type ComputePlatform } from '@/types';
import { useFilterStore, useNotificationStore, usePendingStore } from '@odigos/ui-containers';
import { CREATE_DESTINATION, DELETE_DESTINATION, UPDATE_DESTINATION } from '@/graphql/mutations';
import { CRUD, type Destination, type DestinationOption, DISPLAY_TITLES, ENTITY_TYPES, FORM_ALERTS, getSseTargetFromId, NOTIFICATION_TYPE } from '@odigos/ui-utils';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

interface UseDestinationCrudResponse {
  loading: boolean;
  destinations: Destination[];
  filteredDestinations: Destination[];
  refetchDestinations: () => void;

  createDestination: (destination: DestinationInput) => void;
  updateDestination: (id: string, destination: DestinationInput) => void;
  deleteDestination: (id: string) => void;
}

export const useDestinationCRUD = (params?: Params): UseDestinationCrudResponse => {
  const filters = useFilterStore();
  const { data: config } = useConfig();
  const { addPendingItems, removePendingItems } = usePendingStore();
  const { addNotification, removeNotifications } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({
      type,
      title,
      message,
      crdType: ENTITY_TYPES.DESTINATION,
      target: id ? getSseTargetFromId(id, ENTITY_TYPES.DESTINATION) : undefined,
      hideFromHistory,
    });
  };

  const handleError = (actionType: string, message: string) => {
    notifyUser(NOTIFICATION_TYPE.ERROR, actionType, message);
    params?.onError?.(actionType);
  };

  const handleComplete = (actionType: string) => {
    params?.onSuccess?.(actionType);
  };

  // Fetch data
  const { data, loading, refetch } = useQuery<ComputePlatform>(GET_DESTINATIONS, {
    onError: (error) => handleError(error.name || CRUD.READ, error.cause?.message || error.message),
  });

  // Map fetched data
  const mapped = useMemo(() => {
    return (data?.computePlatform?.destinations || []).map((item) => {
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
  }, [data]);

  // Filter mapped data
  const filtered = useMemo(() => {
    let arr = [...mapped];
    if (!!filters.monitors.length) arr = arr.filter((destination) => !!filters.monitors.find((metric) => destination.exportedSignals[metric.id as keyof DestinationOption['supportedSignals']]));
    return arr;
  }, [mapped, filters]);

  const [createDestination, cState] = useMutation<{ createNewDestination: { id: string } }>(CREATE_DESTINATION, {
    onError: (error) => handleError(CRUD.CREATE, error.message),
    onCompleted: () => handleComplete(CRUD.CREATE),
  });

  const [updateDestination, uState] = useMutation<{ updateDestination: { id: string } }>(UPDATE_DESTINATION, {
    onError: (error) => handleError(CRUD.UPDATE, error.message),
    onCompleted: (res, req) => {
      handleComplete(CRUD.UPDATE);

      // This is instead of toasting a k8s modified-event watcher...
      // If we do toast with a watcher, we can't guarantee an SSE will be sent for this update alone. It will definitely include SSE for all updates, even those unexpected.
      // Not that there's anything about a watcher that would break the UI, it's just that we would receive unexpected events with ridiculous amounts.
      setTimeout(() => {
        const { id, destination } = req?.variables || {};

        refetch();
        notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.UPDATE, `Successfully updated "${destination.type}" destination`, id);
        removePendingItems([{ entityType: ENTITY_TYPES.DESTINATION, entityId: id }]);
      }, 2000);
    },
  });

  const [deleteDestination, dState] = useMutation<{ deleteDestination: boolean }>(DELETE_DESTINATION, {
    onError: (error) => handleError(CRUD.DELETE, error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id;
      removeNotifications(getSseTargetFromId(id, ENTITY_TYPES.DESTINATION));
      handleComplete(CRUD.DELETE);
    },
  });

  return {
    loading: loading || cState.loading || uState.loading || dState.loading,
    destinations: mapped,
    filteredDestinations: filtered,
    refetchDestinations: refetch,

    createDestination: async (destination) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Creating destination...', undefined, true);
        addPendingItems([{ entityType: ENTITY_TYPES.DESTINATION, entityId: undefined }]);
        await createDestination({ variables: { destination: { ...destination, fields: destination.fields.filter(({ value }) => value !== undefined) } } });
      }
    },
    updateDestination: async (id, destination) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Updating destination...', undefined, true);
        addPendingItems([{ entityType: ENTITY_TYPES.DESTINATION, entityId: id }]);
        await updateDestination({ variables: { id, destination: { ...destination, fields: destination.fields.filter(({ value }) => value !== undefined) } } });
      }
    },
    deleteDestination: async (id) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Deleting destination...', undefined, true);
        addPendingItems([{ entityType: ENTITY_TYPES.DESTINATION, entityId: id }]);
        await deleteDestination({ variables: { id } });
      }
    },
  };
};
