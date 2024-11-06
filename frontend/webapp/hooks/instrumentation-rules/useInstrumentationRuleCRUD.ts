import { useDrawerStore } from '@/store';
import { useNotify } from '../useNotify';
import { useMutation } from '@apollo/client';
import { deriveTypeFromRule } from '@/utils';
import { useComputePlatform } from '../compute-platform';
import type { InstrumentationRuleInput, NotificationType } from '@/types';
import { CREATE_INSTRUMENTATION_RULE, UPDATE_INSTRUMENTATION_RULE, DELETE_INSTRUMENTATION_RULE } from '@/graphql/mutations';

interface Params {
  onSuccess?: () => void;
  onError?: () => void;
}

export const useInstrumentationRuleCRUD = (params?: Params) => {
  const { setSelectedItem: setDrawerItem } = useDrawerStore((store) => store);
  const { refetch } = useComputePlatform();
  const notify = useNotify();

  const notifyUser = (type: NotificationType, title: string, message: string) => {
    notify({ type, title, message });
  };

  const handleError = (title: string, message: string) => {
    notifyUser('error', title, message);
    params?.onError?.();
  };

  const handleComplete = (title: string, message: string) => {
    notifyUser('success', title, message);
    setDrawerItem(null);
    refetch();
    params?.onSuccess?.();
  };

  const [createInstrumentationRule, cState] = useMutation<{ createInstrumentationRule: { ruleId: string } }>(CREATE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError('Create', error.message),
    onCompleted: (_, req) => {
      const name = req?.variables?.instrumentationRule.ruleName || deriveTypeFromRule(req?.variables?.instrumentationRule);
      handleComplete('Create', `instrumentation rule "${name}" was created`);
    },
  });
  const [updateInstrumentationRule, uState] = useMutation<{ updateInstrumentationRule: { ruleId: string } }>(UPDATE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError('Update', error.message),
    onCompleted: (_, req) => {
      const name = req?.variables?.instrumentationRule.ruleName || deriveTypeFromRule(req?.variables?.instrumentationRule);
      handleComplete('Update', `instrumentation rule "${name}" was updated`);
    },
  });
  const [deleteInstrumentationRule, dState] = useMutation<{ deleteInstrumentationRule: boolean }>(DELETE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError('Delete', error.message),
    onCompleted: (_, req) => {
      const name = req?.variables?.ruleId;
      handleComplete('Delete', `instrumentation rule "${name}" was deleted`);
    },
  });

  return {
    loading: cState.loading || uState.loading || dState.loading,
    createInstrumentationRule: (instrumentationRule: InstrumentationRuleInput) => createInstrumentationRule({ variables: { instrumentationRule } }),
    updateInstrumentationRule: (ruleId: string, instrumentationRule: InstrumentationRuleInput) =>
      updateInstrumentationRule({ variables: { ruleId, instrumentationRule } }),
    deleteInstrumentationRule: (ruleId: string) => deleteInstrumentationRule({ variables: { ruleId } }),
  };
};
