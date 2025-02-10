import { useMemo } from 'react';
import { useConfig } from '../config';
import { GET_ACTIONS } from '@/graphql';
import { useMutation, useQuery } from '@apollo/client';
import { CREATE_ACTION, DELETE_ACTION, UPDATE_ACTION } from '@/graphql/mutations';
import { type ComputePlatform, type ActionInput, type ParsedActionSpec } from '@/types';
import { ActionFormData, useFilterStore, useNotificationStore } from '@odigos/ui-containers';
import { type Action, ACTION_TYPE, CRUD, DISPLAY_TITLES, ENTITY_TYPES, FORM_ALERTS, getSseTargetFromId, NOTIFICATION_TYPE, safeJsonParse, SIGNAL_TYPE } from '@odigos/ui-utils';

interface UseActionCrudParams {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

interface UseActionCrudResponse {
  loading: boolean;
  actions: Action[];
  filteredActions: Action[];
  refetchActions: () => void;

  createAction: (action: ActionFormData) => void;
  updateAction: (id: string, action: ActionFormData) => void;
  deleteAction: (id: string, actionType: ACTION_TYPE) => void;
}

export const useActionCRUD = (params?: UseActionCrudParams): UseActionCrudResponse => {
  const filters = useFilterStore();
  const { data: config } = useConfig();
  const { addNotification, removeNotifications } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({
      type,
      title,
      message,
      crdType: ENTITY_TYPES.ACTION,
      target: id ? getSseTargetFromId(id, ENTITY_TYPES.ACTION) : undefined,
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
    onError: (error) => handleError(error.name || CRUD.READ, error.cause?.message || error.message),
  });

  // Map fetched data
  const mapped: Action[] = useMemo(() => {
    return (data?.computePlatform?.actions || []).map((item) => {
      const parsedSpec = typeof item.spec === 'string' ? safeJsonParse(item.spec, {} as ParsedActionSpec) : item.spec;

      return {
        ...item,
        spec: {
          actionName: parsedSpec.actionName,
          notes: parsedSpec.notes,
          disabled: parsedSpec.disabled,
          signals: parsedSpec.signals.map((str) => str.toLowerCase() as SIGNAL_TYPE),
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
  }, [data]);

  // Filter mapped data
  const filtered = useMemo(() => {
    let arr = [...mapped];
    if (!!filters.monitors.length) arr = arr.filter((action) => !!filters.monitors.find((metric) => action.spec.signals.find((str) => str.toLowerCase() === metric.id)));
    return arr;
  }, [mapped, filters]);

  const [createAction, cState] = useMutation<{ createAction: { id: string } }, { action: ActionInput }>(CREATE_ACTION, {
    onError: (error) => handleError(CRUD.CREATE, error.message),
    onCompleted: (res) => {
      const id = res?.createAction?.id;
      handleComplete(CRUD.CREATE, `Action "${id}" created`, id);
    },
  });

  const [updateAction, uState] = useMutation<{ updateAction: { id: string } }, { id: string; action: ActionInput }>(UPDATE_ACTION, {
    onError: (error) => handleError(CRUD.UPDATE, error.message),
    onCompleted: (res) => {
      const id = res?.updateAction?.id;
      handleComplete(CRUD.UPDATE, `Action "${id}" updated`, id);
    },
  });

  const [deleteAction, dState] = useMutation<{ deleteAction: boolean }>(DELETE_ACTION, {
    onError: (error) => handleError(CRUD.DELETE, error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.id;
      removeNotifications(getSseTargetFromId(id, ENTITY_TYPES.ACTION));
      handleComplete(CRUD.DELETE, `Action "${id}" deleted`, id);
    },
  });

  const mapFormToInput = (action: ActionFormData): ActionInput => {
    const {
      type,
      name = '',
      notes = '',
      disabled = false,
      signals,
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

  return {
    loading: loading || cState.loading || uState.loading || dState.loading,
    actions: mapped,
    filteredActions: filtered,
    refetchActions: refetch,

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
