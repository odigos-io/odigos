import React, { useCallback } from 'react';
import { EntityTypes } from '@odigos/ui-kit/types';
import { useDrawerStore, useModalStore } from '@odigos/ui-kit/store';
import { ActionFormContextProvider, DestinationFormContextProvider, RuleFormContextProvider, SourceInstrumentFormContextProvider } from '@odigos/ui-kit/contexts';
import { AddActionDrawer, AddDestinationDrawer, AddRuleDrawer, AddSourceDrawer, EditActionDrawer, EditDestinationDrawer, EditRuleDrawer } from '@odigos/ui-kit/containers/v2';
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
import { SourceDrawer } from '@odigos/ui-kit/containers';

const OverviewModalsAndDrawers = () => {
  const { currentModal, setCurrentModal } = useModalStore();
  const { drawerType, drawerEntityId, setDrawerType, setDrawerEntityId } = useDrawerStore();

  const { fetchDescribeSource } = useDescribe();
  const { testConnection } = useTestConnection();
  const { fetchNamespacesWithWorkloads } = useNamespace();
  const { getDestinationCategories } = useDestinationCategories();
  const { getPotentialDestinations } = usePotentialDestinations();
  const { createActionV2, updateAction, deleteAction } = useActionCRUD();
  const { restartWorkloads, restartPod, recoverFromRollback } = useWorkloadUtils();
  const { createDestination, updateDestination, deleteDestination } = useDestinationCRUD();
  const { fetchProfilingSlots, enableProfiling, releaseProfiling, fetchSourceProfiling } = useProfiling();
  const { createInstrumentationRuleV2, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();
  const { persistSources, persistSourcesV2, updateSource, fetchSourceById, fetchSourceLibraries, fetchPeerSources } = useSourceCRUD();

  const handleCloseModal = useCallback(() => {
    setCurrentModal('');
  }, []);
  const handleCloseDrawer = useCallback(() => {
    setDrawerType('');
    setDrawerEntityId(null);
  }, []);

  return (
    <>
      {/* add drawers (v2) */}
      {currentModal === EntityTypes.Source && (
        <SourceInstrumentFormContextProvider>
          <AddSourceDrawer onClose={handleCloseModal} fetchNamespacesWithWorkloads={fetchNamespacesWithWorkloads} persistSources={persistSourcesV2} withOverlay />
        </SourceInstrumentFormContextProvider>
      )}
      {currentModal === EntityTypes.Destination && (
        <DestinationFormContextProvider>
          <AddDestinationDrawer
            onClose={handleCloseModal}
            getDestinationCategories={getDestinationCategories}
            getPotentialDestinations={getPotentialDestinations}
            testConnection={testConnection}
            createDestination={createDestination}
            updateDestination={updateDestination}
            withOverlay
          />
        </DestinationFormContextProvider>
      )}
      {currentModal === EntityTypes.InstrumentationRule && (
        <RuleFormContextProvider>
          <AddRuleDrawer onClose={handleCloseModal} createInstrumentationRule={createInstrumentationRuleV2} withOverlay />
        </RuleFormContextProvider>
      )}
      {currentModal === EntityTypes.Action && (
        <ActionFormContextProvider>
          <AddActionDrawer onClose={handleCloseModal} createAction={createActionV2} withOverlay />
        </ActionFormContextProvider>
      )}

      {/* edit drawers */}
      {drawerType === EntityTypes.Source && drawerEntityId && (
        // <SourceEditFormContextProvider>
        //   <EditSourceDrawer
        //     onClose={handleCloseDrawer}
        //     sourceId={drawerEntityId as WorkloadId}
        //     persistSources={persistSources}
        //     restartWorkloads={restartWorkloads}
        //     restartPod={restartPod}
        //     recoverFromRollback={recoverFromRollback}
        //     updateSource={updateSource}
        //     fetchSourceById={fetchSourceById}
        //     fetchSourceDescribe={fetchDescribeSource}
        //     fetchSourceLibraries={fetchSourceLibraries}
        //     fetchPeerSources={fetchPeerSources}
        //     enableProfiling={enableProfiling}
        //     releaseProfiling={releaseProfiling}
        //     fetchSourceProfiling={fetchSourceProfiling}
        //     fetchProfilingSlots={fetchProfilingSlots}
        //   />
        // </SourceEditFormContextProvider>
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
          enableProfiling={enableProfiling}
          releaseProfiling={releaseProfiling}
          fetchSourceProfiling={fetchSourceProfiling}
          fetchProfilingSlots={fetchProfilingSlots}
        />
      )}
      {drawerType === EntityTypes.Destination && drawerEntityId && (
        <DestinationFormContextProvider>
          <EditDestinationDrawer
            onClose={handleCloseDrawer}
            destinationId={drawerEntityId as string}
            getDestinationCategories={getDestinationCategories}
            testConnection={testConnection}
            updateDestination={updateDestination}
            deleteDestination={deleteDestination}
          />
        </DestinationFormContextProvider>
      )}
      {drawerType === EntityTypes.InstrumentationRule && drawerEntityId && (
        <RuleFormContextProvider>
          <EditRuleDrawer onClose={handleCloseDrawer} ruleId={drawerEntityId} updateInstrumentationRule={updateInstrumentationRule} deleteInstrumentationRule={deleteInstrumentationRule} />
        </RuleFormContextProvider>
      )}
      {drawerType === EntityTypes.Action && drawerEntityId && (
        <ActionFormContextProvider>
          <EditActionDrawer onClose={handleCloseDrawer} actionId={drawerEntityId} updateAction={updateAction} deleteAction={deleteAction} />
        </ActionFormContextProvider>
      )}
    </>
  );
};

export { OverviewModalsAndDrawers };
