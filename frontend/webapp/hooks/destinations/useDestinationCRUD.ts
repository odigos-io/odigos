import { useDrawerStore } from '@/store';
import { useNotify } from '../notification/useNotify';
import { useMutation } from '@apollo/client';
import { useComputePlatform } from '../compute-platform';
import { ACTION, getSseTargetFromId, NOTIFICATION } from '@/utils';
import { OVERVIEW_ENTITY_TYPES, type DestinationInput, type NotificationType } from '@/types';
import { CREATE_DESTINATION, DELETE_DESTINATION, UPDATE_DESTINATION } from '@/graphql/mutations';

interface Params {
  onSuccess?: () => void;
  onError?: () => void;
}

export const useDestinationCRUD = (params?: Params) => {
  const { setSelectedItem: setDrawerItem } = useDrawerStore((store) => store);
  const { data, refetch } = useComputePlatform();
  const notify = useNotify();

  const notifyUser = (type: NotificationType, title: string, message: string, id?: string) => {
    notify({
      type,
      title,
      message,
      crdType: OVERVIEW_ENTITY_TYPES.DESTINATION,
      target: id ? getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.DESTINATION) : undefined,
    });
  };

  const handleError = (title: string, message: string, id?: string) => {
    notifyUser(NOTIFICATION.ERROR, title, message, id);
    params?.onError?.();
  };

  const handleComplete = (title: string, message: string, id?: string) => {
    notifyUser(NOTIFICATION.SUCCESS, title, message, id);
    setDrawerItem(null);
    refetch();
    params?.onSuccess?.();
  };

  const [createDestination, cState] = useMutation<{
    createNewDestination: { id: string };
  }>(CREATE_DESTINATION, {
    onError: (error) => handleError(ACTION.CREATE, error.message),
    onCompleted: (res, req) => {
      const id = res.createNewDestination.id;
      const name = req?.variables?.destination.name || req?.variables?.destination.type;
      handleComplete(ACTION.CREATE, `destination "${name}" was created`, id);
    },
  });
  const [updateDestination, uState] = useMutation<{
    updateDestination: { id: string };
  }>(UPDATE_DESTINATION, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: (res, req) => {
      const id = res.updateDestination.id;
      const name = req?.variables?.destination.name || req?.variables?.destination.type;
      handleComplete(ACTION.UPDATE, `destination "${name}" was updated`, id);
    },
  });
  const [deleteDestination, dState] = useMutation<{
    deleteDestination: boolean;
  }>(DELETE_DESTINATION, {
    onError: (error) => handleError(ACTION.DELETE, error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id;
      handleComplete(ACTION.DELETE, `destination "${id}" was deleted`);
    },
  });

  return {
    loading: cState.loading || uState.loading || dState.loading,
    destinations: data?.computePlatform.destinations || [],

    createDestination: (destination: DestinationInput) => createDestination({ variables: { destination } }),
    updateDestination: (id: string, destination: DestinationInput) => updateDestination({ variables: { id, destination } }),
    deleteDestination: (id: string) => deleteDestination({ variables: { id } }),
  };
};
