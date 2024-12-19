import { useMutation } from '@apollo/client';
import { useNotificationStore } from '@/store';
import { ACTION, getSseTargetFromId } from '@/utils';
import { useComputePlatform } from '../compute-platform';
import { NOTIFICATION_TYPE, OVERVIEW_ENTITY_TYPES, type DestinationInput } from '@/types';
import { CREATE_DESTINATION, DELETE_DESTINATION, UPDATE_DESTINATION } from '@/graphql/mutations';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useDestinationCRUD = (params?: Params) => {
  const removeNotifications = useNotificationStore((store) => store.removeNotifications);
  const { data, refetch } = useComputePlatform();
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string) => {
    addNotification({
      type,
      title,
      message,
      crdType: OVERVIEW_ENTITY_TYPES.DESTINATION,
      target: id ? getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.DESTINATION) : undefined,
    });
  };

  const handleError = (title: string, message: string, id?: string) => {
    notifyUser(NOTIFICATION_TYPE.ERROR, title, message, id);
    params?.onError?.(title);
  };

  const handleComplete = (title: string, message: string, id?: string) => {
    notifyUser(NOTIFICATION_TYPE.SUCCESS, title, message, id);
    refetch();
    params?.onSuccess?.(title);
  };

  const [createDestination, cState] = useMutation<{
    createNewDestination: { id: string };
  }>(CREATE_DESTINATION, {
    onError: (error) => handleError(ACTION.CREATE, error.message),
    onCompleted: (res, req) => {
      const id = res.createNewDestination.id;
      const type = req?.variables?.destination.type;
      const name = req?.variables?.destination.name;
      const label = `${type}${!!name ? ` (${name})` : ''}`;
      handleComplete(ACTION.CREATE, `destination "${label}" was created`, id);
    },
  });
  const [updateDestination, uState] = useMutation<{
    updateDestination: { id: string };
  }>(UPDATE_DESTINATION, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: (res, req) => {
      const id = res.updateDestination.id;
      const type = req?.variables?.destination.type;
      const name = req?.variables?.destination.name;
      const label = `${type}${!!name ? ` (${name})` : ''}`;
      handleComplete(ACTION.UPDATE, `destination "${label}" was updated`, id);
    },
  });
  const [deleteDestination, dState] = useMutation<{
    deleteDestination: boolean;
  }>(DELETE_DESTINATION, {
    onError: (error) => handleError(ACTION.DELETE, error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id;
      removeNotifications(getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.DESTINATION));
      handleComplete(ACTION.DELETE, `destination "${id}" was deleted`);
    },
  });

  return {
    loading: cState.loading || uState.loading || dState.loading,
    destinations: data?.computePlatform.destinations || [],

    createDestination: (destination: DestinationInput) => createDestination({ variables: { destination: { ...destination, fields: destination.fields.filter(({ value }) => value !== undefined) } } }),
    updateDestination: (id: string, destination: DestinationInput) => updateDestination({ variables: { id, destination } }),
    deleteDestination: (id: string) => deleteDestination({ variables: { id } }),
  };
};
