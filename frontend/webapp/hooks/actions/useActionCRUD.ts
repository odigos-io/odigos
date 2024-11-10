import { useDrawerStore } from '@/store';
import { useNotify } from '../notification/useNotify';
import { useMutation } from '@apollo/client';
import { useComputePlatform } from '../compute-platform';
import { ACTION, getSseTargetFromId, NOTIFICATION } from '@/utils';
import {
  CREATE_ACTION,
  DELETE_ACTION,
  UPDATE_ACTION,
} from '@/graphql/mutations';
import {
  OVERVIEW_ENTITY_TYPES,
  type ActionInput,
  type ActionsType,
  type NotificationType,
} from '@/types';

interface UseActionCrudParams {
  onSuccess?: () => void;
  onError?: () => void;
}

export const useActionCRUD = (params?: UseActionCrudParams) => {
  const { setSelectedItem: setDrawerItem } = useDrawerStore((store) => store);
  const { refetch } = useComputePlatform();
  const notify = useNotify();

  const notifyUser = (
    type: NotificationType,
    title: string,
    message: string,
    id?: string
  ) => {
    notify({
      type,
      title,
      message,
      crdType: OVERVIEW_ENTITY_TYPES.ACTION,
      target: id
        ? getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.ACTION)
        : undefined,
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

  const [createAction, cState] = useMutation<{ createAction: { id: string } }>(
    CREATE_ACTION,
    {
      onError: (error) => handleError(ACTION.CREATE, error.message),
      onCompleted: (res, req) => {
        const id = res.createAction.id;
        const name = req?.variables?.action.name || req?.variables?.action.type;
        handleComplete(ACTION.CREATE, `action "${name}" was created`, id);
      },
    }
  );
  const [updateAction, uState] = useMutation<{ updateAction: { id: string } }>(
    UPDATE_ACTION,
    {
      onError: (error) => handleError(ACTION.UPDATE, error.message),
      onCompleted: (res, req) => {
        const id = res.updateAction.id;
        const name = req?.variables?.action.name || req?.variables?.action.type;
        handleComplete(ACTION.UPDATE, `action "${name}" was updated`, id);
      },
    }
  );
  const [deleteAction, dState] = useMutation<{ deleteAction: boolean }>(
    DELETE_ACTION,
    {
      onError: (error) => handleError(ACTION.DELETE, error.message),
      onCompleted: (res, req) => {
        const id = req?.variables?.id;
        handleComplete(ACTION.DELETE, `action "${id}" was deleted`);
      },
    }
  );

  return {
    loading: cState.loading || uState.loading || dState.loading,
    createAction: (action: ActionInput) =>
      createAction({ variables: { action } }),
    updateAction: (id: string, action: ActionInput) =>
      updateAction({ variables: { id, action } }),
    deleteAction: (id: string, actionType: ActionsType) =>
      deleteAction({ variables: { id, actionType } }),
  };
};
