import React from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import styled from 'styled-components';
import { useDestinationCRUD } from '@/hooks';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { FlexRow } from '@odigos/ui-components';
import { DataFlowActionsMenu, DestinationTable, NAV_ICON_IDS, SideNav } from '@odigos/ui-containers';

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

const OverviewFocusedDestinations = () => {
  const router = useRouter();
  const { destinations } = useDestinationCRUD();

  return (
    <Container>
      <DataFlowActionsMenu namespaces={[]} sources={[]} destinations={destinations} actions={[]} instrumentationRules={[]} addEntity={ENTITY_TYPES.DESTINATION} />

      <MainContent>
        <SideNav
          defaultSelectedId={NAV_ICON_IDS.SOURCES}
          onClickOverview={() => router.push(ROUTES.OVERVIEW)}
          onClickRules={() => router.push(ROUTES.OVERVIEW_INSTRUMENTATION_RULES)}
          onClickSources={() => router.push(ROUTES.OVERVIEW_SOURCES)}
          onClickActions={() => router.push(ROUTES.OVERVIEW_ACTIONS)}
          onClickDestinations={() => router.push(ROUTES.OVERVIEW_DESTINATIONS)}
        />
        <DestinationTable destinations={destinations} tableMaxHeight='calc(100vh - 220px)' />
      </MainContent>
    </Container>
  );
};

export { OverviewFocusedDestinations };
