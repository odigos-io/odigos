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

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string) => {
    addNotification({
      type,
      title,
      message,
      crdType: OVERVIEW_ENTITY_TYPES.ACTION,
      target: id ? getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.ACTION) : undefined,
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

  const [createAction, cState] = useMutation<{ createAction: { id: string } }>(CREATE_ACTION, {
    onError: (error) => handleError(ACTION.CREATE, error.message),
    onCompleted: (res, req) => {
      const id = res.createAction.id;
      const type = req?.variables?.action.type;
      const name = req?.variables?.action.name;
      const label = `${type}${!!name ? ` (${name})` : ''}`;
      handleComplete(ACTION.CREATE, `action "${label}" was created`, id);
    },
  });
  const [updateAction, uState] = useMutation<{ updateAction: { id: string } }>(UPDATE_ACTION, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: (res, req) => {
      const id = res.updateAction.id;
      const type = req?.variables?.action.type;
      const name = req?.variables?.action.name;
      const label = `${type}${!!name ? ` (${name})` : ''}`;
      handleComplete(ACTION.UPDATE, `action "${label}" was updated`, id);
    },
  });
  const [deleteAction, dState] = useMutation<{ deleteAction: boolean }>(DELETE_ACTION, {
    onError: (error) => handleError(ACTION.DELETE, error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id;
      removeNotifications(getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.ACTION));
      handleComplete(ACTION.DELETE, `action "${id}" was deleted`);
    },
  });

  return {
    loading: cState.loading || uState.loading || dState.loading,
    actions: data?.computePlatform.actions || [],

    createAction: (action: ActionInput) => createAction({ variables: { action } }),
    updateAction: (id: string, action: ActionInput) => updateAction({ variables: { id, action } }),
    deleteAction: (id: string, actionType: ActionsType) => deleteAction({ variables: { id, actionType } }),
  };
};
