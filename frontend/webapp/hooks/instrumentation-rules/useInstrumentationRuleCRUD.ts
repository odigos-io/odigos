import { useEffect } from 'react';
import { useConfig } from '../config';
import { GET_INSTRUMENTATION_RULES } from '@/graphql';
import { useLazyQuery, useMutation } from '@apollo/client';
import type { FetchedInstrumentationRule } from '@/types';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { useEntityStore, useNotificationStore } from '@odigos/ui-kit/store';
import { deriveTypeFromRule, getSseTargetFromId } from '@odigos/ui-kit/functions';
import { CREATE_INSTRUMENTATION_RULE, UPDATE_INSTRUMENTATION_RULE, DELETE_INSTRUMENTATION_RULE } from '@/graphql/mutations';
import { CRUD, ENTITY_TYPES, type InstrumentationRule, type InstrumentationRuleFormData, STATUS_TYPE } from '@odigos/ui-kit/types';

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
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();
  const { instrumentationRulesLoading, setEntitiesLoading, instrumentationRules, addEntities, removeEntities } = useEntityStore();

  const notifyUser = (type: STATUS_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: ENTITY_TYPES.INSTRUMENTATION_RULE, target: id ? getSseTargetFromId(id, ENTITY_TYPES.INSTRUMENTATION_RULE) : undefined, hideFromHistory });
  };

  const [fetchAll, { loading: isFetching }] = useLazyQuery<{ computePlatform?: { instrumentationRules?: FetchedInstrumentationRule[] } }>(GET_INSTRUMENTATION_RULES, {
    fetchPolicy: 'cache-and-network',
  });

  const fetchInstrumentationRules = async () => {
    setEntitiesLoading(ENTITY_TYPES.INSTRUMENTATION_RULE, true);
    const { error, data } = await fetchAll();

    if (error) {
      notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message);
    } else if (data?.computePlatform?.instrumentationRules) {
      const { instrumentationRules: items } = data.computePlatform;

      addEntities(ENTITY_TYPES.INSTRUMENTATION_RULE, mapFetched(items));
      setEntitiesLoading(ENTITY_TYPES.INSTRUMENTATION_RULE, false);
    }
  };

  const [mutateCreate, cState] = useMutation<{ createInstrumentationRule: FetchedInstrumentationRule }, { instrumentationRule: InstrumentationRuleFormData }>(CREATE_INSTRUMENTATION_RULE, {
    onError: (error) => notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.CREATE, error.cause?.message || error.message),
    onCompleted: (res) => {
      const rule = res.createInstrumentationRule;
      const type = deriveTypeFromRule(rule);
      addEntities(ENTITY_TYPES.INSTRUMENTATION_RULE, mapFetched([rule]));
      notifyUser(STATUS_TYPE.SUCCESS, CRUD.CREATE, `Successfully created "${type}" rule`, rule.ruleId);
    },
  });

  const [mutateUpdate, uState] = useMutation<{ updateInstrumentationRule: FetchedInstrumentationRule }, { ruleId: string; instrumentationRule: InstrumentationRuleFormData }>(
    UPDATE_INSTRUMENTATION_RULE,
    {
      onError: (error) => notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message),
      onCompleted: (res) => {
        const rule = res.updateInstrumentationRule;
        const type = deriveTypeFromRule(rule);
        addEntities(ENTITY_TYPES.INSTRUMENTATION_RULE, mapFetched([rule]));
        notifyUser(STATUS_TYPE.SUCCESS, CRUD.UPDATE, `Successfully updated "${type}" rule`, rule.ruleId);
      },
    },
  );

  const [mutateDelete, dState] = useMutation<{ deleteInstrumentationRule: boolean }, { ruleId: string }>(DELETE_INSTRUMENTATION_RULE, {
    onError: (error) => notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.DELETE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.ruleId as string;
      const rule = instrumentationRules.find((r) => r.ruleId === id);
      const type = rule ? deriveTypeFromRule(rule) : '';
      removeEntities(ENTITY_TYPES.INSTRUMENTATION_RULE, [id]);
      notifyUser(STATUS_TYPE.SUCCESS, CRUD.DELETE, `Successfully deleted "${type || id}" rule`, id);
    },
  });

  const createInstrumentationRule: UseInstrumentationRuleCrud['createInstrumentationRule'] = (instrumentationRule) => {
    if (isReadonly) {
      notifyUser(STATUS_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateCreate({ variables: { instrumentationRule } });
    }
  };

  const updateInstrumentationRule: UseInstrumentationRuleCrud['updateInstrumentationRule'] = (ruleId, instrumentationRule) => {
    if (isReadonly) {
      notifyUser(STATUS_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateUpdate({ variables: { ruleId, instrumentationRule } });
    }
  };

  const deleteInstrumentationRule: UseInstrumentationRuleCrud['deleteInstrumentationRule'] = (ruleId) => {
    if (isReadonly) {
      notifyUser(STATUS_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateDelete({ variables: { ruleId } });
    }
  };

  useEffect(() => {
    if (!instrumentationRules.length && !instrumentationRulesLoading) fetchInstrumentationRules();
  }, []);

  return {
    instrumentationRules,
    instrumentationRulesLoading: isFetching || instrumentationRulesLoading || cState.loading || uState.loading || dState.loading,
    fetchInstrumentationRules,
    createInstrumentationRule,
    updateInstrumentationRule,
    deleteInstrumentationRule,
  };
};
