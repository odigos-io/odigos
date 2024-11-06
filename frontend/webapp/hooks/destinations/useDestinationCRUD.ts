import { useDrawerStore } from '@/store';
import { useNotify } from '../useNotify';
import { useMutation } from '@apollo/client';
import { getSseTargetFromId } from '@/utils';
import { useComputePlatform } from '../compute-platform';
import { OVERVIEW_ENTITY_TYPES, type DestinationInput, type NotificationType } from '@/types';
import { CREATE_DESTINATION, DELETE_DESTINATION, UPDATE_DESTINATION } from '@/graphql/mutations';

interface Params {
  onSuccess?: () => void;
  onError?: () => void;
}

export const useDestinationCRUD = (params?: Params) => {
  const { setSelectedItem: setDrawerItem } = useDrawerStore((store) => store);
  const { refetch } = useComputePlatform();
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
    notifyUser('error', title, message, id);
    params?.onError?.();
  };

  const handleComplete = (title: string, message: string, id?: string) => {
    notifyUser('success', title, message, id);
    setDrawerItem(null);
    refetch();
    params?.onSuccess?.();
  };

  const [createDestination, cState] = useMutation<{ createNewDestination: { id: string } }>(CREATE_DESTINATION, {
    onError: (error) => handleError('Create', error.message),
    onCompleted: (res, req) => {
      const id = res.createNewDestination.id;
      const name = req?.variables?.destination.name || req?.variables?.destination.type;
      handleComplete('Create', `destination "${name}" was created`, id);
    },
  });
  const [updateDestination, uState] = useMutation<{ updateDestination: { id: string } }>(UPDATE_DESTINATION, {
    onError: (error) => handleError('Update', error.message),
    onCompleted: (res, req) => {
      const id = res.updateDestination.id;
      const name = req?.variables?.destination.name || req?.variables?.destination.type;
      handleComplete('Update', `destination "${name}" was updated`, id);
    },
  });
  const [deleteDestination, dState] = useMutation<{ deleteDestination: boolean }>(DELETE_DESTINATION, {
    onError: (error) => handleError('Delete', error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id;
      handleComplete('Delete', `destination "${id}" was deleted`);
    },
  });

  return {
    loading: cState.loading || uState.loading || dState.loading,
    createDestination: (destination: DestinationInput) => createDestination({ variables: { destination } }),
    updateDestination: (id: string, destination: DestinationInput) => updateDestination({ variables: { id, destination } }),
    deleteDestination: (id: string) => deleteDestination({ variables: { id } }),
  };
};
