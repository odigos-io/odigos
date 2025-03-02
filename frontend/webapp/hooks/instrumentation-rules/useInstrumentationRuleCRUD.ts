import { useMemo } from 'react';
import { useConfig } from '../config';
import { GET_INSTRUMENTATION_RULES } from '@/graphql';
import { useMutation, useQuery } from '@apollo/client';
import type { FetchedInstrumentationRule, ComputePlatform } from '@/@types';
import { type InstrumentationRuleFormData, useNotificationStore } from '@odigos/ui-containers';
import { CREATE_INSTRUMENTATION_RULE, UPDATE_INSTRUMENTATION_RULE, DELETE_INSTRUMENTATION_RULE } from '@/graphql/mutations';
import { CRUD, deriveTypeFromRule, DISPLAY_TITLES, ENTITY_TYPES, FORM_ALERTS, getSseTargetFromId, InstrumentationRule, NOTIFICATION_TYPE } from '@odigos/ui-utils';

interface UseInstrumentationRuleCrud {
  instrumentationRules: InstrumentationRule[];
  instrumentationRulesLoading: boolean;
  fetchInstrumentationRules: () => void;
  createInstrumentationRule: (instrumentationRule: InstrumentationRuleFormData) => void;
  updateInstrumentationRule: (ruleId: string, instrumentationRule: InstrumentationRuleFormData) => void;
  deleteInstrumentationRule: (ruleId: string) => void;
}

const mapFetched = (items: FetchedInstrumentationRule[]): InstrumentationRule[] => {
  return items.map((item) => {
    const type = deriveTypeFromRule(item);

    return { ...item, type };
  });
};

export const useInstrumentationRuleCRUD = (): UseInstrumentationRuleCrud => {
  const { data: config } = useConfig();
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: ENTITY_TYPES.INSTRUMENTATION_RULE, target: id ? getSseTargetFromId(id, ENTITY_TYPES.INSTRUMENTATION_RULE) : undefined, hideFromHistory });
  };

  const {
    data,
    loading: isFetching,
    refetch: fetchInstrumentationRules,
  } = useQuery<ComputePlatform>(GET_INSTRUMENTATION_RULES, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message),
  });

  const [createInstrumentationRule, cState] = useMutation<{ createInstrumentationRule: { ruleId: string } }, { instrumentationRule: InstrumentationRuleFormData }>(CREATE_INSTRUMENTATION_RULE, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.CREATE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = res?.createInstrumentationRule?.ruleId;
      const type = deriveTypeFromRule(req?.variables?.instrumentationRule);
      notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.CREATE, `Rule "${type}" created`, id);
      fetchInstrumentationRules();
    },
  });

  const [updateInstrumentationRule, uState] = useMutation<{ updateInstrumentationRule: { ruleId: string } }, { ruleId: string; instrumentationRule: InstrumentationRuleFormData }>(
    UPDATE_INSTRUMENTATION_RULE,
    {
      onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message),
      onCompleted: (res, req) => {
        const id = res?.updateInstrumentationRule?.ruleId;
        const type = deriveTypeFromRule(req?.variables?.instrumentationRule);
        notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.UPDATE, `Rule "${type}" updated`, id);
        fetchInstrumentationRules();
      },
    },
  );

  const [deleteInstrumentationRule, dState] = useMutation<{ deleteInstrumentationRule: boolean }, { ruleId: string }>(DELETE_INSTRUMENTATION_RULE, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.DELETE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.ruleId;
      // TODO: find a way to derive the type, instead of ID in toast
      notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.DELETE, `Rule "${id}" deleted`, id);
      fetchInstrumentationRules();
    },
  });

  const mapped = useMemo(() => mapFetched(data?.computePlatform?.instrumentationRules || []), [data?.computePlatform?.instrumentationRules]);

  return {
    instrumentationRules: mapped,
    instrumentationRulesLoading: isFetching || cState.loading || uState.loading || dState.loading,
    fetchInstrumentationRules,

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
