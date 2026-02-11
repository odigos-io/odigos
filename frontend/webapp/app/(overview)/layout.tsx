'use client';

import React, { type CSSProperties, useMemo, type PropsWithChildren } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import styled from 'styled-components';
import { EntityTypes } from '@odigos/ui-kit/types';
import { OdigosProvider } from '@odigos/ui-kit/contexts';
import { OverviewHeader, OverviewModalsAndDrawers } from '@/components';
import { DataFlowActionsMenu, ToastList } from '@odigos/ui-kit/containers';
import { ErrorBoundary, FlexColumn, IconsNav } from '@odigos/ui-kit/components';
import { useConfig, useDataStreamsCRUD, useSSE, useTokenTracker } from '@/hooks';
import { DATA_FLOW_HEIGHT, MENU_BAR_HEIGHT, ROUTES, getNavbarIcons } from '@/utils';

const PageContent = styled(FlexColumn)`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors?.primary};
  align-items: center;
`;

const ContentWithActions = styled.div<{ $height: CSSProperties['height'] }>`
  width: 100%;
  height: ${({ $height }) => $height};
  position: relative;
`;

const ContentUnderActions = styled.div`
  gap: 12px;
  display: flex;
  justify-content: space-between;
  padding: 0 12px;
  width: calc(100% - 24px);
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

function OverviewLayout({ children }: PropsWithChildren) {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();

  const { updateDataStream, deleteDataStream } = useDataStreamsCRUD();

  const router = useRouter();
  const pathname = usePathname();
  const { config } = useConfig();

  const entityType = useMemo(() => getEntityType(pathname), [pathname]);

  return (
    <ErrorBoundary>
      <OdigosProvider platformType={config?.platformType} tier={config?.tier} version={config?.odigosVersion || ''}>
        <PageContent>
          <OverviewHeader />

          <ContentWithActions $height={DATA_FLOW_HEIGHT}>
            {pathname.includes(ROUTES.SERVICE_MAP) ? (
              <div style={{ height: `${MENU_BAR_HEIGHT}px` }} />
            ) : (
              <DataFlowActionsMenu addEntity={entityType} onClickNewDataStream={() => router.push(ROUTES.CHOOSE_STREAM)} updateDataStream={updateDataStream} deleteDataStream={deleteDataStream} />
            )}

            <ContentUnderActions>
              <IconsNav orientation='vertical' mainIcons={getNavbarIcons(router, pathname)} subIcons={[]} />
              {children}
            </ContentUnderActions>
          </ContentWithActions>

          <OverviewModalsAndDrawers />
          <ToastList />
        </PageContent>
      </OdigosProvider>
    </ErrorBoundary>
  );
}

export default OverviewLayout;
