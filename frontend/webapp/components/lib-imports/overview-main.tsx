import React from 'react';
import styled from 'styled-components';
import type { Source } from '@odigos/ui-utils';
import { DataFlow, DataFlowActionsMenu, MultiSourceControl, type SourceSelectionFormData } from '@odigos/ui-containers';
import { useActionCRUD, useDestinationCRUD, useInstrumentationRuleCRUD, useMetrics, useNamespace, useSourceCRUD } from '@/hooks';

export const MainContent = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

const OverviewMain = () => {
  const { metrics } = useMetrics();
  const { allNamespaces } = useNamespace();
  const { actions, actionsLoading } = useActionCRUD();
  const { destinations, destinationsLoading } = useDestinationCRUD();
  const { sources, sourcesLoading, persistSources } = useSourceCRUD();
  const { instrumentationRules, instrumentationRulesLoading } = useInstrumentationRuleCRUD();

  return (
    <MainContent>
      <DataFlowActionsMenu namespaces={allNamespaces} sources={sources} destinations={destinations} actions={actions} instrumentationRules={instrumentationRules} />
      <DataFlow
        heightToRemove='176px'
        sources={sources}
        sourcesLoading={sourcesLoading}
        destinations={destinations}
        destinationsLoading={destinationsLoading}
        actions={actions}
        actionsLoading={actionsLoading}
        instrumentationRules={instrumentationRules}
        instrumentationRulesLoading={instrumentationRulesLoading}
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
