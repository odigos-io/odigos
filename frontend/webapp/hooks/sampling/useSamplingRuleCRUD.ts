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
  samplingRules: SamplingRules[];
  loading: boolean;
  fetchSamplingRules: () => void;

  createNoisyOperationRule: (samplingId: string, rule: NoisyOperationRuleInput) => void;
  updateNoisyOperationRule: (samplingId: string, ruleId: string, rule: NoisyOperationRuleInput) => void;
  deleteNoisyOperationRule: (samplingId: string, ruleId: string) => void;

  createHighlyRelevantOperationRule: (samplingId: string, rule: HighlyRelevantOperationRuleInput) => void;
  updateHighlyRelevantOperationRule: (samplingId: string, ruleId: string, rule: HighlyRelevantOperationRuleInput) => void;
  deleteHighlyRelevantOperationRule: (samplingId: string, ruleId: string) => void;

  createCostReductionRule: (samplingId: string, rule: CostReductionRuleInput) => void;
  updateCostReductionRule: (samplingId: string, ruleId: string, rule: CostReductionRuleInput) => void;
  deleteCostReductionRule: (samplingId: string, ruleId: string) => void;
}

function updateGroup(groups: SamplingRules[], samplingId: string, updater: (group: SamplingRules) => SamplingRules): SamplingRules[] {
  return groups.map((g) => (g.id === samplingId ? updater(g) : g));
}

export const useSamplingRuleCRUD = (): UseSamplingRuleCrud => {
  const { addNotification } = useNotificationStore();
  const [samplingRules, setSamplingRules] = useState<SamplingRules[]>([]);
  const [loading, setLoading] = useState(false);

  const notifyUser = useCallback(
    (type: StatusType, title: string, message: string) => {
      addNotification({ type, title, message });
    },
    [addNotification],
  );

  // ---- Fetch ----

  const [fetchAll] = useLazyQuery<{ sampling: { rules: SamplingRules[] } }>(GET_SAMPLING_RULES, { fetchPolicy: 'network-only' });

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

  const [mutateCreateNoisy] = useMutation<{ createNoisyOperationRule: NoisyOperationRule }, { samplingId: string; rule: NoisyOperationRuleInput }>(CREATE_NOISY_OPERATION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, Crud.Create, error.cause?.message || error.message),
    onCompleted: (res, opts) => {
      const rule = res.createNoisyOperationRule;
      const sid = opts?.variables?.samplingId as string;
      setSamplingRules((prev) => updateGroup(prev, sid, (g) => ({ ...g, noisyOperations: [...g.noisyOperations, rule] })));
      notifyUser(StatusType.Success, Crud.Create, `Successfully created noisy operation rule "${rule.name || rule.ruleId}"`);
    },
  });

  const [mutateUpdateNoisy] = useMutation<{ updateNoisyOperationRule: NoisyOperationRule }, { samplingId: string; ruleId: string; rule: NoisyOperationRuleInput }>(
    UPDATE_NOISY_OPERATION_RULE,
    {
      onError: (error) => notifyUser(StatusType.Error, Crud.Update, error.cause?.message || error.message),
      onCompleted: (res, opts) => {
        const updated = res.updateNoisyOperationRule;
        const sid = opts?.variables?.samplingId as string;
        setSamplingRules((prev) =>
          updateGroup(prev, sid, (g) => ({
            ...g,
            noisyOperations: g.noisyOperations.map((r) => (r.ruleId === updated.ruleId ? updated : r)),
          })),
        );
        notifyUser(StatusType.Success, Crud.Update, `Successfully updated noisy operation rule "${updated.name || updated.ruleId}"`);
      },
    },
  );

  const [mutateDeleteNoisy] = useMutation<{ deleteNoisyOperationRule: boolean }, { samplingId: string; ruleId: string }>(DELETE_NOISY_OPERATION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, Crud.Delete, error.cause?.message || error.message),
    onCompleted: (_res, req) => {
      const sid = req?.variables?.samplingId as string;
      const id = req?.variables?.ruleId as string;
      setSamplingRules((prev) =>
        updateGroup(prev, sid, (g) => ({
          ...g,
          noisyOperations: g.noisyOperations.filter((r) => r.ruleId !== id),
        })),
      );
      notifyUser(StatusType.Success, Crud.Delete, `Successfully deleted noisy operation rule`);
    },
  });

  // ---- Highly Relevant Operations ----

  const [mutateCreateHighlyRelevant] = useMutation<{ createHighlyRelevantOperationRule: HighlyRelevantOperationRule }, { samplingId: string; rule: HighlyRelevantOperationRuleInput }>(
    CREATE_HIGHLY_RELEVANT_OPERATION_RULE,
    {
      onError: (error) => notifyUser(StatusType.Error, Crud.Create, error.cause?.message || error.message),
      onCompleted: (res, opts) => {
        const rule = res.createHighlyRelevantOperationRule;
        const sid = opts?.variables?.samplingId as string;
        setSamplingRules((prev) => updateGroup(prev, sid, (g) => ({ ...g, highlyRelevantOperations: [...g.highlyRelevantOperations, rule] })));
        notifyUser(StatusType.Success, Crud.Create, `Successfully created highly relevant operation rule "${rule.name || rule.ruleId}"`);
      },
    },
  );

  const [mutateUpdateHighlyRelevant] = useMutation<
    { updateHighlyRelevantOperationRule: HighlyRelevantOperationRule },
    { samplingId: string; ruleId: string; rule: HighlyRelevantOperationRuleInput }
  >(UPDATE_HIGHLY_RELEVANT_OPERATION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, Crud.Update, error.cause?.message || error.message),
    onCompleted: (res, opts) => {
      const updated = res.updateHighlyRelevantOperationRule;
      const sid = opts?.variables?.samplingId as string;
      setSamplingRules((prev) =>
        updateGroup(prev, sid, (g) => ({
          ...g,
          highlyRelevantOperations: g.highlyRelevantOperations.map((r) => (r.ruleId === updated.ruleId ? updated : r)),
        })),
      );
      notifyUser(StatusType.Success, Crud.Update, `Successfully updated highly relevant operation rule "${updated.name || updated.ruleId}"`);
    },
  });

  const [mutateDeleteHighlyRelevant] = useMutation<{ deleteHighlyRelevantOperationRule: boolean }, { samplingId: string; ruleId: string }>(DELETE_HIGHLY_RELEVANT_OPERATION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, Crud.Delete, error.cause?.message || error.message),
    onCompleted: (_res, req) => {
      const sid = req?.variables?.samplingId as string;
      const id = req?.variables?.ruleId as string;
      setSamplingRules((prev) =>
        updateGroup(prev, sid, (g) => ({
          ...g,
          highlyRelevantOperations: g.highlyRelevantOperations.filter((r) => r.ruleId !== id),
        })),
      );
      notifyUser(StatusType.Success, Crud.Delete, `Successfully deleted highly relevant operation rule`);
    },
  });

  // ---- Cost Reduction Rules ----

  const [mutateCreateCostReduction] = useMutation<{ createCostReductionRule: CostReductionRule }, { samplingId: string; rule: CostReductionRuleInput }>(CREATE_COST_REDUCTION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, Crud.Create, error.cause?.message || error.message),
    onCompleted: (res, opts) => {
      const rule = res.createCostReductionRule;
      const sid = opts?.variables?.samplingId as string;
      setSamplingRules((prev) => updateGroup(prev, sid, (g) => ({ ...g, costReductionRules: [...g.costReductionRules, rule] })));
      notifyUser(StatusType.Success, Crud.Create, `Successfully created cost reduction rule "${rule.name || rule.ruleId}"`);
    },
  });

  const [mutateUpdateCostReduction] = useMutation<{ updateCostReductionRule: CostReductionRule }, { samplingId: string; ruleId: string; rule: CostReductionRuleInput }>(
    UPDATE_COST_REDUCTION_RULE,
    {
      onError: (error) => notifyUser(StatusType.Error, Crud.Update, error.cause?.message || error.message),
      onCompleted: (res, opts) => {
        const updated = res.updateCostReductionRule;
        const sid = opts?.variables?.samplingId as string;
        setSamplingRules((prev) =>
          updateGroup(prev, sid, (g) => ({
            ...g,
            costReductionRules: g.costReductionRules.map((r) => (r.ruleId === updated.ruleId ? updated : r)),
          })),
        );
        notifyUser(StatusType.Success, Crud.Update, `Successfully updated cost reduction rule "${updated.name || updated.ruleId}"`);
      },
    },
  );

  const [mutateDeleteCostReduction] = useMutation<{ deleteCostReductionRule: boolean }, { samplingId: string; ruleId: string }>(DELETE_COST_REDUCTION_RULE, {
    onError: (error) => notifyUser(StatusType.Error, Crud.Delete, error.cause?.message || error.message),
    onCompleted: (_res, req) => {
      const sid = req?.variables?.samplingId as string;
      const id = req?.variables?.ruleId as string;
      setSamplingRules((prev) =>
        updateGroup(prev, sid, (g) => ({
          ...g,
          costReductionRules: g.costReductionRules.filter((r) => r.ruleId !== id),
        })),
      );
      notifyUser(StatusType.Success, Crud.Delete, `Successfully deleted cost reduction rule`);
    },
  });

  // ---- Public API ----

  const createNoisyOperationRule: UseSamplingRuleCrud['createNoisyOperationRule'] = (samplingId, rule) => {
    mutateCreateNoisy({ variables: { samplingId, rule } });
  };

  const updateNoisyOperationRule: UseSamplingRuleCrud['updateNoisyOperationRule'] = (samplingId, ruleId, rule) => {
    mutateUpdateNoisy({ variables: { samplingId, ruleId, rule } });
  };

  const deleteNoisyOperationRule: UseSamplingRuleCrud['deleteNoisyOperationRule'] = (samplingId, ruleId) => {
    mutateDeleteNoisy({ variables: { samplingId, ruleId } });
  };

  const createHighlyRelevantOperationRule: UseSamplingRuleCrud['createHighlyRelevantOperationRule'] = (samplingId, rule) => {
    mutateCreateHighlyRelevant({ variables: { samplingId, rule } });
  };

  const updateHighlyRelevantOperationRule: UseSamplingRuleCrud['updateHighlyRelevantOperationRule'] = (samplingId, ruleId, rule) => {
    mutateUpdateHighlyRelevant({ variables: { samplingId, ruleId, rule } });
  };

  const deleteHighlyRelevantOperationRule: UseSamplingRuleCrud['deleteHighlyRelevantOperationRule'] = (samplingId, ruleId) => {
    mutateDeleteHighlyRelevant({ variables: { samplingId, ruleId } });
  };

  const createCostReductionRule: UseSamplingRuleCrud['createCostReductionRule'] = (samplingId, rule) => {
    mutateCreateCostReduction({ variables: { samplingId, rule } });
  };

  const updateCostReductionRule: UseSamplingRuleCrud['updateCostReductionRule'] = (samplingId, ruleId, rule) => {
    mutateUpdateCostReduction({ variables: { samplingId, ruleId, rule } });
  };

  const deleteCostReductionRule: UseSamplingRuleCrud['deleteCostReductionRule'] = (samplingId, ruleId) => {
    mutateDeleteCostReduction({ variables: { samplingId, ruleId } });
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
