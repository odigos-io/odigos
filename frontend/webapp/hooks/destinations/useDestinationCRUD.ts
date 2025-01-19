import { useMemo } from 'react';
import { GET_DESTINATIONS } from '@/graphql';
import { ACTION, getSseTargetFromId } from '@/utils';
import { useMutation, useQuery } from '@apollo/client';
import { useFilterStore, useNotificationStore, usePendingStore } from '@/store';
import { CREATE_DESTINATION, DELETE_DESTINATION, UPDATE_DESTINATION } from '@/graphql/mutations';
import { NOTIFICATION_TYPE, OVERVIEW_ENTITY_TYPES, type SupportedSignals, type DestinationInput, type ComputePlatform } from '@/types';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useDestinationCRUD = (params?: Params) => {
  const filters = useFilterStore();
  const { addPendingItems } = usePendingStore();
  const { addNotification, removeNotifications } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({
      type,
      title,
      message,
      crdType: OVERVIEW_ENTITY_TYPES.DESTINATION,
      target: id ? getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.DESTINATION) : undefined,
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
    onError: (error) => handleError(error.name || ACTION.FETCH, error.cause?.message || error.message),
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
    if (!!filters.monitors.length) arr = arr.filter((destination) => !!filters.monitors.find((metric) => destination.exportedSignals[metric.id as keyof SupportedSignals]));
    return arr;
  }, [mapped, filters]);

  const [createDestination, cState] = useMutation<{ createNewDestination: { id: string } }>(CREATE_DESTINATION, {
    onError: (error) => handleError(ACTION.CREATE, error.message),
    onCompleted: () => handleComplete(ACTION.CREATE),
  });

  const [updateDestination, uState] = useMutation<{ updateDestination: { id: string } }>(UPDATE_DESTINATION, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: () => handleComplete(ACTION.UPDATE),
  });

  const [deleteDestination, dState] = useMutation<{ deleteDestination: boolean }>(DELETE_DESTINATION, {
    onError: (error) => handleError(ACTION.DELETE, error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id;
      removeNotifications(getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.DESTINATION));
      handleComplete(ACTION.DELETE);
    },
  });

  return {
    loading: loading || cState.loading || uState.loading || dState.loading,
    destinations: mapped,
    filteredDestinations: filtered,
    refetchDestinations: refetch,

    createDestination: (destination: DestinationInput) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Creating destination...', undefined, true);
      addPendingItems([{ entityType: OVERVIEW_ENTITY_TYPES.DESTINATION, entityId: undefined }]);
      createDestination({ variables: { destination: { ...destination, fields: destination.fields.filter(({ value }) => value !== undefined) } } });
    },
    updateDestination: (id: string, destination: DestinationInput) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Updating destination...', undefined, true);
      addPendingItems([{ entityType: OVERVIEW_ENTITY_TYPES.DESTINATION, entityId: id }]);
      updateDestination({ variables: { id, destination: { ...destination, fields: destination.fields.filter(({ value }) => value !== undefined) } } });
    },
    deleteDestination: (id: string) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Deleting destination...', undefined, true);
      addPendingItems([{ entityType: OVERVIEW_ENTITY_TYPES.DESTINATION, entityId: id }]);
      deleteDestination({ variables: { id } });
    },
  };
};
