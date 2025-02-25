import React from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import styled from 'styled-components';
import { FlexRow } from '@odigos/ui-components';
import { useNamespace, useSourceCRUD } from '@/hooks';
import { ENTITY_TYPES, type Source } from '@odigos/ui-utils';
import { DataFlowActionsMenu, MultiSourceControl, NAV_ICON_IDS, SideNav, SourceTable, type SourceSelectionFormData } from '@odigos/ui-containers';

const Container = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

const MainContent = styled(FlexRow)`
  align-items: flex-start;
  width: calc(100% - 24px);
  padding: 0 12px;
  gap: 12px;
`;

const OverviewFocusedSources = () => {
  const router = useRouter();
  const { allNamespaces } = useNamespace();
  const { sources, persistSources } = useSourceCRUD();

  return (
    <Container>
      <DataFlowActionsMenu namespaces={allNamespaces} sources={sources} destinations={[]} actions={[]} instrumentationRules={[]} addEntity={ENTITY_TYPES.SOURCE} />

      <MainContent>
        <SideNav
          defaultSelectedId={NAV_ICON_IDS.SOURCES}
          onClickOverview={() => router.push(ROUTES.OVERVIEW)}
          onClickRules={() => router.push(ROUTES.OVERVIEW_INSTRUMENTATION_RULES)}
          onClickSources={() => router.push(ROUTES.OVERVIEW_SOURCES)}
          onClickActions={() => router.push(ROUTES.OVERVIEW_ACTIONS)}
          onClickDestinations={() => router.push(ROUTES.OVERVIEW_DESTINATIONS)}
        />
        <SourceTable sources={sources} tableMaxHeight='calc(100vh - 220px)' />
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

export { OverviewFocusedSources };
