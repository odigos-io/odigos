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
  useTestConnection,
  useWorkloadUtils,
} from '@/hooks';

const OverviewModalsAndDrawers = () => {
  const { fetchNamespacesWithWorkloads } = useNamespace();
  const { fetchDescribeSource } = useDescribe();
  const { testConnection } = useTestConnection();
  const { categories } = useDestinationCategories();
  const { restartWorkloads, restartPod } = useWorkloadUtils();
  const { potentialDestinations } = usePotentialDestinations();
  const { createAction, updateAction, deleteAction } = useActionCRUD();
  const { persistSources, updateSource, fetchSourceById, fetchSourceLibraries } = useSourceCRUD();
  const { createDestination, updateDestination, deleteDestination } = useDestinationCRUD();
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
        updateSource={updateSource}
        fetchSourceById={fetchSourceById}
        fetchSourceDescribe={fetchDescribeSource}
        fetchSourceLibraries={fetchSourceLibraries}
      />
      <DestinationDrawer categories={categories} updateDestination={updateDestination} deleteDestination={deleteDestination} testConnection={testConnection} />
      <InstrumentationRuleDrawer updateInstrumentationRule={updateInstrumentationRule} deleteInstrumentationRule={deleteInstrumentationRule} />
      <ActionDrawer updateAction={updateAction} deleteAction={deleteAction} />
    </>
  );
};

export { OverviewModalsAndDrawers };
