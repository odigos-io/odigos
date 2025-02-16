import React from 'react';
import styled from 'styled-components';
import { usePaginatedStore } from '@/store';
import type { Source } from '@odigos/ui-utils';
import { DataFlow, DataFlowActionsMenu, MultiSourceControl, type SourceSelectionFormData } from '@odigos/ui-containers';
import { useActionCRUD, useDestinationCRUD, useInstrumentationRuleCRUD, useMetrics, useNamespace, useSourceCRUD } from '@/hooks';

export const MainContent = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

const OverviewMain = () => {
  const { sourcesFetching } = usePaginatedStore();

  const { metrics } = useMetrics();
  const { allNamespaces } = useNamespace();
  const { actions, filteredActions, loading: actLoad } = useActionCRUD();
  const { sources, filteredSources, loading: srcLoad, persistSources } = useSourceCRUD();
  const { destinations, filteredDestinations, loading: destLoad } = useDestinationCRUD();
  const { instrumentationRules, filteredInstrumentationRules, loading: ruleLoad } = useInstrumentationRuleCRUD();

  return (
    <MainContent>
      <DataFlowActionsMenu namespaces={allNamespaces} sources={filteredSources} destinations={filteredDestinations} actions={filteredActions} instrumentationRules={filteredInstrumentationRules} />
      <DataFlow
        heightToRemove='176px'
        sources={filteredSources}
        sourcesLoading={srcLoad || sourcesFetching}
        sourcesTotalCount={sources.length}
        destinations={filteredDestinations}
        destinationsLoading={destLoad}
        destinationsTotalCount={destinations.length}
        actions={filteredActions}
        actionsLoading={actLoad}
        actionsTotalCount={actions.length}
        instrumentationRules={filteredInstrumentationRules}
        instrumentationRulesLoading={ruleLoad}
        instrumentationRulesTotalCount={instrumentationRules.length}
        metrics={metrics}
      />
      <MultiSourceControl
        totalSourceCount={sources.length}
        uninstrumentSources={(payload) => {
          const inp: SourceSelectionFormData = {};

          Object.entries(payload).forEach(([namespace, sources]: [string, Source[]]) => {
            inp[namespace] = sources.map(({ name, kind }) => ({ name, kind, selected: false }));
          });

          persistSources(inp, {});
        }}
      />
    </MainContent>
  );
};

export default OverviewMain;
