'use client';

import React, { useCallback, useState } from 'react';
import type { WorkloadId } from '@odigos/ui-kit/types';
import { Overview } from '@odigos/ui-kit/containers/v2';
import { ProfileTypeToggle } from '@/components/profiling/profile-type-toggle';
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
  type ProfileType,
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

  // CPU⇄memory toggle state. Selecting a type re-queries the flame graph with that sample type.
  const [profileType, setProfileType] = useState<ProfileType>('cpu');
  // Wrap fetchSourceProfiling so the flame graph view always uses the currently-selected profile type.
  const fetchSourceProfilingByType = useCallback((source: WorkloadId) => fetchSourceProfiling(source, profileType), [fetchSourceProfiling, profileType]);
  const { createDestination, updateDestination, deleteDestination } = useDestinationCRUD();
  const { fetchSources, persistSourcesV2, updateSource, fetchSourceById, fetchSourceLibraries, fetchPeerSources } = useSourceCRUD();
  const { fetchInstrumentationRules, createInstrumentationRuleV2, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'flex-end', padding: '8px 16px' }}>
        <ProfileTypeToggle value={profileType} onChange={setProfileType} />
      </div>
      <Overview
      columnsMaxHeight='calc(100vh - 224px)'
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
      fetchSourceProfiling={fetchSourceProfilingByType}
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
    </>
  );
}
