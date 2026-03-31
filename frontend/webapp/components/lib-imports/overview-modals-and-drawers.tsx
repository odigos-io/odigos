import React, { useCallback } from 'react';
import { useModalStore } from '@odigos/ui-kit/store';
import { EntityTypes } from '@odigos/ui-kit/types';
import { ActionDrawer, DestinationDrawer, InstrumentationRuleDrawer, SourceDrawer } from '@odigos/ui-kit/containers';
import {
  AddActionDrawer,
  AddDestinationDrawer,
  AddRuleDrawer,
  AddSourceDrawer,
  AddActionFormContextProvider,
  AddDestinationFormContextProvider,
  AddRuleFormContextProvider,
  AddSourceFormContextProvider,
} from '@odigos/ui-kit/containers/v2';
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
  const { currentModal, setCurrentModal } = useModalStore();

  const { fetchDescribeSource } = useDescribe();
  const { testConnection } = useTestConnection();
  const { fetchNamespacesWithWorkloads } = useNamespace();
  const { getPotentialDestinations } = usePotentialDestinations();
  const { createActionV2, updateAction, deleteAction } = useActionCRUD();
  const { categories, getDestinationCategories } = useDestinationCategories();
  const { restartWorkloads, restartPod, recoverFromRollback } = useWorkloadUtils();
  const { createDestinationV2, updateDestination, deleteDestination } = useDestinationCRUD();
  const { createInstrumentationRuleV2, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();
  const { persistSources, persistSourcesV2, updateSource, fetchSourceById, fetchSourceLibraries, fetchPeerSources } = useSourceCRUD();

  const handleCloseModal = useCallback(() => setCurrentModal(''), [setCurrentModal]);

  return (
    <>
      {/* add drawers (v2) */}
      {currentModal === EntityTypes.Source && (
        <AddSourceFormContextProvider fetchNamespacesWithWorkloads={fetchNamespacesWithWorkloads}>
          <AddSourceDrawer onClose={handleCloseModal} persistSources={persistSourcesV2} />
        </AddSourceFormContextProvider>
      )}
      {currentModal === EntityTypes.Destination && (
        <AddDestinationFormContextProvider>
          <AddDestinationDrawer
            onClose={handleCloseModal}
            getDestinationCategories={getDestinationCategories}
            getPotentialDestinations={getPotentialDestinations}
            testConnection={testConnection}
            createDestination={createDestinationV2}
          />
        </AddDestinationFormContextProvider>
      )}
      {currentModal === EntityTypes.InstrumentationRule && (
        <AddRuleFormContextProvider>
          <AddRuleDrawer onClose={handleCloseModal} createInstrumentationRule={createInstrumentationRuleV2} />
        </AddRuleFormContextProvider>
      )}
      {currentModal === EntityTypes.Action && (
        <AddActionFormContextProvider>
          <AddActionDrawer onClose={handleCloseModal} createAction={createActionV2} />
        </AddActionFormContextProvider>
      )}

      {/* edit drawers */}
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
      />
      <DestinationDrawer categories={categories} updateDestination={updateDestination} deleteDestination={deleteDestination} testConnection={testConnection} />
      <InstrumentationRuleDrawer updateInstrumentationRule={updateInstrumentationRule} deleteInstrumentationRule={deleteInstrumentationRule} />
      <ActionDrawer updateAction={updateAction} deleteAction={deleteAction} />
    </>
  );
};

export { OverviewModalsAndDrawers };
