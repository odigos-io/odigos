import { useMemo } from 'react';
import { GET_ACTIONS } from '@/graphql';
import { useMutation, useQuery } from '@apollo/client';
import { useFilterStore, useNotificationStore } from '@/store';
import { ACTION, getSseTargetFromId, safeJsonParse } from '@/utils';
import { CREATE_ACTION, DELETE_ACTION, UPDATE_ACTION } from '@/graphql/mutations';
import { type ActionItem, type ComputePlatform, NOTIFICATION_TYPE, OVERVIEW_ENTITY_TYPES, type ActionInput, type ActionsType } from '@/types';

interface UseActionCrudParams {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useActionCRUD = (params?: UseActionCrudParams) => {
  const filters = useFilterStore();
  const { addNotification, removeNotifications } = useNotificationStore();

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

  // Fetch data
  const { data, loading, refetch } = useQuery<ComputePlatform>(GET_ACTIONS, {
    onError: (error) => handleError(error.name || ACTION.FETCH, error.cause?.message || error.message),
  });

  // Map fetched data
  const mapped = useMemo(() => {
    return (data?.computePlatform?.actions || []).map((item) => {
      const parsedSpec = typeof item.spec === 'string' ? safeJsonParse(item.spec, {} as ActionItem) : item.spec;
      return { ...item, spec: parsedSpec };
    });
  }, [data]);

  // Filter mapped data
  const filtered = useMemo(() => {
    let arr = [...mapped];
    if (!!filters.monitors.length) arr = arr.filter((action) => !!filters.monitors.find((metric) => action.spec.signals.find((str) => str.toLowerCase() === metric.id)));
    return arr;
  }, [mapped, filters]);

  const [createAction, cState] = useMutation<{ createAction: { id: string } }>(CREATE_ACTION, {
    onError: (error) => handleError(ACTION.CREATE, error.message),
    onCompleted: (res) => {
      const id = res?.createAction?.id;
      handleComplete(ACTION.CREATE, `Action "${id}" created`, id);
    },
  });

  const [updateAction, uState] = useMutation<{ updateAction: { id: string } }>(UPDATE_ACTION, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: (res) => {
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

  return {
    loading: loading || cState.loading || uState.loading || dState.loading,
    actions: mapped,
    filteredActions: filtered,
    refetchActions: refetch,

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
