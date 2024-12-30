import { useMutation } from '@apollo/client';
import { useNotificationStore } from '@/store';
import { ACTION, getSseTargetFromId } from '@/utils';
import { useComputePlatform } from '../compute-platform';
import { NOTIFICATION_TYPE, OVERVIEW_ENTITY_TYPES, type InstrumentationRuleInput } from '@/types';
import { CREATE_INSTRUMENTATION_RULE, UPDATE_INSTRUMENTATION_RULE, DELETE_INSTRUMENTATION_RULE } from '@/graphql/mutations';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useInstrumentationRuleCRUD = (params?: Params) => {
  const removeNotifications = useNotificationStore((store) => store.removeNotifications);
  const { data, refetch } = useComputePlatform();
  const { addNotification } = useNotificationStore();

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

  const handleError = (title: string, message: string) => {
    notifyUser(NOTIFICATION_TYPE.ERROR, title, message);
    params?.onError?.(title);
  };

  const handleComplete = (title: string) => {
    refetch();
    params?.onSuccess?.(title);
  };

  const [createInstrumentationRule, cState] = useMutation<{ createInstrumentationRule: { ruleId: string } }>(CREATE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError(ACTION.CREATE, error.message),
    onCompleted: () => handleComplete(ACTION.CREATE),
  });
  const [updateInstrumentationRule, uState] = useMutation<{ updateInstrumentationRule: { ruleId: string } }>(UPDATE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: () => handleComplete(ACTION.UPDATE),
  });
  const [deleteInstrumentationRule, dState] = useMutation<{ deleteInstrumentationRule: boolean }>(DELETE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError(ACTION.DELETE, error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.ruleId;
      removeNotifications(getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.RULE));
      handleComplete(ACTION.DELETE);
    },
  });

  return {
    loading: cState.loading || uState.loading || dState.loading,
    instrumentationRules: data?.computePlatform?.instrumentationRules || [],

    createInstrumentationRule: (instrumentationRule: InstrumentationRuleInput) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'creating instrumentation rule...', undefined, true);
      createInstrumentationRule({ variables: { instrumentationRule } });
    },
    updateInstrumentationRule: (ruleId: string, instrumentationRule: InstrumentationRuleInput) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'updating instrumentation rule...', undefined, true);
      updateInstrumentationRule({ variables: { ruleId, instrumentationRule } });
    },
    deleteInstrumentationRule: (ruleId: string) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'deleting instrumentation rule...', undefined, true);
      deleteInstrumentationRule({ variables: { ruleId } });
    },
  };
};
