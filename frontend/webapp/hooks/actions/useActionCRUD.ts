import { useEffect } from 'react';
import { useConfig } from '../config';
import { GET_ACTIONS } from '@/graphql';
import type { ActionInput, FetchedAction } from '@/types';
import { useLazyQuery, useMutation } from '@apollo/client';
import { getSseTargetFromId } from '@odigos/ui-kit/functions';
import { mapActionsFormToGqlInput, mapFetchedActions } from '@/utils';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { useEntityStore, useNotificationStore } from '@odigos/ui-kit/store';
import { CREATE_ACTION, DELETE_ACTION, UPDATE_ACTION } from '@/graphql/mutations';
import { ACTION_TYPE, CRUD, ENTITY_TYPES, STATUS_TYPE, type Action, type ActionFormData } from '@odigos/ui-kit/types';

interface UseActionCrud {
  actions: Action[];
  actionsLoading: boolean;
  fetchActions: () => void;
  createAction: (action: ActionFormData) => void;
  updateAction: (id: string, action: ActionFormData) => void;
  deleteAction: (id: string, actionType: ACTION_TYPE) => void;
}

export const useActionCRUD = (): UseActionCrud => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();
  const { actionsLoading, setEntitiesLoading, actions, addEntities, removeEntities } = useEntityStore();

  const notifyUser = (type: STATUS_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: ENTITY_TYPES.ACTION, target: id ? getSseTargetFromId(id, ENTITY_TYPES.ACTION) : undefined, hideFromHistory });
  };

  const [fetchAll] = useLazyQuery<{ computePlatform?: { actions?: FetchedAction[] } }>(GET_ACTIONS, {
    fetchPolicy: 'cache-and-network',
  });

  const fetchActions = async () => {
    setEntitiesLoading(ENTITY_TYPES.ACTION, true);
    const { error, data } = await fetchAll();

    if (error) {
      notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message);
    } else if (data?.computePlatform?.actions) {
      const { actions: items } = data.computePlatform;

      addEntities(ENTITY_TYPES.ACTION, mapFetchedActions(items));
      setEntitiesLoading(ENTITY_TYPES.ACTION, false);
    }
  };

  const [mutateCreate] = useMutation<{ createAction: { id: string; type: ACTION_TYPE } }, { action: ActionInput }>(CREATE_ACTION, {
    onError: (error) => notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.CREATE, error.cause?.message || error.message),
    onCompleted: (res) => {
      const id = res.createAction.id;
      const type = res.createAction.type;
      notifyUser(STATUS_TYPE.SUCCESS, CRUD.CREATE, `Successfully created "${type}" action`, id);
      fetchActions();
    },
  });

  const [mutateUpdate] = useMutation<{ updateAction: { id: string; type: ACTION_TYPE } }, { id: string; action: ActionInput }>(UPDATE_ACTION, {
    onError: (error) => notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message),
    onCompleted: (res) => {
      const id = res.updateAction.id;
      const type = res.updateAction.type;
      notifyUser(STATUS_TYPE.SUCCESS, CRUD.UPDATE, `Successfully updated "${type}" action`, id);
      fetchActions();
    },
  });

  const [mutateDelete] = useMutation<{ deleteAction: boolean }, { id: string; actionType: ACTION_TYPE }>(DELETE_ACTION, {
    onError: (error) => notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.DELETE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id as string;
      const type = req?.variables?.actionType;
      removeEntities(ENTITY_TYPES.ACTION, [id]);
      notifyUser(STATUS_TYPE.SUCCESS, CRUD.DELETE, `Successfully deleted "${type}" action`, id);
    },
  });

  const createAction: UseActionCrud['createAction'] = (action) => {
    if (isReadonly) {
      notifyUser(STATUS_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateCreate({ variables: { action: mapActionsFormToGqlInput({ ...action }) } });
    }
  };

  const updateAction: UseActionCrud['updateAction'] = (id, action) => {
    if (isReadonly) {
      notifyUser(STATUS_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateUpdate({ variables: { id, action: mapActionsFormToGqlInput({ ...action }) } });
    }
  };

  const deleteAction: UseActionCrud['deleteAction'] = (id, actionType) => {
    if (isReadonly) {
      notifyUser(STATUS_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateDelete({ variables: { id, actionType } });
    }
  };

  useEffect(() => {
    if (!actions.length && !actionsLoading) fetchActions();
  }, []);

  return {
    actions,
    actionsLoading,
    fetchActions,
    createAction,
    updateAction,
    deleteAction,
  };
};
