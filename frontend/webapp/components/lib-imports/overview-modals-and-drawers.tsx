import React from 'react';
import { ActionDrawer, ActionModal, DestinationDrawer, DestinationModal, InstrumentationRuleDrawer, InstrumentationRuleModal, SourceDrawer, SourceModal } from '@odigos/ui-kit/containers';
import {
  useActionCRUD,
  useDescribe,
  useDestinationCategories,
  useDestinationCRUD,
  useInstrumentationRuleCRUD,
  useNamespace,
  usePotentialDestinations,
  useSourceCRUD,
  useProfiling,
  useTestConnection,
  useWorkloadUtils,
} from '@/hooks';

const OverviewModalsAndDrawers = () => {
  const { fetchDescribeSource } = useDescribe();
  const { testConnection } = useTestConnection();
  const { categories } = useDestinationCategories();
  const { fetchNamespacesWithWorkloads } = useNamespace();
  const { potentialDestinations } = usePotentialDestinations();
  const { createAction, updateAction, deleteAction } = useActionCRUD();
  const { restartWorkloads, restartPod, recoverFromRollback } = useWorkloadUtils();
  const { createDestination, updateDestination, deleteDestination } = useDestinationCRUD();
  const { fetchProfilingSlots, enableProfiling, releaseProfiling, fetchSourceProfiling } = useProfiling();
  const { persistSources, updateSource, fetchSourceById, fetchSourceLibraries, fetchPeerSources } = useSourceCRUD();
  const { createInstrumentationRule, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();

  return (
    <>
      {/* modals */}
      <SourceModal fetchNamespacesWithWorkloads={fetchNamespacesWithWorkloads} persistSources={persistSources} />
      <DestinationModal
        isOnboarding={false}
        categories={categories}
        potentialDestinations={potentialDestinations}
        createDestination={createDestination}
        updateDestination={updateDestination}
        deleteDestination={deleteDestination}
        testConnection={testConnection}
      />
      <InstrumentationRuleModal createInstrumentationRule={createInstrumentationRule} />
      <ActionModal createAction={createAction} />

      {/* drawers */}
      <SourceDrawer
        persistSources={persistSources}
        restartWorkloads={restartWorkloads}
        restartPod={restartPod}
        recoverFromRollback={recoverFromRollback}
        updateSource={updateSource}
        fetchSourceById={fetchSourceById}
        fetchSourceDescribe={fetchDescribeSource}
        fetchSourceLibraries={fetchSourceLibraries}
        fetchPeerSources={fetchPeerSources}
        fetchProfilingSlots={fetchProfilingSlots}
        enableProfiling={enableProfiling}
        releaseProfiling={releaseProfiling}
        fetchSourceProfiling={fetchSourceProfiling}
      />
      <DestinationDrawer categories={categories} updateDestination={updateDestination} deleteDestination={deleteDestination} testConnection={testConnection} />
      <InstrumentationRuleDrawer updateInstrumentationRule={updateInstrumentationRule} deleteInstrumentationRule={deleteInstrumentationRule} />
      <ActionDrawer updateAction={updateAction} deleteAction={deleteAction} />
    </>
  );
};

export { OverviewModalsAndDrawers };
