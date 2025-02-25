import React from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import styled from 'styled-components';
import type { Source } from '@odigos/ui-utils';
import { FlexRow } from '@odigos/ui-components';
import { useActionCRUD, useDestinationCRUD, useInstrumentationRuleCRUD, useMetrics, useNamespace, useSourceCRUD } from '@/hooks';
import { DataFlow, DataFlowActionsMenu, MultiSourceControl, NAV_ICON_IDS, SideNav, type SourceSelectionFormData } from '@odigos/ui-containers';

const Container = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

const MainContent = styled(FlexRow)`
  align-items: flex-start;
  padding-left: 12px;
  width: calc(100% - 12px);
`;

const OverviewMain = () => {
  const router = useRouter();
  const { metrics } = useMetrics();
  const { allNamespaces } = useNamespace();
  const { actions, actionsLoading } = useActionCRUD();
  const { destinations, destinationsLoading } = useDestinationCRUD();
  const { sources, sourcesLoading, persistSources } = useSourceCRUD();
  const { instrumentationRules, instrumentationRulesLoading } = useInstrumentationRuleCRUD();

  return (
    <Container>
      <DataFlowActionsMenu namespaces={allNamespaces} sources={sources} destinations={destinations} actions={actions} instrumentationRules={instrumentationRules} />

      <MainContent>
        <SideNav
          defaultSelectedId={NAV_ICON_IDS.OVERVIEW}
          onClickOverview={() => router.push(ROUTES.OVERVIEW)}
          onClickRules={() => router.push(ROUTES.OVERVIEW_INSTRUMENTATION_RULES)}
          onClickSources={() => router.push(ROUTES.OVERVIEW_SOURCES)}
          onClickActions={() => router.push(ROUTES.OVERVIEW_ACTIONS)}
          onClickDestinations={() => router.push(ROUTES.OVERVIEW_DESTINATIONS)}
        />
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
      </MainContent>

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
    </Container>
  );
};

export { OverviewMain };
