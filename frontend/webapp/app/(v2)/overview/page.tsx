'use client';

import React from 'react';
import { Overview } from '@odigos/ui-kit/containers/v2';
import {
  useActionCRUD,
  useDataStreamsCRUD,
  useDestinationCRUD,
  useDestinationCategories,
  useEffectiveConfig,
  useInstrumentationRuleCRUD,
  useK8sManifest,
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
  const { fetchK8sManifest } = useK8sManifest();
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
  const { sources, fetchSources, persistSourcesV2, updateSource, fetchSourceById, fetchPeerSources } = useSourceCRUD();
  const { fetchInstrumentationRules, createInstrumentationRuleV2, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();

  return (
    <Overview
      metrics={metrics}
      effectiveConfig={effectiveConfig}
      workloads={sources}
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
      fetchPeerSources={fetchPeerSources}
      enableProfiling={enableProfiling}
      fetchProfilingSlots={fetchProfilingSlots}
      fetchSourceProfiling={fetchSourceProfiling}
      fetchK8sManifest={fetchK8sManifest}
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
