import { useDrawerStore } from '@/store';
import { useNotify } from '../useNotify';
import { useMutation } from '@apollo/client';
import { useComputePlatform } from '../compute-platform';
import type { DestinationInput, NotificationType } from '@/types';
import { CREATE_DESTINATION, DELETE_DESTINATION, UPDATE_DESTINATION } from '@/graphql/mutations';

interface Params {
  onSuccess?: () => void;
  onError?: () => void;
}

export const useDestinationCRUD = (params?: Params) => {
  const { setSelectedItem: setDrawerItem } = useDrawerStore((store) => store);
  const { refetch } = useComputePlatform();
  const notify = useNotify();

  const notifyUser = (type: NotificationType, title: string, message: string) => {
    notify({ type, title, message });
  };

  const handleError = (title: string, message: string) => {
    notifyUser('error', title, message);
    params?.onError?.();
  };

  const handleComplete = (title: string, message: string) => {
    notifyUser('success', title, message);
    setDrawerItem(null);
    refetch();
    params?.onSuccess?.();
  };

  const [createDestination, cState] = useMutation<{ createNewDestination: { id: string } }>(CREATE_DESTINATION, {
    onError: (error) => handleError('Create', error.message),
    onCompleted: (_, req) => {
      const name = req?.variables?.destination.name || req?.variables?.destination.type;
      handleComplete('Create', `destination "${name}" was created`);
    },
  });
  const [updateDestination, uState] = useMutation<{ updateDestination: { id: string } }>(UPDATE_DESTINATION, {
    onError: (error) => handleError('Update', error.message),
    onCompleted: (_, req) => {
      const name = req?.variables?.destination.name || req?.variables?.destination.type;
      handleComplete('Update', `destination "${name}" was updated`);
    },
  });
  const [deleteDestination, dState] = useMutation<{ deleteDestination: boolean }>(DELETE_DESTINATION, {
    onError: (error) => handleError('Delete', error.message),
    onCompleted: (_, req) => {
      const name = req?.variables?.id;
      handleComplete('Delete', `destination "${name}" was deleted`);
    },
  });

  return {
    loading: cState.loading || uState.loading || dState.loading,
    createDestination: (destination: DestinationInput) => createDestination({ variables: { destination } }),
    updateDestination: (id: string, destination: DestinationInput) => updateDestination({ variables: { id, destination } }),
    deleteDestination: (id: string) => deleteDestination({ variables: { id } }),
  };
};
