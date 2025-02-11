import { useMemo } from 'react';
import { useConfig } from '../config';
import { type ComputePlatform } from '@/@types';
import { GET_INSTRUMENTATION_RULES } from '@/graphql';
import { useMutation, useQuery } from '@apollo/client';
import { type InstrumentationRuleFormData, useNotificationStore } from '@odigos/ui-containers';
import { CREATE_INSTRUMENTATION_RULE, UPDATE_INSTRUMENTATION_RULE, DELETE_INSTRUMENTATION_RULE } from '@/graphql/mutations';
import { CRUD, deriveTypeFromRule, DISPLAY_TITLES, ENTITY_TYPES, FORM_ALERTS, getSseTargetFromId, NOTIFICATION_TYPE } from '@odigos/ui-utils';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useInstrumentationRuleCRUD = (params?: Params) => {
  const { data: config } = useConfig();
  const { addNotification, removeNotifications } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({
      type,
      title,
      message,
      crdType: ENTITY_TYPES.INSTRUMENTATION_RULE,
      target: id ? getSseTargetFromId(id, ENTITY_TYPES.INSTRUMENTATION_RULE) : undefined,
      hideFromHistory,
    });
  };

  const handleError = (actionType: string, message: string) => {
    notifyUser(NOTIFICATION_TYPE.ERROR, actionType, message);
    params?.onError?.(actionType);
  };

  const handleComplete = (actionType: string, message: string, id?: string) => {
    notifyUser(NOTIFICATION_TYPE.SUCCESS, actionType, message, id);
    // refetch();
    params?.onSuccess?.(actionType);
  };

  // Fetch data
  const { data, loading, refetch } = useQuery<ComputePlatform>(GET_INSTRUMENTATION_RULES, {
    onError: (error) => handleError(error.name || CRUD.READ, error.cause?.message || error.message),
  });

  // Map fetched data
  const mapped = useMemo(() => {
    return (data?.computePlatform?.instrumentationRules || []).map((item) => {
      const type = deriveTypeFromRule(item);
      return { ...item, type };
    });
  }, [data]);

  // Filter mapped data
  const filtered = mapped; // no filters for rules yet, TBA in future

  const [createInstrumentationRule, cState] = useMutation<{ createInstrumentationRule: { ruleId: string } }>(CREATE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError(CRUD.CREATE, error.message),
    onCompleted: (res, req) => {
      const id = res?.createInstrumentationRule?.ruleId;
      handleComplete(CRUD.CREATE, `Rule "${id}" created`, id);
    },
  });

  const [updateInstrumentationRule, uState] = useMutation<{ updateInstrumentationRule: { ruleId: string } }>(UPDATE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError(CRUD.UPDATE, error.message),
    onCompleted: (res, req) => {
      const id = res?.updateInstrumentationRule?.ruleId;
      handleComplete(CRUD.UPDATE, `Rule "${id}" updated`, id);
    },
  });

  const [deleteInstrumentationRule, dState] = useMutation<{ deleteInstrumentationRule: boolean }>(DELETE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError(CRUD.DELETE, error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.ruleId;
      removeNotifications(getSseTargetFromId(id, ENTITY_TYPES.INSTRUMENTATION_RULE));
      handleComplete(CRUD.DELETE, `Rule "${id}" deleted`, id);
    },
  });

  return {
    loading: loading || cState.loading || uState.loading || dState.loading,
    instrumentationRules: mapped,
    filteredInstrumentationRules: filtered,
    refetchInstrumentationRules: refetch,

    createInstrumentationRule: (instrumentationRule: InstrumentationRuleFormData) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        createInstrumentationRule({ variables: { instrumentationRule } });
      }
    },
    updateInstrumentationRule: (ruleId: string, instrumentationRule: InstrumentationRuleFormData) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        updateInstrumentationRule({ variables: { ruleId, instrumentationRule } });
      }
    },
    deleteInstrumentationRule: (ruleId: string) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        deleteInstrumentationRule({ variables: { ruleId } });
      }
    },
  };
};
