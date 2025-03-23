import { useEffect } from 'react';
import { useConfig } from '../config';
import { GET_INSTRUMENTATION_RULES } from '@/graphql';
import { useLazyQuery, useMutation } from '@apollo/client';
import { getSseTargetFromId } from '@odigos/ui-kit/functions';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { useEntityStore, useNotificationStore } from '@odigos/ui-kit/store';
import { CREATE_INSTRUMENTATION_RULE, UPDATE_INSTRUMENTATION_RULE, DELETE_INSTRUMENTATION_RULE } from '@/graphql/mutations';
import { Crud, EntityTypes, type InstrumentationRule, type InstrumentationRuleFormData, StatusType } from '@odigos/ui-kit/types';

interface UseInstrumentationRuleCrud {
  instrumentationRules: InstrumentationRule[];
  instrumentationRulesLoading: boolean;
  fetchInstrumentationRules: () => void;
  createInstrumentationRule: (instrumentationRule: InstrumentationRuleFormData) => void;
  updateInstrumentationRule: (ruleId: string, instrumentationRule: InstrumentationRuleFormData) => void;
  deleteInstrumentationRule: (ruleId: string) => void;
}

export const useInstrumentationRuleCRUD = (): UseInstrumentationRuleCrud => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();
  const { instrumentationRulesLoading, setEntitiesLoading, instrumentationRules, addEntities, removeEntities } = useEntityStore();

  const notifyUser = (type: StatusType, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: EntityTypes.InstrumentationRule, target: id ? getSseTargetFromId(id, EntityTypes.InstrumentationRule) : undefined, hideFromHistory });
  };

  const [fetchAll] = useLazyQuery<{ computePlatform?: { instrumentationRules?: InstrumentationRule[] } }>(GET_INSTRUMENTATION_RULES, {
    fetchPolicy: 'cache-and-network',
  });

  const fetchInstrumentationRules = async () => {
    setEntitiesLoading(EntityTypes.InstrumentationRule, true);
    const { error, data } = await fetchAll();

    if (error) {
      notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message);
    } else if (data?.computePlatform?.instrumentationRules) {
      const { instrumentationRules: items } = data.computePlatform;

      addEntities(EntityTypes.InstrumentationRule, items);
      setEntitiesLoading(EntityTypes.InstrumentationRule, false);
    }
  };

  const [mutateCreate] = useMutation<{ createInstrumentationRule: InstrumentationRule }, { instrumentationRule: InstrumentationRuleFormData }>(CREATE_INSTRUMENTATION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Create, error.cause?.message || error.message),
    onCompleted: (res) => {
      const rule = res.createInstrumentationRule;
      const type = rule.type;
      addEntities(EntityTypes.InstrumentationRule, [rule]);
      notifyUser(StatusType.Success, Crud.Create, `Successfully created "${type}" rule`, rule.ruleId);
    },
  });

  const [mutateUpdate] = useMutation<{ updateInstrumentationRule: InstrumentationRule }, { ruleId: string; instrumentationRule: InstrumentationRuleFormData }>(UPDATE_INSTRUMENTATION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message),
    onCompleted: (res) => {
      const rule = res.updateInstrumentationRule;
      const type = rule.type;
      addEntities(EntityTypes.InstrumentationRule, [rule]);
      notifyUser(StatusType.Success, Crud.Update, `Successfully updated "${type}" rule`, rule.ruleId);
    },
  });

  const [mutateDelete] = useMutation<{ deleteInstrumentationRule: boolean }, { ruleId: string }>(DELETE_INSTRUMENTATION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Delete, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.ruleId as string;
      const rule = instrumentationRules.find((r) => r.ruleId === id);
      const type = rule?.type || '';
      removeEntities(EntityTypes.InstrumentationRule, [id]);
      notifyUser(StatusType.Success, Crud.Delete, `Successfully deleted "${type || id}" rule`, id);
    },
  });

  const createInstrumentationRule: UseInstrumentationRuleCrud['createInstrumentationRule'] = (instrumentationRule) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateCreate({ variables: { instrumentationRule } });
    }
  };

  const updateInstrumentationRule: UseInstrumentationRuleCrud['updateInstrumentationRule'] = (ruleId, instrumentationRule) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateUpdate({ variables: { ruleId, instrumentationRule } });
    }
  };

  const deleteInstrumentationRule: UseInstrumentationRuleCrud['deleteInstrumentationRule'] = (ruleId) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      mutateDelete({ variables: { ruleId } });
    }
  };

  useEffect(() => {
    if (!instrumentationRules.length && !instrumentationRulesLoading) fetchInstrumentationRules();
  }, []);

  return {
    instrumentationRules,
    instrumentationRulesLoading,
    fetchInstrumentationRules,
    createInstrumentationRule,
    updateInstrumentationRule,
    deleteInstrumentationRule,
  };
};
