import { useEffect } from 'react';
import { useConfig } from '../config';
import { usePaginatedStore } from '@/store';
import { GET_INSTRUMENTATION_RULES } from '@/graphql';
import { useLazyQuery, useMutation } from '@apollo/client';
import type { FetchedInstrumentationRule } from '@/@types';
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
  const { instrumentationRulesPaginating, setPaginating, instrumentationRules, addPaginated, removePaginated } = usePaginatedStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: ENTITY_TYPES.INSTRUMENTATION_RULE, target: id ? getSseTargetFromId(id, ENTITY_TYPES.INSTRUMENTATION_RULE) : undefined, hideFromHistory });
  };

  const [fetchAll, { loading: isFetching }] = useLazyQuery<{ computePlatform?: { instrumentationRules?: FetchedInstrumentationRule[] } }>(GET_INSTRUMENTATION_RULES, {
    fetchPolicy: 'cache-and-network',
  });

  const fetchInstrumentationRules = async () => {
    setPaginating(ENTITY_TYPES.INSTRUMENTATION_RULE, true);
    const { error, data } = await fetchAll();

    if (!!error) {
      notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message);
    } else if (!!data?.computePlatform?.instrumentationRules) {
      const { instrumentationRules: items } = data.computePlatform;

      addPaginated(ENTITY_TYPES.INSTRUMENTATION_RULE, mapFetched(items));
      setPaginating(ENTITY_TYPES.INSTRUMENTATION_RULE, false);
    }
  };

  const [createInstrumentationRule, cState] = useMutation<{ createInstrumentationRule: FetchedInstrumentationRule }, { instrumentationRule: InstrumentationRuleFormData }>(
    CREATE_INSTRUMENTATION_RULE,
    {
      onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.CREATE, error.cause?.message || error.message),
      onCompleted: (res) => {
        const rule = res.createInstrumentationRule;
        const type = deriveTypeFromRule(rule);
        addPaginated(ENTITY_TYPES.INSTRUMENTATION_RULE, mapFetched([rule]));
        notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.CREATE, `Successfully created "${type}" rule`, rule.ruleId);
      },
    },
  );

  const [updateInstrumentationRule, uState] = useMutation<{ updateInstrumentationRule: FetchedInstrumentationRule }, { ruleId: string; instrumentationRule: InstrumentationRuleFormData }>(
    UPDATE_INSTRUMENTATION_RULE,
    {
      onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message),
      onCompleted: (res) => {
        const rule = res.updateInstrumentationRule;
        const type = deriveTypeFromRule(rule);
        addPaginated(ENTITY_TYPES.INSTRUMENTATION_RULE, mapFetched([rule]));
        notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.UPDATE, `Successfully updated "${type}" rule`, rule.ruleId);
      },
    },
  );

  const [deleteInstrumentationRule, dState] = useMutation<{ deleteInstrumentationRule: boolean }, { ruleId: string }>(DELETE_INSTRUMENTATION_RULE, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.DELETE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.ruleId as string;
      const rule = instrumentationRules.find((r) => r.ruleId === id);
      const type = !!rule ? deriveTypeFromRule(rule) : '';
      removePaginated(ENTITY_TYPES.INSTRUMENTATION_RULE, [id]);
      notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.DELETE, `Successfully deleted "${type || id}" rule`, id);
    },
  });

  useEffect(() => {
    if (!instrumentationRules.length && !instrumentationRulesPaginating) fetchInstrumentationRules();
  }, []);

  return {
    instrumentationRules,
    instrumentationRulesLoading: isFetching || instrumentationRulesPaginating || cState.loading || uState.loading || dState.loading,
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
