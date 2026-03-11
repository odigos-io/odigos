import { useState, useEffect, useCallback } from 'react';
import { useLazyQuery, useMutation } from '@apollo/client';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { Crud, StatusType } from '@odigos/ui-kit/types';
import { GET_SAMPLING_RULES } from '@/graphql';
import {
  CREATE_NOISY_OPERATION_RULE,
  UPDATE_NOISY_OPERATION_RULE,
  DELETE_NOISY_OPERATION_RULE,
  CREATE_HIGHLY_RELEVANT_OPERATION_RULE,
  UPDATE_HIGHLY_RELEVANT_OPERATION_RULE,
  DELETE_HIGHLY_RELEVANT_OPERATION_RULE,
  CREATE_COST_REDUCTION_RULE,
  UPDATE_COST_REDUCTION_RULE,
  DELETE_COST_REDUCTION_RULE,
} from '@/graphql/mutations';
import type {
  SamplingRules,
  NoisyOperationRule,
  NoisyOperationRuleInput,
  HighlyRelevantOperationRule,
  HighlyRelevantOperationRuleInput,
  CostReductionRule,
  CostReductionRuleInput,
} from '@/types';

interface UseSamplingRuleCrud {
  samplingRules: SamplingRules;
  loading: boolean;
  fetchSamplingRules: () => void;

  createNoisyOperationRule: (rule: NoisyOperationRuleInput) => void;
  updateNoisyOperationRule: (ruleId: string, rule: NoisyOperationRuleInput) => void;
  deleteNoisyOperationRule: (ruleId: string) => void;

  createHighlyRelevantOperationRule: (rule: HighlyRelevantOperationRuleInput) => void;
  updateHighlyRelevantOperationRule: (ruleId: string, rule: HighlyRelevantOperationRuleInput) => void;
  deleteHighlyRelevantOperationRule: (ruleId: string) => void;

  createCostReductionRule: (rule: CostReductionRuleInput) => void;
  updateCostReductionRule: (ruleId: string, rule: CostReductionRuleInput) => void;
  deleteCostReductionRule: (ruleId: string) => void;
}

const EMPTY_RULES: SamplingRules = {
  noisyOperations: [],
  highlyRelevantOperations: [],
  costReductionRules: [],
};

export const useSamplingRuleCRUD = (): UseSamplingRuleCrud => {
  const { addNotification } = useNotificationStore();
  const [samplingRules, setSamplingRules] = useState<SamplingRules>(EMPTY_RULES);
  const [loading, setLoading] = useState(false);

  const notifyUser = useCallback(
    (type: StatusType, title: string, message: string) => {
      addNotification({ type, title, message });
    },
    [addNotification],
  );

  // ---- Fetch ----

  const [fetchAll] = useLazyQuery<{ sampling: { rules: SamplingRules } }>(GET_SAMPLING_RULES, { fetchPolicy: 'network-only' });

  const fetchSamplingRules = useCallback(async () => {
    setLoading(true);
    const { error, data } = await fetchAll();

    if (error) {
      notifyUser(StatusType.Error, Crud.Read, error.cause?.message || error.message);
    } else if (data?.sampling?.rules) {
      setSamplingRules(data.sampling.rules);
    }
    setLoading(false);
  }, [fetchAll, notifyUser]);

  // ---- Noisy Operations ----

  const [mutateCreateNoisy] = useMutation<{ createNoisyOperationRule: NoisyOperationRule }, { rule: NoisyOperationRuleInput }>(CREATE_NOISY_OPERATION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, Crud.Create, error.cause?.message || error.message),
    onCompleted: (res) => {
      const rule = res.createNoisyOperationRule;
      setSamplingRules((prev) => ({ ...prev, noisyOperations: [...prev.noisyOperations, rule] }));
      notifyUser(StatusType.Success, Crud.Create, `Successfully created noisy operation rule "${rule.name || rule.ruleId}"`);
    },
  });

  const [mutateUpdateNoisy] = useMutation<{ updateNoisyOperationRule: NoisyOperationRule }, { ruleId: string; rule: NoisyOperationRuleInput }>(UPDATE_NOISY_OPERATION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, Crud.Update, error.cause?.message || error.message),
    onCompleted: (res) => {
      const updated = res.updateNoisyOperationRule;
      setSamplingRules((prev) => ({
        ...prev,
        noisyOperations: prev.noisyOperations.map((r) => (r.ruleId === updated.ruleId ? updated : r)),
      }));
      notifyUser(StatusType.Success, Crud.Update, `Successfully updated noisy operation rule "${updated.name || updated.ruleId}"`);
    },
  });

  const [mutateDeleteNoisy] = useMutation<{ deleteNoisyOperationRule: boolean }, { ruleId: string }>(DELETE_NOISY_OPERATION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, Crud.Delete, error.cause?.message || error.message),
    onCompleted: (_res, req) => {
      const id = req?.variables?.ruleId as string;
      setSamplingRules((prev) => ({
        ...prev,
        noisyOperations: prev.noisyOperations.filter((r) => r.ruleId !== id),
      }));
      notifyUser(StatusType.Success, Crud.Delete, `Successfully deleted noisy operation rule`);
    },
  });

  // ---- Highly Relevant Operations ----

  const [mutateCreateHighlyRelevant] = useMutation<{ createHighlyRelevantOperationRule: HighlyRelevantOperationRule }, { rule: HighlyRelevantOperationRuleInput }>(
    CREATE_HIGHLY_RELEVANT_OPERATION_RULE,
    {
      onError: (error) => notifyUser(StatusType.Error, Crud.Create, error.cause?.message || error.message),
      onCompleted: (res) => {
        const rule = res.createHighlyRelevantOperationRule;
        setSamplingRules((prev) => ({ ...prev, highlyRelevantOperations: [...prev.highlyRelevantOperations, rule] }));
        notifyUser(StatusType.Success, Crud.Create, `Successfully created highly relevant operation rule "${rule.name || rule.ruleId}"`);
      },
    },
  );

  const [mutateUpdateHighlyRelevant] = useMutation<{ updateHighlyRelevantOperationRule: HighlyRelevantOperationRule }, { ruleId: string; rule: HighlyRelevantOperationRuleInput }>(
    UPDATE_HIGHLY_RELEVANT_OPERATION_RULE,
    {
      onError: (error) => notifyUser(StatusType.Error, Crud.Update, error.cause?.message || error.message),
      onCompleted: (res) => {
        const updated = res.updateHighlyRelevantOperationRule;
        setSamplingRules((prev) => ({
          ...prev,
          highlyRelevantOperations: prev.highlyRelevantOperations.map((r) => (r.ruleId === updated.ruleId ? updated : r)),
        }));
        notifyUser(StatusType.Success, Crud.Update, `Successfully updated highly relevant operation rule "${updated.name || updated.ruleId}"`);
      },
    },
  );

  const [mutateDeleteHighlyRelevant] = useMutation<{ deleteHighlyRelevantOperationRule: boolean }, { ruleId: string }>(DELETE_HIGHLY_RELEVANT_OPERATION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, Crud.Delete, error.cause?.message || error.message),
    onCompleted: (_res, req) => {
      const id = req?.variables?.ruleId as string;
      setSamplingRules((prev) => ({
        ...prev,
        highlyRelevantOperations: prev.highlyRelevantOperations.filter((r) => r.ruleId !== id),
      }));
      notifyUser(StatusType.Success, Crud.Delete, `Successfully deleted highly relevant operation rule`);
    },
  });

  // ---- Cost Reduction Rules ----

  const [mutateCreateCostReduction] = useMutation<{ createCostReductionRule: CostReductionRule }, { rule: CostReductionRuleInput }>(CREATE_COST_REDUCTION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, Crud.Create, error.cause?.message || error.message),
    onCompleted: (res) => {
      const rule = res.createCostReductionRule;
      setSamplingRules((prev) => ({ ...prev, costReductionRules: [...prev.costReductionRules, rule] }));
      notifyUser(StatusType.Success, Crud.Create, `Successfully created cost reduction rule "${rule.name || rule.ruleId}"`);
    },
  });

  const [mutateUpdateCostReduction] = useMutation<{ updateCostReductionRule: CostReductionRule }, { ruleId: string; rule: CostReductionRuleInput }>(UPDATE_COST_REDUCTION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, Crud.Update, error.cause?.message || error.message),
    onCompleted: (res) => {
      const updated = res.updateCostReductionRule;
      setSamplingRules((prev) => ({
        ...prev,
        costReductionRules: prev.costReductionRules.map((r) => (r.ruleId === updated.ruleId ? updated : r)),
      }));
      notifyUser(StatusType.Success, Crud.Update, `Successfully updated cost reduction rule "${updated.name || updated.ruleId}"`);
    },
  });

  const [mutateDeleteCostReduction] = useMutation<{ deleteCostReductionRule: boolean }, { ruleId: string }>(DELETE_COST_REDUCTION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, Crud.Delete, error.cause?.message || error.message),
    onCompleted: (_res, req) => {
      const id = req?.variables?.ruleId as string;
      setSamplingRules((prev) => ({
        ...prev,
        costReductionRules: prev.costReductionRules.filter((r) => r.ruleId !== id),
      }));
      notifyUser(StatusType.Success, Crud.Delete, `Successfully deleted cost reduction rule`);
    },
  });

  // ---- Public API ----

  const createNoisyOperationRule: UseSamplingRuleCrud['createNoisyOperationRule'] = (rule) => {
    mutateCreateNoisy({ variables: { rule } });
  };

  const updateNoisyOperationRule: UseSamplingRuleCrud['updateNoisyOperationRule'] = (ruleId, rule) => {
    mutateUpdateNoisy({ variables: { ruleId, rule } });
  };

  const deleteNoisyOperationRule: UseSamplingRuleCrud['deleteNoisyOperationRule'] = (ruleId) => {
    mutateDeleteNoisy({ variables: { ruleId } });
  };

  const createHighlyRelevantOperationRule: UseSamplingRuleCrud['createHighlyRelevantOperationRule'] = (rule) => {
    mutateCreateHighlyRelevant({ variables: { rule } });
  };

  const updateHighlyRelevantOperationRule: UseSamplingRuleCrud['updateHighlyRelevantOperationRule'] = (ruleId, rule) => {
    mutateUpdateHighlyRelevant({ variables: { ruleId, rule } });
  };

  const deleteHighlyRelevantOperationRule: UseSamplingRuleCrud['deleteHighlyRelevantOperationRule'] = (ruleId) => {
    mutateDeleteHighlyRelevant({ variables: { ruleId } });
  };

  const createCostReductionRule: UseSamplingRuleCrud['createCostReductionRule'] = (rule) => {
    mutateCreateCostReduction({ variables: { rule } });
  };

  const updateCostReductionRule: UseSamplingRuleCrud['updateCostReductionRule'] = (ruleId, rule) => {
    mutateUpdateCostReduction({ variables: { ruleId, rule } });
  };

  const deleteCostReductionRule: UseSamplingRuleCrud['deleteCostReductionRule'] = (ruleId) => {
    mutateDeleteCostReduction({ variables: { ruleId } });
  };

  useEffect(() => {
    fetchSamplingRules();
  }, []);

  return {
    samplingRules,
    loading,
    fetchSamplingRules,
    createNoisyOperationRule,
    updateNoisyOperationRule,
    deleteNoisyOperationRule,
    createHighlyRelevantOperationRule,
    updateHighlyRelevantOperationRule,
    deleteHighlyRelevantOperationRule,
    createCostReductionRule,
    updateCostReductionRule,
    deleteCostReductionRule,
  };
};
