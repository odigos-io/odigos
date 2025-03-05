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

  const { persistSources, updateSource } = useSourceCRUD();
  const { createAction, updateAction, deleteAction } = useActionCRUD();
  const { createDestination, updateDestination, deleteDestination } = useDestinationCRUD();
  const { createInstrumentationRule, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();

  const [selectedNamespace, setSelectedNamespace] = useState('');
  const { namespaces, data: namespace, loading: nsLoad } = useNamespace(selectedNamespace);

  const { categories } = useDestinationCategories();
  const { fetchDescribeSource } = useDescribeSource();
  const { potentialDestinations } = usePotentialDestinations();
  const { data: testResult, loading: testLoading, testConnection } = useTestConnection();

  return (
    <>
      {/* modals */}
      <SourceModal
        namespaces={namespaces}
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
      <SourceDrawer persistSources={persistSources} updateSource={updateSource} fetchDescribeSource={fetchDescribeSource} />
      <DestinationDrawer
        categories={categories}
        updateDestination={updateDestination}
        deleteDestination={deleteDestination}
        testConnection={testConnection}
        testLoading={testLoading}
        testResult={testResult}
      />
      <InstrumentationRuleDrawer updateInstrumentationRule={updateInstrumentationRule} deleteInstrumentationRule={deleteInstrumentationRule} />
      <ActionDrawer updateAction={updateAction} deleteAction={deleteAction} />
    </>
  );
};

export { OverviewModalsAndDrawers };
