import { useMemo } from 'react';
import { useNotificationStore } from '@/store';
import { GET_INSTRUMENTATION_RULES } from '@/graphql';
import { useMutation, useQuery } from '@apollo/client';
import { ACTION, deriveTypeFromRule, getSseTargetFromId } from '@/utils';
import { type ComputePlatform, NOTIFICATION_TYPE, OVERVIEW_ENTITY_TYPES, type InstrumentationRuleInput } from '@/types';
import { CREATE_INSTRUMENTATION_RULE, UPDATE_INSTRUMENTATION_RULE, DELETE_INSTRUMENTATION_RULE } from '@/graphql/mutations';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useInstrumentationRuleCRUD = (params?: Params) => {
  const { addNotification, removeNotifications } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({
      type,
      title,
      message,
      crdType: OVERVIEW_ENTITY_TYPES.RULE,
      target: id ? getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.RULE) : undefined,
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
  const { data, loading, refetch } = useQuery<ComputePlatform>(GET_INSTRUMENTATION_RULES, {
    onError: (error) => handleError(error.name || ACTION.FETCH, error.cause?.message || error.message),
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
    onError: (error) => handleError(ACTION.CREATE, error.message),
    onCompleted: (res, req) => {
      const id = res?.createInstrumentationRule?.ruleId;
      handleComplete(ACTION.CREATE, `Rule "${id}" created`, id);
    },
  });

  const [updateInstrumentationRule, uState] = useMutation<{ updateInstrumentationRule: { ruleId: string } }>(UPDATE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: (res, req) => {
      const id = res?.updateInstrumentationRule?.ruleId;
      handleComplete(ACTION.UPDATE, `Rule "${id}" updated`, id);
    },
  });

  const [deleteInstrumentationRule, dState] = useMutation<{ deleteInstrumentationRule: boolean }>(DELETE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError(ACTION.DELETE, error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.ruleId;
      removeNotifications(getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.RULE));
      handleComplete(ACTION.DELETE, `Rule "${id}" deleted`, id);
    },
  });

  return {
    loading: loading || cState.loading || uState.loading || dState.loading,
    instrumentationRules: mapped,
    filteredInstrumentationRules: filtered,
    refetchInstrumentationRules: refetch,

    createInstrumentationRule: (instrumentationRule: InstrumentationRuleInput) => {
      createInstrumentationRule({ variables: { instrumentationRule } });
    },
    updateInstrumentationRule: (ruleId: string, instrumentationRule: InstrumentationRuleInput) => {
      updateInstrumentationRule({ variables: { ruleId, instrumentationRule } });
    },
    deleteInstrumentationRule: (ruleId: string) => {
      deleteInstrumentationRule({ variables: { ruleId } });
    },
  };
};
