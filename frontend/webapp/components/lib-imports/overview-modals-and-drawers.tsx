import React from 'react';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { AddSourceModal, SourceDrawer } from '@/containers';
import { useActionCRUD, useDescribeOdigos, useDestinationCategories, useDestinationCRUD, useInstrumentationRuleCRUD, usePotentialDestinations, useTestConnection } from '@/hooks';
import { ActionDrawer, ActionModal, DestinationDrawer, DestinationModal, InstrumentationRuleDrawer, InstrumentationRuleModal, useDrawerStore, useModalStore } from '@odigos/ui-containers';

const OverviewModalsAndDrawers = () => {
  const { drawerType } = useDrawerStore();
  const { currentModal, setCurrentModal } = useModalStore();
  const handleClose = () => setCurrentModal('');

  const { isPro } = useDescribeOdigos();
  const { categories } = useDestinationCategories();
  const { potentialDestinations } = usePotentialDestinations();
  const { actions, createAction, updateAction, deleteAction } = useActionCRUD();
  const { data: testResult, loading: testLoading, testConnection } = useTestConnection();
  const { destinations, createDestination, updateDestination, deleteDestination } = useDestinationCRUD();
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
      {drawerType === ENTITY_TYPES.SOURCE && <SourceDrawer />}
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
