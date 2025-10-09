import { useEffect } from 'react';
import { useConfig } from '../config';
import { GET_ACTIONS } from '@/graphql';
import { ActionInput, FetchedAction } from '@/types';
import { useLazyQuery, useMutation } from '@apollo/client';
import { getSseTargetFromId } from '@odigos/ui-kit/functions';
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

const stringifyRenames = (action: ActionFormData): ActionInput => {
  const sanitizeFromArray = <T extends { from?: string | null }>(arr?: T[]) => {
    if (!Array.isArray(arr)) return arr as unknown as T[] | undefined;
    return arr.map((item) => {
      const fromVal = (item as { from?: unknown }).from as string | undefined;
      const hasValidFrom = typeof fromVal === 'string' && fromVal.trim().length > 0;
      if (!hasValidFrom) {
        const { from, ...rest } = item as Record<string, unknown>;
        return rest as T;
      }
      return item;
    });
  };

  return {
    ...action,
    fields: {
      ...action.fields,
      labelsAttributes: sanitizeFromArray(action.fields.labelsAttributes as any),
      annotationsAttributes: sanitizeFromArray(action.fields.annotationsAttributes as any),
      renames: action.fields.renames ? JSON.stringify(action.fields.renames) : null,
    },
  };
};

const parseRenames = (action: FetchedAction): Action => {
  return {
    ...action,
    fields: {
      ...action.fields,
      renames: action.fields.renames ? JSON.parse(action.fields.renames) : null,
    },
  };
};

export const useActionCRUD = (): UseActionCrud => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();
  const { actionsLoading, setEntitiesLoading, actions, setEntities, removeEntities } = useEntityStore();

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

      setEntities(EntityTypes.Action, items.map(parseRenames));
      setEntitiesLoading(EntityTypes.Action, false);
    }
  };

  const [mutateCreate] = useMutation<{ createAction: FetchedAction }, { action: ActionInput }>(CREATE_ACTION, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Create, error.cause?.message || error.message),
    onCompleted: (res) => {
      const action = res.createAction;
      const { id, type } = action;
      // addEntities(EntityTypes.Action, [action]);
      fetchActions(); // refetch because of conditions
      notifyUser(StatusType.Success, Crud.Create, `Successfully created "${type}" action`, id);
    },
  });

  const [mutateUpdate] = useMutation<{ updateAction: FetchedAction }, { id: string; action: ActionInput }>(UPDATE_ACTION, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message),
    onCompleted: (res) => {
      const action = res.updateAction;
      const { id, type } = action;
      // addEntities(EntityTypes.Action, [action]);
      fetchActions(); // refetch because of conditions
      notifyUser(StatusType.Success, Crud.Update, `Successfully updated "${type}" action`, id);
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
      mutateCreate({ variables: { action: stringifyRenames(action) } });
    }
  };

  const updateAction: UseActionCrud['updateAction'] = (id, action) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateUpdate({ variables: { id, action: stringifyRenames(action) } });
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
