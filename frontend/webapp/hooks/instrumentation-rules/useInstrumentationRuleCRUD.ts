import { useMutation } from '@apollo/client';
import { useNotificationStore } from '@/store';
import { useNotify } from '../notification/useNotify';
import { useComputePlatform } from '../compute-platform';
import { ACTION, deriveTypeFromRule, getSseTargetFromId, NOTIFICATION } from '@/utils';
import { OVERVIEW_ENTITY_TYPES, type InstrumentationRuleInput, type NotificationType } from '@/types';
import { CREATE_INSTRUMENTATION_RULE, UPDATE_INSTRUMENTATION_RULE, DELETE_INSTRUMENTATION_RULE } from '@/graphql/mutations';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useInstrumentationRuleCRUD = (params?: Params) => {
  const removeNotifications = useNotificationStore((store) => store.removeNotifications);
  const { data, refetch } = useComputePlatform();
  const notify = useNotify();

  const notifyUser = (type: NotificationType, title: string, message: string, id?: string) => {
    notify({
      type,
      title,
      message,
      crdType: OVERVIEW_ENTITY_TYPES.RULE,
      target: id ? getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.RULE) : undefined,
    });
  };

  const handleError = (title: string, message: string, id?: string) => {
    notifyUser(NOTIFICATION.ERROR, title, message, id);
    params?.onError?.(title);
  };

  const handleComplete = (title: string, message: string, id?: string) => {
    notifyUser(NOTIFICATION.SUCCESS, title, message, id);
    refetch();
    params?.onSuccess?.(title);
  };

  const [createInstrumentationRule, cState] = useMutation<{
    createInstrumentationRule: { ruleId: string };
  }>(CREATE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError(ACTION.CREATE, error.message),
    onCompleted: (res, req) => {
      const id = res.createInstrumentationRule.ruleId;
      const type = deriveTypeFromRule(req?.variables?.instrumentationRule);
      const name = req?.variables?.instrumentationRule.ruleName;
      const label = `${type}${!!name ? ` (${name})` : ''}`;
      handleComplete(ACTION.CREATE, `instrumentation rule "${label}" was created`, id);
    },
  });
  const [updateInstrumentationRule, uState] = useMutation<{
    updateInstrumentationRule: { ruleId: string };
  }>(UPDATE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: (res, req) => {
      const id = res.updateInstrumentationRule.ruleId;
      const type = deriveTypeFromRule(req?.variables?.instrumentationRule);
      const name = req?.variables?.instrumentationRule.ruleName;
      const label = `${type}${!!name ? ` (${name})` : ''}`;
      handleComplete(ACTION.UPDATE, `instrumentation rule "${label}" was updated`, id);
    },
  });
  const [deleteInstrumentationRule, dState] = useMutation<{
    deleteInstrumentationRule: boolean;
  }>(DELETE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError(ACTION.DELETE, error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.ruleId;
      removeNotifications(getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.RULE));
      handleComplete(ACTION.DELETE, `instrumentation rule "${id}" was deleted`);
    },
  });

  return {
    loading: cState.loading || uState.loading || dState.loading,
    instrumentationRules:
      data?.computePlatform?.instrumentationRules?.map((item) => {
        const type = deriveTypeFromRule(item);

        return { ...item, type };
      }) || [],

    createInstrumentationRule: (instrumentationRule: InstrumentationRuleInput) => createInstrumentationRule({ variables: { instrumentationRule } }),
    updateInstrumentationRule: (ruleId: string, instrumentationRule: InstrumentationRuleInput) => updateInstrumentationRule({ variables: { ruleId, instrumentationRule } }),
    deleteInstrumentationRule: (ruleId: string) => deleteInstrumentationRule({ variables: { ruleId } }),
  };
};
