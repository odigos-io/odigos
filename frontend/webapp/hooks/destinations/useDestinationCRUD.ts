import { useDrawerStore } from '@/store';
import { useNotify } from '../useNotify';
import { useMutation } from '@apollo/client';
import type { DestinationInput } from '@/types';
import { useComputePlatform } from '../compute-platform';
import { CREATE_DESTINATION, DELETE_DESTINATION, UPDATE_DESTINATION } from '@/graphql/mutations';

interface Params {
  onSuccess?: () => void;
  onError?: () => void;
}

export const useDestinationCRUD = (params?: Params) => {
  const { setSelectedItem: setDrawerItem } = useDrawerStore((store) => store);
  const { refetch } = useComputePlatform();
  const notify = useNotify();

  const notifyUser = (title: string, message: string, type: 'error' | 'success') => {
    notify({ title, message, type, target: 'notification', crdType: 'notification' });
  };

  const handleError = (title: string, message: string) => {
    notifyUser(title, message, 'error');
    params?.onError?.();
  };

  const handleComplete = (title: string, message: string) => {
    notifyUser(title, message, 'success');
    setDrawerItem(null);
    refetch();
    params?.onSuccess?.();
  };

  const [createDestination, cState] = useMutation<{ createNewDestination: { id: string } }>(CREATE_DESTINATION, {
    onError: (error) => handleError('Create Destination', error.message),
    onCompleted: () => handleComplete('Create Destination', 'successfully created'),
  });
  const [updateDestination, uState] = useMutation<{ updateDestination: { id: string } }>(UPDATE_DESTINATION, {
    onError: (error) => handleError('Update Destination', error.message),
    onCompleted: () => handleComplete('Update Destination', 'successfully updated'),
  });
  const [deleteDestination, dState] = useMutation<{ deleteDestination: boolean }>(DELETE_DESTINATION, {
    onError: (error) => handleError('Delete Destination', error.message),
    onCompleted: () => handleComplete('Delete Destination', 'successfully deleted'),
  });

  return {
    loading: cState.loading || uState.loading || dState.loading,
    createDestination: (destination: DestinationInput) => createDestination({ variables: { destination } }),
    updateDestination: (id: string, destination: DestinationInput) => updateDestination({ variables: { id, destination } }),
    deleteDestination: (id: string) => deleteDestination({ variables: { id } }),
  };
};
