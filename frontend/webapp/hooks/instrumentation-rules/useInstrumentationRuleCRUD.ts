import { useDrawerStore } from '@/store';
import { useNotify } from '../useNotify';
import { useMutation } from '@apollo/client';
import type { InstrumentationRuleInput } from '@/types';
import { useComputePlatform } from '../compute-platform';
import { CREATE_INSTRUMENTATION_RULE, UPDATE_INSTRUMENTATION_RULE, DELETE_INSTRUMENTATION_RULE } from '@/graphql/mutations';

interface Params {
  onSuccess?: () => void;
  onError?: () => void;
}

export const useInstrumentationRuleCRUD = (params?: Params) => {
  const { setSelectedItem: setDrawerItem } = useDrawerStore((store) => store);
  const { refetch } = useComputePlatform();
  const notify = useNotify();

  const notifyUser = (title: string, message: string, type: 'error' | 'success') => {
    notify({ title, message, type, target: 'notification', crdType: 'notification' });
  };

  const handleError = (title: string, message: string) => {
    notifyUser(title, message, 'error');
    params?.onError?.();
  };

  const handleComplete = (title: string, message: string) => {
    notifyUser(title, message, 'success');
    setDrawerItem(null);
    refetch();
    params?.onSuccess?.();
  };

  const [createInstrumentationRule, cState] = useMutation<{ createInstrumentationRule: { ruleId: string } }>(CREATE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError('Create Rule', error.message),
    onCompleted: () => handleComplete('Create Rule', 'successfully created'),
  });
  const [updateInstrumentationRule, uState] = useMutation<{ updateInstrumentationRule: { ruleId: string } }>(UPDATE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError('Update Rule', error.message),
    onCompleted: () => handleComplete('Update Rule', 'successfully updated'),
  });
  const [deleteInstrumentationRule, dState] = useMutation<{ deleteInstrumentationRule: boolean }>(DELETE_INSTRUMENTATION_RULE, {
    onError: (error) => handleError('Delete Rule', error.message),
    onCompleted: () => handleComplete('Delete Rule', 'successfully deleted'),
  });

  return {
    loading: cState.loading || uState.loading || dState.loading,
    createInstrumentationRule: (instrumentationRule: InstrumentationRuleInput) => createInstrumentationRule({ variables: { instrumentationRule } }),
    updateInstrumentationRule: (ruleId: string, instrumentationRule: InstrumentationRuleInput) =>
      updateInstrumentationRule({ variables: { ruleId, instrumentationRule } }),
    deleteInstrumentationRule: (ruleId: string) => deleteInstrumentationRule({ variables: { ruleId } }),
  };
};
