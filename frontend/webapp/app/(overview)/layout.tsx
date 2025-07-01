'use client';

import React, { useCallback, useMemo, type PropsWithChildren } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import styled from 'styled-components';
import { EntityTypes } from '@odigos/ui-kit/types';
import { useDataStreamsCRUD, useServiceMap, useSSE, useTokenTracker } from '@/hooks';
import { OverviewHeader, OverviewModalsAndDrawers } from '@/components';
import { ErrorBoundary, FlexColumn, FlexRow } from '@odigos/ui-kit/components';
import { DataFlowActionsMenu, NavIconIds, SideNav, ToastList } from '@odigos/ui-kit/containers';

const PageContent = styled(FlexColumn)`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors?.primary};
  align-items: center;
`;

const ContentWithActions = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

const ContentUnderActions = styled(FlexRow)`
  align-items: flex-start !important;
  justify-content: space-between;
  padding-left: 12px;
  width: calc(100% - 12px);
`;

const getEntityType = (pathname: string) => {
  return pathname.includes(ROUTES.SOURCES)
    ? EntityTypes.Source
    : pathname.includes(ROUTES.DESTINATIONS)
    ? EntityTypes.Destination
    : pathname.includes(ROUTES.ACTIONS)
    ? EntityTypes.Action
    : pathname.includes(ROUTES.INSTRUMENTATION_RULES)
    ? EntityTypes.InstrumentationRule
    : undefined;
};

const getSelectedId = (pathname: string) => {
  return pathname.includes(ROUTES.OVERVIEW)
    ? NavIconIds.Overview
    : pathname.includes(ROUTES.SOURCES)
    ? NavIconIds.Sources
    : pathname.includes(ROUTES.DESTINATIONS)
    ? NavIconIds.Destinations
    : pathname.includes(ROUTES.ACTIONS)
    ? NavIconIds.Actions
    : pathname.includes(ROUTES.INSTRUMENTATION_RULES)
    ? NavIconIds.InstrumentationRules
    : undefined;
};

const routesMap = {
  [NavIconIds.Overview]: ROUTES.OVERVIEW,
  [NavIconIds.Sources]: ROUTES.SOURCES,
  [NavIconIds.Destinations]: ROUTES.DESTINATIONS,
  [NavIconIds.Actions]: ROUTES.ACTIONS,
  [NavIconIds.InstrumentationRules]: ROUTES.INSTRUMENTATION_RULES,
};

function OverviewLayout({ children }: PropsWithChildren) {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();
  const { updateDataStream, deleteDataStream } = useDataStreamsCRUD();

  // TODO: move to releveant file after release of UI-Kit: https://github.com/odigos-io/ui-kit/pull/207
  useServiceMap();

  const router = useRouter();
  const pathname = usePathname();

  const entityType = useMemo(() => getEntityType(pathname), [pathname]);
  const selectedId = useMemo(() => getSelectedId(pathname), [pathname]);

  const onClickId = useCallback(
    (navId: NavIconIds) => {
      const route = routesMap[navId];
      if (route) router.push(route);
    },
    [router],
  );

  return (
    <ErrorBoundary>
      <PageContent>
        <OverviewHeader />

        <ContentWithActions>
          <DataFlowActionsMenu addEntity={entityType} onClickNewDataStream={() => router.push(ROUTES.CHOOSE_STREAM)} updateDataStream={updateDataStream} deleteDataStream={deleteDataStream} />
          <ContentUnderActions>
            <SideNav defaultSelectedId={selectedId} onClickId={onClickId} />
            {children}
          </ContentUnderActions>
        </ContentWithActions>

        <OverviewModalsAndDrawers />
        <ToastList />
      </PageContent>
    </ErrorBoundary>
  );
}

export default OverviewLayout;
