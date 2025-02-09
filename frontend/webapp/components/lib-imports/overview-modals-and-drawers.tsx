import React from 'react';
import { AddSourceModal } from '@/containers';
import { ENTITY_TYPES, type WorkloadId } from '@odigos/ui-utils';
import {
  ActionDrawer,
  ActionModal,
  DestinationDrawer,
  DestinationModal,
  InstrumentationRuleDrawer,
  InstrumentationRuleModal,
  SourceDrawer,
  useDrawerStore,
  useModalStore,
} from '@odigos/ui-containers';
import {
  useActionCRUD,
  useDescribeOdigos,
  useDescribeSource,
  useDestinationCategories,
  useDestinationCRUD,
  useInstrumentationRuleCRUD,
  usePotentialDestinations,
  useSourceCRUD,
  useTestConnection,
} from '@/hooks';

const OverviewModalsAndDrawers = () => {
  const { drawerEntityId } = useDrawerStore();
  const { currentModal, setCurrentModal } = useModalStore();
  const handleClose = () => setCurrentModal('');

  const { isPro } = useDescribeOdigos();
  const { categories } = useDestinationCategories();
  const { potentialDestinations } = usePotentialDestinations();
  const { sources, persistSources, updateSource } = useSourceCRUD();
  const { actions, createAction, updateAction, deleteAction } = useActionCRUD();
  const { data: testResult, loading: testLoading, testConnection } = useTestConnection();
  const { destinations, createDestination, updateDestination, deleteDestination } = useDestinationCRUD();
  const { data: describeSource } = useDescribeSource(typeof drawerEntityId === 'object' ? (drawerEntityId as WorkloadId) : undefined);
  const { instrumentationRules, createInstrumentationRule, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();

  return (
    <>
      {/* modals */}
      {currentModal === ENTITY_TYPES.SOURCE && <AddSourceModal isOpen onClose={handleClose} />}
      <DestinationModal
        isOnboarding={false}
        addConfiguredDestination={() => {}}
        categories={categories}
        potentialDestinations={potentialDestinations}
        createDestination={createDestination}
        testConnection={testConnection}
        testLoading={testLoading}
        testResult={testResult}
      />
      <InstrumentationRuleModal isEnterprise={isPro} createInstrumentationRule={createInstrumentationRule} />
      <ActionModal createAction={createAction} />

      {/* drawers */}
      <SourceDrawer sources={sources} persistSources={persistSources} updateSource={updateSource} describe={describeSource} />
      <DestinationDrawer
        categories={categories}
        destinations={destinations}
        updateDestination={updateDestination}
        deleteDestination={deleteDestination}
        testConnection={testConnection}
        testLoading={testLoading}
        testResult={testResult}
      />
      <InstrumentationRuleDrawer instrumentationRules={instrumentationRules} updateInstrumentationRule={updateInstrumentationRule} deleteInstrumentationRule={deleteInstrumentationRule} />
      <ActionDrawer actions={actions} updateAction={updateAction} deleteAction={deleteAction} />
    </>
  );
};

export default OverviewModalsAndDrawers;
