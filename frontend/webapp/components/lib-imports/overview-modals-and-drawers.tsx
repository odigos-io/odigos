import React, { useState } from 'react';
import { type WorkloadId } from '@odigos/ui-utils';
import { ActionDrawer, ActionModal, DestinationDrawer, DestinationModal, InstrumentationRuleDrawer, InstrumentationRuleModal, SourceDrawer, SourceModal, useDrawerStore } from '@odigos/ui-containers';
import {
  useActionCRUD,
  useDescribeOdigos,
  useDescribeSource,
  useDestinationCategories,
  useDestinationCRUD,
  useInstrumentationRuleCRUD,
  useNamespace,
  usePotentialDestinations,
  useSourceCRUD,
  useTestConnection,
} from '@/hooks';

const OverviewModalsAndDrawers = () => {
  const { drawerEntityId } = useDrawerStore();

  const { sources, persistSources, updateSource } = useSourceCRUD();
  const { actions, createAction, updateAction, deleteAction } = useActionCRUD();
  const { destinations, createDestination, updateDestination, deleteDestination } = useDestinationCRUD();
  const { instrumentationRules, createInstrumentationRule, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();

  const [selectedNamespace, setSelectedNamespace] = useState('');
  const { allNamespaces, data: namespace, loading: nsLoad } = useNamespace(selectedNamespace);

  const { isPro } = useDescribeOdigos();
  const { categories } = useDestinationCategories();
  const { potentialDestinations } = usePotentialDestinations();
  const { data: testResult, loading: testLoading, testConnection } = useTestConnection();
  const { data: describeSource } = useDescribeSource(typeof drawerEntityId === 'object' ? (drawerEntityId as WorkloadId) : undefined);

  return (
    <>
      {/* modals */}
      <SourceModal
        namespaces={allNamespaces}
        namespace={namespace}
        namespacesLoading={nsLoad}
        selectedNamespace={selectedNamespace}
        setSelectedNamespace={setSelectedNamespace}
        persistSources={persistSources}
      />
      <DestinationModal
        isOnboarding={false}
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
