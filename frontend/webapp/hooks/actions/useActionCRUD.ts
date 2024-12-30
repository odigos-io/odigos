import { useMutation } from '@apollo/client';
import { useNotificationStore } from '@/store';
import { ACTION, getSseTargetFromId } from '@/utils';
import { useComputePlatform } from '../compute-platform';
import { CREATE_ACTION, DELETE_ACTION, UPDATE_ACTION } from '@/graphql/mutations';
import { NOTIFICATION_TYPE, OVERVIEW_ENTITY_TYPES, type ActionInput, type ActionsType } from '@/types';

interface UseActionCrudParams {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useActionCRUD = (params?: UseActionCrudParams) => {
  const removeNotifications = useNotificationStore((store) => store.removeNotifications);
  const { data, refetch } = useComputePlatform();
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({
      type,
      title,
      message,
      crdType: OVERVIEW_ENTITY_TYPES.ACTION,
      target: id ? getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.ACTION) : undefined,
      hideFromHistory,
    });
  };

  const handleError = (actionType: string, message: string) => {
    notifyUser(NOTIFICATION_TYPE.ERROR, actionType, message);
    params?.onError?.(actionType);
  };

  const handleComplete = (actionType: string) => {
    refetch();
    params?.onSuccess?.(actionType);
  };

  const [createAction, cState] = useMutation<{ createAction: { id: string } }>(CREATE_ACTION, {
    onError: (error) => handleError(ACTION.CREATE, error.message),
    onCompleted: () => handleComplete(ACTION.CREATE),
  });
  const [updateAction, uState] = useMutation<{ updateAction: { id: string } }>(UPDATE_ACTION, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: () => handleComplete(ACTION.UPDATE),
  });
  const [deleteAction, dState] = useMutation<{ deleteAction: boolean }>(DELETE_ACTION, {
    onError: (error) => handleError(ACTION.DELETE, error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id;
      removeNotifications(getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.ACTION));
      handleComplete(ACTION.DELETE);
    },
  });

  return {
    loading: cState.loading || uState.loading || dState.loading,
    actions: data?.computePlatform?.actions || [],

    createAction: (action: ActionInput) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'creating pipeline action...', undefined, true);
      createAction({ variables: { action } });
    },
    updateAction: (id: string, action: ActionInput) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'updating pipeline action...', undefined, true);
      updateAction({ variables: { id, action } });
    },
    deleteAction: (id: string, actionType: ActionsType) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'deleting pipeline action...', undefined, true);
      deleteAction({ variables: { id, actionType } });
    },
  };
};
