import { useMutation } from '@apollo/client';
import { ACTION, getSseTargetFromId } from '@/utils';
import { useComputePlatform } from '../compute-platform';
import { useNotificationStore, usePendingStore } from '@/store';
import { NOTIFICATION_TYPE, OVERVIEW_ENTITY_TYPES, type DestinationInput } from '@/types';
import { CREATE_DESTINATION, DELETE_DESTINATION, UPDATE_DESTINATION } from '@/graphql/mutations';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useDestinationCRUD = (params?: Params) => {
  const { data } = useComputePlatform();
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
    loading: cState.loading || uState.loading || dState.loading,
    destinations: data?.computePlatform?.destinations || [],

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
