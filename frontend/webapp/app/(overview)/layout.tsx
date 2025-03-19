'use client';

import React, { useCallback, useMemo, type PropsWithChildren } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import styled from 'styled-components';
import { ENTITY_TYPES } from '@odigos/ui-kit/types';
import { useNamespace, useSSE, useTokenTracker } from '@/hooks';
import { OverviewHeader, OverviewModalsAndDrawers } from '@/components';
import { ErrorBoundary, FlexColumn, FlexRow } from '@odigos/ui-kit/components';
import { DataFlowActionsMenu, NAV_ICON_IDS, SideNav, ToastList } from '@odigos/ui-kit/containers';

const PageContent = styled(FlexColumn)`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors.primary};
  align-items: center;
`;

const ContentWithActions = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

const ContentUnderActions = styled(FlexRow)`
  align-items: flex-start;
  justify-content: space-between;
  padding-left: 12px;
  width: calc(100% - 12px);
`;

const getEntityType = (pathname: string) => {
  return pathname.includes(ROUTES.SOURCES)
    ? ENTITY_TYPES.SOURCE
    : pathname.includes(ROUTES.DESTINATIONS)
    ? ENTITY_TYPES.DESTINATION
    : pathname.includes(ROUTES.ACTIONS)
    ? ENTITY_TYPES.ACTION
    : pathname.includes(ROUTES.INSTRUMENTATION_RULES)
    ? ENTITY_TYPES.INSTRUMENTATION_RULE
    : undefined;
};

const getSelectedId = (pathname: string) => {
  return pathname.includes(ROUTES.OVERVIEW)
    ? NAV_ICON_IDS.OVERVIEW
    : pathname.includes(ROUTES.SOURCES)
    ? NAV_ICON_IDS.SOURCES
    : pathname.includes(ROUTES.DESTINATIONS)
    ? NAV_ICON_IDS.DESTINATIONS
    : pathname.includes(ROUTES.ACTIONS)
    ? NAV_ICON_IDS.ACTIONS
    : pathname.includes(ROUTES.INSTRUMENTATION_RULES)
    ? NAV_ICON_IDS.INSTRUMENTATION_RULES
    : undefined;
};

const routesMap = {
  [NAV_ICON_IDS.OVERVIEW]: ROUTES.OVERVIEW,
  [NAV_ICON_IDS.SOURCES]: ROUTES.SOURCES,
  [NAV_ICON_IDS.DESTINATIONS]: ROUTES.DESTINATIONS,
  [NAV_ICON_IDS.ACTIONS]: ROUTES.ACTIONS,
  [NAV_ICON_IDS.INSTRUMENTATION_RULES]: ROUTES.INSTRUMENTATION_RULES,
};

function OverviewLayout({ children }: PropsWithChildren) {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();

  const router = useRouter();
  const pathname = usePathname();
  const { namespaces } = useNamespace();

  const entityType = useMemo(() => getEntityType(pathname), [pathname]);
  const selectedId = useMemo(() => getSelectedId(pathname), [pathname]);

  const onClickId = useCallback(
    (navId: NAV_ICON_IDS) => {
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
          <DataFlowActionsMenu namespaces={namespaces} addEntity={entityType} />
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
