import React, { useState } from 'react';
import { ActionDrawer, ActionModal, DestinationDrawer, DestinationModal, InstrumentationRuleDrawer, InstrumentationRuleModal, SourceDrawer, SourceModal } from '@odigos/ui-kit/containers';
import {
  useActionCRUD,
  useConfig,
  useDescribe,
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

  const { fetchDescribeSource } = useDescribe();
  const { categories } = useDestinationCategories();
  const { persistSources, updateSource } = useSourceCRUD();
  const { potentialDestinations } = usePotentialDestinations();
  const { createAction, updateAction, deleteAction } = useActionCRUD();
  const { testConnection, testConnectionResult, isTestConnectionLoading } = useTestConnection();
  const { createDestination, updateDestination, deleteDestination } = useDestinationCRUD();
  const { createInstrumentationRule, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();

  const [selectedNamespace, setSelectedNamespace] = useState('');
  const { namespaces, namespace, loading: nsLoad } = useNamespace(selectedNamespace);

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
        testResult={testConnectionResult}
        testLoading={isTestConnectionLoading}
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
        testResult={testConnectionResult}
        testLoading={isTestConnectionLoading}
      />
      <InstrumentationRuleDrawer updateInstrumentationRule={updateInstrumentationRule} deleteInstrumentationRule={deleteInstrumentationRule} />
      <ActionDrawer updateAction={updateAction} deleteAction={deleteAction} />
    </>
  );
};

export { OverviewModalsAndDrawers };
