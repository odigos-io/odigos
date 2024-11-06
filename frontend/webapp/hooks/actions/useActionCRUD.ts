import { useDrawerStore } from '@/store';
import { useNotify } from '../useNotify';
import { useMutation } from '@apollo/client';
import type { ActionInput, ActionsType, NotificationType } from '@/types';
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

  const [createAction, cState] = useMutation<{ createAction: { id: string } }>(CREATE_ACTION, {
    onError: (error) => handleError('Create', error.message),
    onCompleted: (_, req) => {
      const name = req?.variables?.action.name || req?.variables?.action.type;
      handleComplete('Create', `action "${name}" was created`);
    },
  });
  const [updateAction, uState] = useMutation<{ updateAction: { id: string } }>(UPDATE_ACTION, {
    onError: (error) => handleError('Update', error.message),
    onCompleted: (_, req) => {
      const name = req?.variables?.action.name || req?.variables?.action.type;
      handleComplete('Update', `action "${name}" was updated`);
    },
  });
  const [deleteAction, dState] = useMutation<{ deleteAction: boolean }>(DELETE_ACTION, {
    onError: (error) => handleError('Delete', error.message),
    onCompleted: (_, req) => {
      const name = req?.variables?.id;
      handleComplete('Delete', `action "${name}" was deleted`);
    },
  });

  return {
    loading: cState.loading || uState.loading || dState.loading,
    createAction: (action: ActionInput) => createAction({ variables: { action } }),
    updateAction: (id: string, action: ActionInput) => updateAction({ variables: { id, action } }),
    deleteAction: (id: string, actionType: ActionsType) => deleteAction({ variables: { id, actionType } }),
  };
};
