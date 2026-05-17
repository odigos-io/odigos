'use client';

import React from 'react';
import { Overview } from '@odigos/ui-kit/containers/v2';
import {
  useActionCRUD,
  useDataStreamsCRUD,
  useDescribe,
  useDestinationCRUD,
  useDestinationCategories,
  useEffectiveConfig,
  useInstrumentationRuleCRUD,
  useMetrics,
  useNamespace,
  usePotentialDestinations,
  useProfiling,
  useSourceCRUD,
  useTestConnection,
  useWorkloadUtils,
} from '@/hooks';

export default function Page() {
  const { metrics } = useMetrics();
  const { effectiveConfig } = useEffectiveConfig();

  const { fetchActions } = useActionCRUD();
  const { fetchDescribeSource } = useDescribe();
  const { testConnection } = useTestConnection();
  const { fetchDestinations } = useDestinationCRUD();
  const { fetchNamespacesWithWorkloads } = useNamespace();
  const { getDestinationCategories } = useDestinationCategories();
  const { getPotentialDestinations } = usePotentialDestinations();
  const { updateDataStream, deleteDataStream } = useDataStreamsCRUD();
  const { createActionV2, updateAction, deleteAction } = useActionCRUD();
  const { restartWorkloads, restartPod, recoverFromRollback } = useWorkloadUtils();
  const { fetchProfilingSlots, enableProfiling, fetchSourceProfiling } = useProfiling();
  const { createDestination, updateDestination, deleteDestination } = useDestinationCRUD();
  const { fetchSources, persistSourcesV2, updateSource, fetchSourceById, fetchSourceLibraries, fetchPeerSources } = useSourceCRUD();
  const { fetchInstrumentationRules, createInstrumentationRuleV2, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();

  return (
    <Overview
      columnsMaxHeight={`calc(100vh - ${224}px)`}
      metrics={metrics}
      effectiveConfig={effectiveConfig}
      refetchSources={fetchSources}
      refetchDestinations={fetchDestinations}
      refetchActions={fetchActions}
      refetchInstrumentationRules={fetchInstrumentationRules}
      updateDataStream={updateDataStream}
      deleteDataStream={deleteDataStream}
      fetchNamespacesWithWorkloads={fetchNamespacesWithWorkloads}
      persistSources={persistSourcesV2}
      restartWorkloads={restartWorkloads}
      restartPod={restartPod}
      recoverFromRollback={recoverFromRollback}
      updateSource={updateSource}
      fetchSourceById={fetchSourceById}
      fetchSourceDescribe={fetchDescribeSource}
      fetchSourceLibraries={fetchSourceLibraries}
      fetchPeerSources={fetchPeerSources}
      enableProfiling={enableProfiling}
      fetchProfilingSlots={fetchProfilingSlots}
      fetchSourceProfiling={fetchSourceProfiling}
      getDestinationCategories={getDestinationCategories}
      getPotentialDestinations={getPotentialDestinations}
      testConnection={testConnection}
      createDestination={createDestination}
      updateDestination={updateDestination}
      deleteDestination={deleteDestination}
      createInstrumentationRule={createInstrumentationRuleV2}
      updateInstrumentationRule={updateInstrumentationRule}
      deleteInstrumentationRule={deleteInstrumentationRule}
      createAction={createActionV2}
      updateAction={updateAction}
      deleteAction={deleteAction}
    />
  );
}
