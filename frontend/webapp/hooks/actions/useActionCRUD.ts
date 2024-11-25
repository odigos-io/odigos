import { useMutation } from '@apollo/client';
import { useNotificationStore } from '@/store';
import { useNotify } from '../notification/useNotify';
import { useComputePlatform } from '../compute-platform';
import { ACTION, getSseTargetFromId, NOTIFICATION, safeJsonParse } from '@/utils';
import { CREATE_ACTION, DELETE_ACTION, UPDATE_ACTION } from '@/graphql/mutations';
import { type ActionItem, OVERVIEW_ENTITY_TYPES, type ActionInput, type ActionsType, type NotificationType } from '@/types';

interface UseActionCrudParams {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useActionCRUD = (params?: UseActionCrudParams) => {
  const removeNotifications = useNotificationStore((store) => store.removeNotifications);
  const { data, refetch } = useComputePlatform();
  const notify = useNotify();

  const notifyUser = (type: NotificationType, title: string, message: string, id?: string) => {
    notify({
      type,
      title,
      message,
      crdType: OVERVIEW_ENTITY_TYPES.ACTION,
      target: id ? getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.ACTION) : undefined,
    });
  };

  const handleError = (title: string, message: string, id?: string) => {
    notifyUser(NOTIFICATION.ERROR, title, message, id);
    params?.onError?.(title);
  };

  const handleComplete = (title: string, message: string, id?: string) => {
    notifyUser(NOTIFICATION.SUCCESS, title, message, id);
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
    actions:
      data?.computePlatform?.actions?.map((item) => {
        const parsedSpec = typeof item.spec === 'string' ? safeJsonParse(item.spec, {} as ActionItem) : item.spec;

        return { ...item, spec: parsedSpec };
      }) || [],

    createAction: (action: ActionInput) => createAction({ variables: { action } }),
    updateAction: (id: string, action: ActionInput) => updateAction({ variables: { id, action } }),
    deleteAction: (id: string, actionType: ActionsType) => deleteAction({ variables: { id, actionType } }),
  };
};
