import { useConfig } from '../config';
import { GET_ACTIONS } from '@/graphql';
import { useMutation, useQuery } from '@apollo/client';
import type { ActionInput, ParsedActionSpec, FetchedAction } from '@/@types';
import { CREATE_ACTION, DELETE_ACTION, UPDATE_ACTION } from '@/graphql/mutations';
import { type ActionFormData, useNotificationStore } from '@odigos/ui-containers';
import { type Action, ACTION_TYPE, CRUD, DISPLAY_TITLES, ENTITY_TYPES, FORM_ALERTS, getSseTargetFromId, NOTIFICATION_TYPE, safeJsonParse, SIGNAL_TYPE } from '@odigos/ui-utils';

interface UseActionCrud {
  actions: Action[];
  actionsLoading: boolean;
  fetchActions: () => void;
  createAction: (action: ActionFormData) => void;
  updateAction: (id: string, action: ActionFormData) => void;
  deleteAction: (id: string, actionType: ACTION_TYPE) => void;
}

const mapFetched = (items: FetchedAction[]): Action[] => {
  return items.map((item) => {
    const parsedSpec = typeof item.spec === 'string' ? safeJsonParse(item.spec, {} as ParsedActionSpec) : item.spec;

    return {
      ...item,
      spec: {
        actionName: parsedSpec.actionName,
        notes: parsedSpec.notes,
        disabled: parsedSpec.disabled,
        signals: parsedSpec.signals.map((str) => str.toLowerCase() as SIGNAL_TYPE),
        collectContainerAttributes: parsedSpec.collectContainerAttributes || false,
        collectWorkloadId: parsedSpec.collectWorkloadUID || false,
        collectClusterId: parsedSpec.collectClusterUID || false,
        labelsAttributes: parsedSpec.labelsAttributes,
        annotationsAttributes: parsedSpec.annotationsAttributes,
        clusterAttributes: parsedSpec.clusterAttributes,
        attributeNamesToDelete: parsedSpec.attributeNamesToDelete,
        renames: parsedSpec.renames,
        piiCategories: parsedSpec.piiCategories,
        fallbackSamplingRatio: parsedSpec.fallback_sampling_ratio,
        samplingPercentage: Number(parsedSpec.sampling_percentage),
        endpointsFilters: parsedSpec.endpoints_filters?.map(({ service_name, http_route, minimum_latency_threshold, fallback_sampling_ratio }) => ({
          serviceName: service_name,
          httpRoute: http_route,
          minimumLatencyThreshold: minimum_latency_threshold,
          fallbackSamplingRatio: fallback_sampling_ratio,
        })),
      },
    };
  });
};

const mapFormToInput = (action: ActionFormData): ActionInput => {
  const {
    type,
    name = '',
    notes = '',
    disabled = false,
    signals,
    collectContainerAttributes,
    collectWorkloadId,
    collectClusterId,
    labelsAttributes,
    annotationsAttributes,
    clusterAttributes,
    attributeNamesToDelete,
    renames,
    piiCategories,
    fallbackSamplingRatio,
    samplingPercentage,
    endpointsFilters,
  } = action;

  const payload: ActionInput = {
    type,
    name,
    notes,
    disable: disabled,
    signals: signals.map((signal) => signal.toUpperCase()),
    details: '',
  };

  switch (type) {
    case ACTION_TYPE.K8S_ATTRIBUTES:
      payload['details'] = JSON.stringify({ collectContainerAttributes, collectWorkloadId, collectClusterId, labelsAttributes, annotationsAttributes });
      break;

    case ACTION_TYPE.ADD_CLUSTER_INFO:
      payload['details'] = JSON.stringify({ clusterAttributes });
      break;

    case ACTION_TYPE.DELETE_ATTRIBUTES:
      payload['details'] = JSON.stringify({ attributeNamesToDelete });
      break;

    case ACTION_TYPE.RENAME_ATTRIBUTES:
      payload['details'] = JSON.stringify({ renames });
      break;

    case ACTION_TYPE.PII_MASKING:
      payload['details'] = JSON.stringify({ piiCategories });
      break;

    case ACTION_TYPE.ERROR_SAMPLER:
      payload['details'] = JSON.stringify({ fallback_sampling_ratio: fallbackSamplingRatio });
      break;

    case ACTION_TYPE.PROBABILISTIC_SAMPLER:
      payload['details'] = JSON.stringify({ sampling_percentage: String(samplingPercentage) });
      break;

    case ACTION_TYPE.LATENCY_SAMPLER:
      payload['details'] = JSON.stringify({
        endpoints_filters:
          endpointsFilters?.map(({ serviceName, httpRoute, minimumLatencyThreshold, fallbackSamplingRatio }) => ({
            service_name: serviceName,
            http_route: httpRoute,
            minimum_latency_threshold: minimumLatencyThreshold,
            fallback_sampling_ratio: fallbackSamplingRatio,
          })) || [],
      });
      break;

    default:
      break;
  }

  return payload;
};

export const useActionCRUD = (): UseActionCrud => {
  const { data: config } = useConfig();
  const { addNotification, removeNotifications } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: ENTITY_TYPES.ACTION, target: id ? getSseTargetFromId(id, ENTITY_TYPES.ACTION) : undefined, hideFromHistory });
  };

  const {
    data,
    loading: isFetching,
    refetch: fetchActions,
  } = useQuery<{ computePlatform: { actions: FetchedAction[] } }>(GET_ACTIONS, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message),
  });

  const [createAction, cState] = useMutation<{ createAction: { id: string } }, { action: ActionInput }>(CREATE_ACTION, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.CREATE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = res?.createAction?.id;
      const type = req?.variables?.action?.type;
      notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.CREATE, `Action "${type}" created`, id);
      fetchActions();
    },
  });

  const [updateAction, uState] = useMutation<{ updateAction: { id: string } }, { id: string; action: ActionInput }>(UPDATE_ACTION, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = res?.updateAction?.id;
      const type = req?.variables?.action?.type;
      notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.UPDATE, `Action "${type}" updated`, id);
      fetchActions();
    },
  });

  const [deleteAction, dState] = useMutation<{ deleteAction: boolean }, { id: string; actionType: ACTION_TYPE }>(DELETE_ACTION, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.DELETE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id;
      const type = req?.variables?.actionType;
      removeNotifications(getSseTargetFromId(id, ENTITY_TYPES.ACTION));
      notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.DELETE, `Action "${type}" deleted`, id);
      fetchActions();
    },
  });

  return {
    actions: mapFetched(data?.computePlatform?.actions || []),
    actionsLoading: isFetching || cState.loading || uState.loading || dState.loading,
    fetchActions,

    createAction: (action) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        createAction({ variables: { action: mapFormToInput({ ...action }) } });
      }
    },
    updateAction: (id, action) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        updateAction({ variables: { id, action: mapFormToInput({ ...action }) } });
      }
    },
    deleteAction: (id, actionType) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        deleteAction({ variables: { id, actionType } });
      }
    },
  };
};
