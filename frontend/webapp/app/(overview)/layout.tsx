'use client';

import React, { type CSSProperties, useMemo, type PropsWithChildren, useEffect } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import styled from 'styled-components';
import { OdigosProvider } from '@odigos/ui-kit/contexts';
import { useDarkMode, useModalStore } from '@odigos/ui-kit/store';
import { EntityTypes, OtherEntityTypes } from '@odigos/ui-kit/types';
import { OverviewHeader, OverviewModalsAndDrawers } from '@/components';
import { ErrorBoundary, FlexColumn, IconsNav } from '@odigos/ui-kit/components';
import { useConfig, useDataStreamsCRUD, useSSE, useTokenTracker } from '@/hooks';
import { DATA_FLOW_HEIGHT, MENU_BAR_HEIGHT, ROUTES, getNavbarIcons } from '@/utils';
import { DataFlowActionsMenu, DataStreamModal, ToastList } from '@odigos/ui-kit/containers';

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
  switch (pathname) {
    case ROUTES.SOURCES:
      return EntityTypes.Source;
    case ROUTES.DESTINATIONS:
      return EntityTypes.Destination;
    case ROUTES.ACTIONS:
      return EntityTypes.Action;
    case ROUTES.INSTRUMENTATION_RULES:
      return EntityTypes.InstrumentationRule;
    default:
      return undefined;
  }
};

function OverviewLayout({ children }: PropsWithChildren) {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();

  // TODO: remove this after migration to v2
  const { darkMode, setDarkMode } = useDarkMode();
  useEffect(() => {
    if (!darkMode) setDarkMode(true);
    document.body.style.backgroundColor = '#151618';
  }, []);

  const { setCurrentModal } = useModalStore();
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
            {pathname === ROUTES.SERVICE_MAP ? (
              <div style={{ height: `${MENU_BAR_HEIGHT}px` }} />
            ) : (
              <DataFlowActionsMenu
                addEntity={entityType}
                onClickNewDataStream={() => setCurrentModal(OtherEntityTypes.DataStream)}
                updateDataStream={updateDataStream}
                deleteDataStream={deleteDataStream}
              />
            )}

            <ContentUnderActions>
              <IconsNav orientation='vertical' mainIcons={getNavbarIcons(router, pathname)} subIcons={[]} />
              {children}
            </ContentUnderActions>
          </ContentWithActions>

          <DataStreamModal />
          <OverviewModalsAndDrawers />
          <ToastList />
        </PageContent>
      </OdigosProvider>
    </ErrorBoundary>
  );
}

export default OverviewLayout;
