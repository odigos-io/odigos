'use client';

import React from 'react';
import { SamplingRules } from '@odigos/ui-kit/containers/v2';
import { useSamplingRuleCRUD, useWorkloads } from '@/hooks';

export default function Page() {
  const { workloads } = useWorkloads({ markedForInstrumentation: true });
  const {
    samplingRules,
    k8sHealthProbesConfig,
    loading,
    fetchSamplingRules,
    createNoisyOperationRule,
    updateNoisyOperationRule,
    createHighlyRelevantOperationRule,
    updateHighlyRelevantOperationRule,
    createCostReductionRule,
    updateCostReductionRule,
    deleteSamplingRule,
    updateK8sHealthProbesConfig,
  } = useSamplingRuleCRUD();

  return (
    <SamplingRules
      workloads={workloads}
      samplingRules={samplingRules}
      k8sHealthProbesConfig={k8sHealthProbesConfig}
      loading={loading}
      tableRowsMaxHeight='calc(100vh - 480px)'
      fetchSamplingRules={fetchSamplingRules}
      createNoisyOperationRule={createNoisyOperationRule}
      updateNoisyOperationRule={updateNoisyOperationRule}
      createHighlyRelevantOperationRule={createHighlyRelevantOperationRule}
      updateHighlyRelevantOperationRule={updateHighlyRelevantOperationRule}
      createCostReductionRule={createCostReductionRule}
      updateCostReductionRule={updateCostReductionRule}
      deleteSamplingRule={deleteSamplingRule}
      updateK8sHealthProbesConfig={updateK8sHealthProbesConfig}
    />
  );
}
