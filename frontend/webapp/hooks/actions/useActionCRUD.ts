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
import { ActionType, Crud, EntityTypes, StatusType, type Action, type ActionFormData } from '@odigos/ui-kit/types';

interface UseActionCrud {
  actions: Action[];
  actionsLoading: boolean;
  fetchActions: () => void;
  createAction: (action: ActionFormData) => void;
  updateAction: (id: string, action: ActionFormData) => void;
  deleteAction: (id: string, actionType: ActionType) => void;
}

export const useActionCRUD = (): UseActionCrud => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();
  const { actionsLoading, setEntitiesLoading, actions, addEntities, removeEntities } = useEntityStore();

  const notifyUser = (type: StatusType, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: EntityTypes.Action, target: id ? getSseTargetFromId(id, EntityTypes.Action) : undefined, hideFromHistory });
  };

  const [fetchAll] = useLazyQuery<{ computePlatform?: { actions?: FetchedAction[] } }>(GET_ACTIONS);

  const fetchActions = async () => {
    setEntitiesLoading(EntityTypes.Action, true);
    const { error, data } = await fetchAll();

    if (error) {
      notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message);
    } else if (data?.computePlatform?.actions) {
      const { actions: items } = data.computePlatform;

      addEntities(EntityTypes.Action, mapFetchedActions(items));
      setEntitiesLoading(EntityTypes.Action, false);
    }
  };

  const [mutateCreate] = useMutation<{ createAction: { id: string; type: ActionType } }, { action: ActionInput }>(CREATE_ACTION, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Create, error.cause?.message || error.message),
    onCompleted: (res) => {
      const id = res.createAction.id;
      const type = res.createAction.type;
      notifyUser(StatusType.Success, Crud.Create, `Successfully created "${type}" action`, id);
      fetchActions();
    },
  });

  const [mutateUpdate] = useMutation<{ updateAction: { id: string; type: ActionType } }, { id: string; action: ActionInput }>(UPDATE_ACTION, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message),
    onCompleted: (res) => {
      const id = res.updateAction.id;
      const type = res.updateAction.type;
      notifyUser(StatusType.Success, Crud.Update, `Successfully updated "${type}" action`, id);
      fetchActions();
    },
  });

  const [mutateDelete] = useMutation<{ deleteAction: boolean }, { id: string; actionType: ActionType }>(DELETE_ACTION, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Delete, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id as string;
      const type = req?.variables?.actionType;
      removeEntities(EntityTypes.Action, [id]);
      notifyUser(StatusType.Success, Crud.Delete, `Successfully deleted "${type}" action`, id);
    },
  });

  const createAction: UseActionCrud['createAction'] = (action) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateCreate({ variables: { action: mapActionsFormToGqlInput({ ...action }) } });
    }
  };

  const updateAction: UseActionCrud['updateAction'] = (id, action) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateUpdate({ variables: { id, action: mapActionsFormToGqlInput({ ...action }) } });
    }
  };

  const deleteAction: UseActionCrud['deleteAction'] = (id, actionType) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
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
