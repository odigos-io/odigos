import React, { useState } from 'react';
import { ActionDrawer, ActionModal, DestinationDrawer, DestinationModal, InstrumentationRuleDrawer, InstrumentationRuleModal, SourceDrawer, SourceModal } from '@odigos/ui-containers';
import {
  useActionCRUD,
  useConfig,
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
  const { isEnterprise } = useConfig();

  const { sources, persistSources, updateSource } = useSourceCRUD();
  const { actions, createAction, updateAction, deleteAction } = useActionCRUD();
  const { destinations, createDestination, updateDestination, deleteDestination } = useDestinationCRUD();
  const { instrumentationRules, createInstrumentationRule, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();

  const [selectedNamespace, setSelectedNamespace] = useState('');
  const { allNamespaces, data: namespace, loading: nsLoad } = useNamespace(selectedNamespace);

  const { categories } = useDestinationCategories();
  const { fetchDescribeSource } = useDescribeSource();
  const { potentialDestinations } = usePotentialDestinations();
  const { data: testResult, loading: testLoading, testConnection } = useTestConnection();

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
      <InstrumentationRuleModal isEnterprise={isEnterprise} createInstrumentationRule={createInstrumentationRule} />
      <ActionModal createAction={createAction} />

      {/* drawers */}
      <SourceDrawer sources={sources} persistSources={persistSources} updateSource={updateSource} fetchDescribeSource={fetchDescribeSource} />
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
