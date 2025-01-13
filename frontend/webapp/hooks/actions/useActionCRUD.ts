import { useMemo } from 'react';
import { useMutation } from '@apollo/client';
import { ACTION, getSseTargetFromId } from '@/utils';
import { useComputePlatform } from '../compute-platform';
import { useFilterStore, useNotificationStore } from '@/store';
import { CREATE_ACTION, DELETE_ACTION, UPDATE_ACTION } from '@/graphql/mutations';
import { NOTIFICATION_TYPE, OVERVIEW_ENTITY_TYPES, type ActionInput, type ActionsType } from '@/types';

interface UseActionCrudParams {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useActionCRUD = (params?: UseActionCrudParams) => {
  const filters = useFilterStore();
  const { data, loading, refetch } = useComputePlatform();
  const { addNotification, removeNotifications } = useNotificationStore();

  const actions = data?.computePlatform?.actions || [];

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

  const handleComplete = (actionType: string, message: string, id?: string) => {
    notifyUser(NOTIFICATION_TYPE.SUCCESS, actionType, message, id);
    refetch();
    params?.onSuccess?.(actionType);
  };

  const [createAction, cState] = useMutation<{ createAction: { id: string } }>(CREATE_ACTION, {
    onError: (error) => handleError(ACTION.CREATE, error.message),
    onCompleted: (res, req) => {
      const id = res?.createAction?.id;
      handleComplete(ACTION.CREATE, `Action "${id}" created`, id);
    },
  });
  const [updateAction, uState] = useMutation<{ updateAction: { id: string } }>(UPDATE_ACTION, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: (res, req) => {
      const id = res?.updateAction?.id;
      handleComplete(ACTION.UPDATE, `Action "${id}" updated`, id);
    },
  });
  const [deleteAction, dState] = useMutation<{ deleteAction: boolean }>(DELETE_ACTION, {
    onError: (error) => handleError(ACTION.DELETE, error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id;
      removeNotifications(getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.ACTION));
      handleComplete(ACTION.DELETE, `Action "${id}" deleted`, id);
    },
  });

  const filtered = useMemo(() => {
    let arr = [...actions];

    if (!!filters.monitors.length) arr = arr.filter((action) => !!filters.monitors.find((metric) => action.spec.signals.find((str) => str.toLowerCase() === metric.id)));

    return arr;
  }, [actions, filters]);

  return {
    loading: loading || cState.loading || uState.loading || dState.loading,
    actions,
    filteredActions: filtered,

    createAction: (action: ActionInput) => {
      createAction({ variables: { action } });
    },
    updateAction: (id: string, action: ActionInput) => {
      updateAction({ variables: { id, action } });
    },
    deleteAction: (id: string, actionType: ActionsType) => {
      deleteAction({ variables: { id, actionType } });
    },
  };
};
