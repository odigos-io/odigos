import React from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import styled from 'styled-components';
import { useActionCRUD } from '@/hooks';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { FlexRow } from '@odigos/ui-components';
import { ActionTable, DataFlowActionsMenu, NAV_ICON_IDS, SideNav } from '@odigos/ui-containers';

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

const OverviewFocusedActions = () => {
  const router = useRouter();
  const { actions } = useActionCRUD();

  return (
    <Container>
      <DataFlowActionsMenu namespaces={[]} sources={[]} destinations={[]} actions={actions} instrumentationRules={[]} addEntity={ENTITY_TYPES.ACTION} />

      <MainContent>
        <SideNav
          defaultSelectedId={NAV_ICON_IDS.ACTIONS}
          onClickOverview={() => router.push(ROUTES.OVERVIEW)}
          onClickRules={() => router.push(ROUTES.OVERVIEW_INSTRUMENTATION_RULES)}
          onClickSources={() => router.push(ROUTES.OVERVIEW_SOURCES)}
          onClickActions={() => router.push(ROUTES.OVERVIEW_ACTIONS)}
          onClickDestinations={() => router.push(ROUTES.OVERVIEW_DESTINATIONS)}
        />
        <ActionTable actions={actions} tableMaxHeight='calc(100vh - 220px)' />
      </MainContent>
    </Container>
  );
};

export { OverviewFocusedActions };
