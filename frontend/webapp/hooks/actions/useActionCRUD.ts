import { useDrawerStore } from '@/store';
import { useNotify } from '../useNotify';
import { useMutation } from '@apollo/client';
import type { ActionInput, ActionsType } from '@/types';
import { useComputePlatform } from '../compute-platform';
import { CREATE_ACTION, DELETE_ACTION, UPDATE_ACTION } from '@/graphql/mutations';

interface UseActionCrudParams {
  onSuccess?: () => void;
  onError?: () => void;
}

export const useActionCRUD = (params?: UseActionCrudParams) => {
  const { setSelectedItem: setDrawerItem } = useDrawerStore((store) => store);
  const { refetch } = useComputePlatform();
  const notify = useNotify();

  const handleError = (title: string, message: string) => {
    notify({
      title,
      message,
      type: 'error',
      target: 'notification',
      crdType: 'notification',
    });

    if (params?.onError) params.onError();
  };

  const handleComplete = (title: string, message: string) => {
    setDrawerItem(null);
    refetch();
    notify({
      title,
      message,
      type: 'success',
      target: 'notification',
      crdType: 'notification',
    });

    if (params?.onSuccess) params.onSuccess();
  };

  const [createAction, cState] = useMutation(CREATE_ACTION, {
    onError: (error) => {
      handleError('Create Action', error.message);
    },
    onCompleted: () => {
      handleComplete('Create Action', 'successfully created');
    },
  });

  const [updateAction, uState] = useMutation(UPDATE_ACTION, {
    onError: (error) => {
      handleError('Update Action', error.message);
    },
    onCompleted: () => {
      handleComplete('Update Action', 'successfully updated');
    },
  });

  const [deleteAction, dState] = useMutation(DELETE_ACTION, {
    onError: (error) => {
      handleError('Delete Action', error.message);
    },
    onCompleted: () => {
      handleComplete('Delete Action', 'successfully deleted');
    },
  });

  return {
    loading: cState.loading || uState.loading || dState.loading,
    createAction: (action: ActionInput) => createAction({ variables: { action } }),
    updateAction: (id: string, action: ActionInput) => updateAction({ variables: { id, action } }),
    deleteAction: (id: string, actionType: ActionsType) => deleteAction({ variables: { id, actionType } }),
  };
};
